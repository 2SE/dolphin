package ws

import (
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/route"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

// Client websocket client info
type Client struct {
	ID               string
	conn             *net.Conn
	wsServer         *WsServer
	Subscribes       []Subscribe
	eventUnSubscribe chan bool
	Timestamp        int64
}

type Subscribe struct {
	eventIn  <-chan event.Event
	SubPid   string
	SubKey   string
	ClientID string
	conn     *net.Conn
}

// WsServer the struct of the websocket server
type WsServer struct {
	Subscribs   map[*Client][]Subscribe
	Clients     map[string]*Client
	AddCli      chan *Client
	DelCli      chan *Client
	Subscribe   chan *Subscribe
	UnSubscribe chan *Client
	Message     chan []byte
	m           sync.RWMutex
}

//type Message struct {
//	ClientID string `json:"client_id"`
//	Message  string `json:"message"`
//}

type Param struct {
	ClientID string `schema:"client_id"`
}

// NewWsServer Create an instance of the websocket server
func NewWsServer() *WsServer {
	return &WsServer{
		Subscribs:   make(map[*Client][]Subscribe),
		Clients:     make(map[string]*Client),
		AddCli:      make(chan *Client),
		DelCli:      make(chan *Client),
		Message:     make(chan []byte),
		Subscribe:   make(chan *Subscribe),
		UnSubscribe: make(chan *Client),
	}
}

type WS interface {
	Start()
	SendMessage(id, msg string)
	delClient(c *Client)
	addClient(c *Client)
	sbuscribe(c *Client)
}

func (w *WsServer) Start() {
	for {
		select {
		case msg := <-w.Message:
			w.SendMessage(msg)
		case c := <-w.AddCli:
			w.addClient(c)
		case c := <-w.DelCli:
			w.delClient(c)
		case s := <-w.Subscribe:
			w.sbuscribe(s)
		case c := <-w.UnSubscribe:
			w.unSubscribe(c)
		}
	}
}

// SendMessage send message to the ws client by clientId
func (w *WsServer) SendMessage(msg []byte) {
	var data route.ClientComMeta
	proto.Unmarshal(msg, &data)
	if _, ok := w.Clients[data.Key]; ok {
		if err := wsutil.WriteServerMessage(*w.Clients[data.Key].conn, ws.OpText, msg); err != nil {
			log.Errorf("Ws: failed to send msg to client %s", data.Key)
		}
	} else {
		log.Errorf("Ws: client not found, user_id: %s", data.Key)
	}
}

// delClient delete the ws client from session by clientId
func (w *WsServer) delClient(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()
	if _, ok := w.Clients[c.ID]; ok {
		delete(w.Clients, c.ID)
		w.unSubscribe(c)
	}
	log.Printf("Ws: client %s has been deleted", c.ID)
}

// addClient add a ws client to session
func (w *WsServer) addClient(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()
	if _, ok := w.Clients[c.ID]; !ok {
		w.Clients[c.ID] = c
		log.Printf("Ws: add client %s to session successfully ", c.ID)
	}
}

// subscribe ws client subscribe
func (w *WsServer) sbuscribe(s *Subscribe) {
	w.m.Lock()
	defer w.m.Unlock()
	cli := &Client{ID: s.ClientID, conn: s.conn, Timestamp: time.Now().UnixNano()/1e6}
	w.AddCli <- cli
	subscribes := w.Subscribs[cli]
	if subscribes == nil {
		subscribes = []Subscribe{}
	}
	subscribes = append(subscribes, *s)
}

// unSubscribe ws client unSubscribe
func (w *WsServer) unSubscribe(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()

	if client, ok := w.Clients[c.ID]; ok {
		if subKeys, ok := w.Subscribs[client]; ok {
			for _, s := range subKeys {
				Emmiter.UnSubscribe(s.SubPid)
			}
			delete(w.Subscribs, client)
		}
	}

	w.Clients[c.ID].eventUnSubscribe = make(chan bool)
	w.Clients[c.ID].eventUnSubscribe <- true
}

// HandleHeartBeat handle websocket heart beat
func (w *WsServer) HandleHeartBeat(equation_ms int64) {
	clients := w.Clients
	for _, v := range clients {
		if time.Now().UnixNano()/1e6 - v.Timestamp > equation_ms {
			w.DelCli <- v
			w.UnSubscribe <- v
		}
	}
}

//func (w *WsServer) handleSubscribe() {
//	for c, v := range w.Subscribs {
//		go func() {
//			for {
//				flag := true
//				select {
//				case <-c.eventUnSubscribe:
//					close(c.eventUnSubscribe)
//					flag = false
//					break
//				}
//				for _, s := range v {
//					if flag == false {
//						break
//					}
//					data, ok := <-s.eventIn
//					if !ok {
//						log.Printf("Ws: channel closed")
//						break
//					}
//					metaData := data.GetMetaData()
//					w.Message <- metaData
//				}
//				//data, ok := <-v.eventIn
//				//if !ok {
//				//	log.Printf("Ws: channel closed")
//				//	break
//				//}
//				//metaData := data.GetMetaData()
//				//w.Message <- metaData
//			}
//		}()
//	}
//}
