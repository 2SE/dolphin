package ws

import (
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	Emmiter = event.NewEmitter(2)
	wg      = new(sync.WaitGroup)
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
		msgType, msgData, err := cli.conn.ReadMessage()
		if err != nil {
			log.Error("ws: readLoop got error", err)
			//w.DelCli <- cli
			cli.closeChan <- true
		}
		msg := buildWSMessage(msgType, msgData)
		select {
		case cli.inChan <- msg:
			log.Println("333333333333")
		case <-cli.closeChan:
			log.Println("4444444444444")
			goto CLOSED
		default:
			cli.inChan <- msg
		}
	}
CLOSED:
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
			if err = cli.conn.WriteMessage(message.MsgType, message.MsgData); err != nil {
				log.Error("ws: writeLoop error", err)
				cli.closeChan <- true
			}
		case <-cli.closeChan:
			goto CLOSED
		}
	}
CLOSED:
}

func buildWSMessage(msgType int, msgData []byte) (wsMessage *WSMessage) {
	return &WSMessage{
		MsgType: msgType,
		MsgData: msgData,
	}
}

func (w *WsServer) handleClientData(cli *Client) {
	cli.conn.SetReadDeadline(time.Now().Add(pongWait))
	for {
		select {
		case msg := <-cli.inChan:
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

		wssub := &SubscribeTopicer{req}
		// todo handle data
		//log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
		// todo
		mp := core.NewMethodPath(req.Meta.Revision, req.Meta.Resource, req.Meta.Action)
		res, err := router.RouteIn(mp, cli.ID, req)
		if err != nil {
			// TODO
			log.Error("ws: router in error", err)
			return
		}
		data, err := proto.Marshal(res)
		if err != nil {
			// todo handle error
			log.Error("Ws: marshal ServerComResponse data error", err)
		}
		cli.outChan <- &WSMessage{websocket.BinaryMessage, data}
		// 订阅
		if req.Meta.Key != "" {
			log.Printf("Ws: client [%s] subscribe...", cli.ID)
			// subscribe
			subPid, event := Emmiter.Subscribe(wssub)
			log.Printf("Ws: client [%s] subscribe success, subPid is [%s]", cli.ID, subPid)
			subscribe := &Subscribe{event, nil, subPid, req.Meta.Key, cli.conn}
			w.Subscribe <- subscribe
		}
	default:
	}
}

func (w *WsServer) handlePing(cli *Client) {
	msg := &WSMessage{websocket.PongMessage, nil}
	w.m.Unlock()
	defer w.m.Lock()
	w.Clients[w.id2Conns[cli.conn]].Timestamp = time.Now().UnixNano() / 1e6
	cli.sendMsg(msg)
}

func (cli *Client) sendMsg(msg *WSMessage) {
	select {
	case cli.outChan <- msg:
		log.Println("ws: send message success")
	case <-cli.closeChan:
		log.Errorf("ws: Client [%s] connection had been closed", cli.ID)
	default:
		log.Println("ws: send message but buffer is full")
	}
}

func (w *WsServer) handlePush(cli *Client) {
	cli.conn.SetWriteDeadline(time.Now().Add(writeWait))
	for {
		select {
		case msg := <-cli.outChan:
			cli.sendMsg(msg)
		case <- cli.closeChan:
			goto CLOSED
		}
	}
	CLOSED:
}