package ws

import (
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/route"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

// Client websocket client info
type Client struct {
	ID       string
	conn     *net.Conn
	wsServer *WsServer
	//Subscribes       []Subscribe
	//eventUnSubscribe chan bool
	Timestamp int64
	Message   []byte
}

type Subscribe struct {
	eventIn          <-chan event.Event
	eventUnSubscribe chan bool
	SubPid           string
	SubKey           string
	cli              *Client
}

// WsServer the struct of the websocket server
type WsServer struct {
	Subscribs   map[*Client][]Subscribe
	Clients     map[string]*Client
	Conns       map[*net.Conn]string
	Pushers     map[string]*Client
	AddCli      chan *Client
	DelCli      chan *Client
	Subscribe   chan *Subscribe
	UnSubscribe chan *Client
	SendMsg     chan *Client
	m           *sync.RWMutex
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
		Conns:       make(map[*net.Conn]string),
		Pushers:     make(map[string]*Client),
		AddCli:      make(chan *Client),
		DelCli:      make(chan *Client),
		SendMsg:     make(chan *Client),
		Subscribe:   make(chan *Subscribe),
		UnSubscribe: make(chan *Client),
		m:           new(sync.RWMutex),
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
		case c := <-w.SendMsg:
			w.SendMessage(c)
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
func (w *WsServer) SendMessage(cli *Client) {
	var data route.ClientComMeta
	if _, ok := w.Conns[cli.conn]; ok {
		if err := wsutil.WriteServerMessage(*cli.conn, ws.OpBinary, cli.Message); err != nil {
			// todo handle error and resend
			log.Errorf("Ws: failed to send msg to client %s", data.Key)
		}
	} else {
		// todo handle error
		log.Errorf("Ws: client not found, user_id: %s", data.Key)
	}
}

// delClient delete the ws client from session by clientId
func (w *WsServer) delClient(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()

	// cancel subscribe
	w.UnSubscribe <- c

	if c.conn != nil {
		delete(w.Conns, c.conn)
	}

	if c.ID != "" {
		delete(w.Clients, c.ID)
		log.Printf("Ws: client [%s] has been deleted", c.ID)
	}
}

// addClient add a ws client to session
func (w *WsServer) addClient(c *Client) {
	if c.ID == "" || c.conn == nil {
		log.Println("Ws: could not add the client cause client info error")
		return
	}
	w.m.Lock()
	defer w.m.Unlock()
	c.Timestamp = time.Now().UnixNano() / 1e6
	if _, ok := w.Conns[c.conn]; !ok {
		w.Conns[c.conn] = c.ID
	}

	if _, ok := w.Clients[c.ID]; !ok {
		w.Clients[c.ID] = c
	}
	log.Printf("Ws: add client %s to session successfully ", c.ID)
}

// subscribe ws client subscribe
func (w *WsServer) sbuscribe(s *Subscribe) {
	// add client to session
	w.AddCli <- s.cli

	w.m.Lock()
	defer w.m.Unlock()
	subscribes := w.Subscribs[s.cli]
	if subscribes == nil {
		subscribes = []Subscribe{}
	}
	subscribes = append(subscribes, *s)
	w.Pushers[s.SubKey] = s.cli
}

// unSubscribe ws client unSubscribe
func (w *WsServer) unSubscribe(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()

	if subKeys, ok := w.Subscribs[c]; ok {
		for _, s := range subKeys {
			Emmiter.UnSubscribe(s.SubPid)
			s.eventUnSubscribe <- true
			delete(w.Pushers, s.SubKey)
		}
		delete(w.Subscribs, c)
	}
}

// HandleHeartBeat handle websocket heart beat
func (w *WsServer) HandleHeartBeat(equation_ms int64) {
	clients := w.Clients
	for _, v := range clients {
		if time.Now().UnixNano()/1e6-v.Timestamp > equation_ms {
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
