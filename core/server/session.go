package server

import (
	"github.com/2se/dolphin/common/security"
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"runtime"
	"sync"
	"time"
)

var _ io.WriteCloser = &session{}

const userEmpty = ""

// session
type session struct {
	opt  *Opt
	conn *ws.Conn
	//用户id
	userId string
	send   chan []byte
	quit   chan bool
	luid   security.ID
	guid   string
	sync.Mutex
}

func NewSession(conn *ws.Conn, opt *Opt) (io.WriteCloser, error) {
	if conn == nil {
		return nil, NilConnErr
	}

	if opt.Dispatcher == nil {
		return nil, NilDispatcherErr
	}

	sess := &session{
		conn: conn,
		send: make(chan []byte, opt.SessionQueueSize),
		quit: make(chan bool),
		opt:  opt,
		luid: security.NewID(),
	}
	sess.guid = sess.luid.Unique(uint64(security.GetHardware()), opt.IDSalt)

	go sess.readLoop()
	go sess.writeLoop()
	return sess, nil
}

func (sess *session) GetID() string {
	return sess.guid
}
func (sess *session) LoggedIn() bool {
	sess.Lock()
	defer sess.Unlock()
	return sess.userId != userEmpty
}
func (sess *session) GetUserId() string {
	return sess.userId
}
func (sess *session) SetUserId(userId string) {
	sess.Lock()
	defer sess.Unlock()
	sess.userId = userId
}

func (sess *session) Send(message proto.Message) (err error) {
	var data []byte
	data, err = core.Marshal(message)
	if err != nil {
		return
	}

	_, err = sess.Write(data)
	return
}

func (sess *session) Write(data []byte) (int, error) {
	return sess.queueOut(data)
}

func (sess *session) Close() error {
	close(sess.quit)
	return nil
}

func (sess *session) closeWs() {
	sess.conn.Close()
}

func (sess *session) dispatch(data []byte) {
	if sess.opt.Dispatcher != nil {
		sess.opt.Dispatcher.Dispatch(sess, data)
	}
}

func (sess *session) queueOut(data []byte) (n int, err error) {
	defer func() {
		if e := recover(); e != nil {
			n = 0
			if e, ok := e.(runtime.Error); ok && e.Error() == "send on closed channel" {
				err = WriteInClosedWriterErr
			} else {
				err = e.(error)
			}
		}
	}()

	n = len(data)
	select {
	case sess.send <- data:
	case <-time.After(sess.opt.QueueOutTimeout):
		log.Warnf("queueOut: timeout")
		n = 0
		err = TimeoutErr
	}
	return
}

func (sess *session) readLoop() {
	defer sess.closeWs()
	sess.conn.SetReadDeadline(calcTimeout(sess.opt.pongWait))
	// receive pong message from client, then reset readLoop deadline
	sess.conn.SetPongHandler(func(string) error {
		return sess.conn.SetReadDeadline(calcTimeout(sess.opt.pongWait))
	})

	var (
		data []byte
		err  error
	)

	for {
		_, data, err = sess.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure, ws.CloseAbnormalClosure) {
				log.WithError(err).Error("websocket: unexpected error in read loop")
			}
			return
		}
		sess.dispatch(data)
	}
}

func (sess *session) writeLoop() {
	defer sess.closeWs()
	go sess.ping()
	for {
		select {
		case data := <-sess.send:
			if err := sess.write(data); err != nil {
				if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure, ws.CloseAbnormalClosure) {
					log.WithError(err).Error("websocket: unexpected error in write loop")
				}
				return
			}
		case <-sess.quit:
			close(sess.send)
			if len(sess.send) > 0 {
				left := len(sess.send)
				var data []byte
				for i := 0; i < left; i++ {
					data = <-sess.send
					if err := sess.write(data); err != nil {
						if ws.IsUnexpectedCloseError(err,
							ws.CloseGoingAway,
							ws.CloseNormalClosure,
							ws.CloseAbnormalClosure) {
							log.WithError(err).Error("websocket: unexpected error in write loop")
						}
						break
					}
				}
			}
			return
		}
	}

}

func (sess *session) write(data []byte) error {
	sess.conn.SetWriteDeadline(calcTimeout(sess.opt.WriteWait))
	return sess.conn.WriteMessage(ws.BinaryMessage, data)
}

func (sess *session) ping() {
	hasErr := make(chan error)
	action := func() {
		err := sess.conn.WriteControl(ws.BinaryMessage, []byte{}, calcTimeout(sess.opt.WriteWait))
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.WithError(err).Error("")
			}
			hasErr <- err
			return
		}

		hasErr <- nil
	}

	for {
		sess.opt.Ticker.AfterFunc(sess.opt.pingPeriod, action)
		if err := <-hasErr; err != nil {
			return
		}
	}
}

func calcTimeout(d time.Duration) time.Time {
	return time.Now().Add(d)
}
