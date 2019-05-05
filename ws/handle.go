package ws

import (
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	Emmiter = event.NewEmitter(2)
	wg      = new(sync.WaitGroup)
	readWait int64
)

type SubscribeTopicer struct {
	*pb.ClientComRequest
}

func (s *SubscribeTopicer) GetTopic() string {
	// todo
	return s.Meta.Key
}

func (w *WsServer) handleWsConnection(conn *websocket.Conn) {
	if _, ok := w.id2Conns[conn]; ok {
		return
	}
	cli := &Client{conn: conn}
	w.AddCli <- cli
}

// 读websocket
func (w *WsServer) readLoop(cli *Client) {
	for {
		cli.conn.SetReadDeadline(time.Now().Add(Endpoint.cfg.ReadWait*time.Second))
		msgType, msgData, err := cli.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Println("ws: client closed")
			} else {
				log.Error("ws: readLoop got error ", err)
			}
			cli.closeChan <- true
		}

		msg := buildWSMessage(msgType, msgData)
		select {
		case cli.inChan <- msg:
		case <-cli.closeChan:
			goto CLOSED
		default:
			cli.inChan <- msg
		}
	}
CLOSED:
	w.DelCli <- cli
}

// 写websocket
func (w *WsServer) writeLoop(cli *Client) {
	var (
		message *WSMessage
		err     error
	)
	for {
		select {
		case message = <-cli.outChan:
			cli.conn.SetWriteDeadline(time.Now().Add(Endpoint.cfg.WriteWait*time.Second))
			if err = cli.conn.WriteMessage(message.MsgType, message.MsgData); err != nil {
				log.Error("ws: writeLoop error", err)
				cli.closeChan <- true
			}
		case <-cli.closeChan:
			goto CLOSED
		}
	}
CLOSED:
	w.DelCli <- cli
}

func buildWSMessage(msgType int, msgData []byte) (wsMessage *WSMessage) {
	return &WSMessage{
		MsgType: msgType,
		MsgData: msgData,
	}
}

func (w *WsServer) handleClientData(cli *Client) {
	for {
		select {
		case msg := <-cli.inChan:
			cli.conn.SetReadDeadline(time.Now().Add(Endpoint.cfg.ReadWait*time.Second))
			w.handleData(cli, msg)
		case <-cli.closeChan:
			goto CLOSED
		}
	}
CLOSED:
}

func (w *WsServer) handleData(cli *Client, msg *WSMessage) {
	switch msg.MsgType {
	case websocket.PingMessage:
		w.handlePing(cli)
	case websocket.BinaryMessage, websocket.TextMessage:
		req := new(pb.ClientComRequest)
		err := proto.Unmarshal(msg.MsgData, req)
		if err != nil {
			log.Error("Ws: proto unmarsh msg error", err)
		}

		log.Println("unmashal msg", req)
		if cli.ID == "" {
			// todo handle clientId
			cli.ID = verifyMsg(req.Meta.Signature)
		}

		mp := core.NewMethodPath(req.Meta.Revision, req.Meta.Resource, req.Meta.Action)
		res, err := router.RouteIn(mp, cli.ID, req)
		if err != nil {
			// TODO handle error
			log.Error("ws: router in error", err)
			return
		}
		data, err := proto.Marshal(res)
		if err != nil {
			// todo handle error
			log.Error("ws: marshal ServerComResponse data error", err)
		}
		cli.sendMsg(&WSMessage{websocket.TextMessage, data})
		// 订阅
		if req.Meta.Key != "" {
			wssub := &SubscribeTopicer{req}
			log.Printf("ws: client [%s] subscribe...", cli.ID)
			// subscribe
			subPid, event := Emmiter.Subscribe(wssub)
			log.Printf("ws: client [%s] subscribe success, subPid is [%s]", cli.ID, subPid)
			subscribe := &Subscribe{event, nil, subPid, req.Meta.Key, cli.conn}
			w.Subscribe <- subscribe
		}
	}
}

func (w *WsServer) handlePing(cli *Client) {
	msg := &WSMessage{websocket.PongMessage, nil}
	w.KeepAlive(cli)
	cli.sendMsg(msg)
}

func (cli *Client) sendMsg(msg *WSMessage) {
	select {
	case cli.outChan <- msg:
	case <-cli.closeChan:
		cli.conn.Close()
		return
	}
}

// todo mock get clientId
func verifyMsg(msg string) string{
	return uuid.New().String()
}
