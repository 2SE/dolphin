package ws

import (
	"github.com/2se/dolphin/event"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Client websocket client info
type Client struct {
	ID        string            // client id
	conn      *websocket.Conn   // websocket connection
	Timestamp int64             // last heartbeat timestamp：millisecond
	inChan    chan *WSMessage // channel for data in
	outChan   chan *WSMessage // channel for data out
	closeChan chan bool
	pushers   []*Subscribe
}

// PushJob struct for push task
type Subscribe struct {
	eventIn          <-chan event.Event
	eventUnSubscribe chan bool
	SubPid           string
	SubKey           string
	conn             *websocket.Conn
}

// WsServer websocket connection pool
type WsServer struct {
	key2Pusher map[string]*Client
	Clients    map[string]*Client
	id2Conns   map[*websocket.Conn]string
	AddCli     chan *Client
	DelCli     chan *Client
	Subscribe  chan *Subscribe
	m          *sync.RWMutex
}

type Param struct {
	ClientID string `schema:"client_id"`
}

// NewWsServer Create an instance of the websocket server
func NewWsServer() *WsServer {
	return &WsServer{
		key2Pusher: make(map[string]*Client),
		Clients:    make(map[string]*Client),
		id2Conns:   make(map[*websocket.Conn]string),
		AddCli:     make(chan *Client),
		DelCli:     make(chan *Client),
		Subscribe:  make(chan *Subscribe),
		m:          new(sync.RWMutex),
	}
}

func (w *WsServer) Start() {
	for {
		select {
		case c := <-w.AddCli:
			w.addClient(c)
		case c := <-w.DelCli:
			w.delClient(c)
		case s := <-w.Subscribe:
			w.sbuscribe(s)
		}
	}
}

// addClient add a ws client to session
func (w *WsServer) addClient(c *Client){
	log.Print("ws: add a new client...")
	if c.conn == nil {
		log.Println("ws: could not add the client cause client info error")
		return
	}
	w.m.Lock()
	defer w.m.Unlock()
	c.Timestamp = time.Now().UnixNano() / 1e6
	clientId, ok := w.id2Conns[c.conn]
	if ok {
		log.Printf("ws: add a new client, but Client [%s] already exists", clientId)
		return
	}
	clientId = uuid.New().String()
	c.inChan = make(chan *WSMessage)
	c.outChan = make(chan *WSMessage)
	c.closeChan = make(chan bool)
	w.Clients[clientId] = c
	w.id2Conns[c.conn] = clientId

	log.Println("ws: add a new client ", clientId)
	go w.readLoop(c)
	go w.writeLoop(c)
	go w.handleClientData(c)
	go w.handlePush(c)
}

// delClient delete the ws client from session by clientId
func (w *WsServer) delClient(c *Client) {
	w.m.Lock()
	defer func() {
		w.m.Unlock()
		c.conn.Close()
	}()

	// cancel subscribe
	w.unSubscribe(c)

	if c.conn != nil {
		delete(w.id2Conns, c.conn)
	}

	if c.ID != "" {
		delete(w.Clients, c.ID)
		log.Printf("Ws: client [%s] has been deleted", c.ID)
	}
}

// subscribe ws client subscribe
func (w *WsServer) sbuscribe(s *Subscribe) {
	w.m.Lock()
	defer w.m.Unlock()

	if _, ok := w.id2Conns[s.conn]; !ok {
		return
	}
	cli := w.Clients[w.id2Conns[s.conn]]

	if _, ok := w.key2Pusher[s.SubKey]; !ok {
		s.eventUnSubscribe = make(chan bool)
		cli.pushers = append(cli.pushers, s)
		w.key2Pusher[s.SubKey] = cli
	}
	for {
		select {
		case pmsg := <-s.eventIn:
			data := &WSMessage{websocket.BinaryMessage, pmsg.GetData()}
			cli.outChan <- data
		case <-s.eventUnSubscribe:
			goto CLOSED
		}
	}
	CLOSED:
	//w.Clients[w.id2Conns[s.conn]].pushers = subscribes
}

// unSubscribe ws client unSubscribe
func (w *WsServer) unSubscribe(c *Client) {
	w.m.Lock()
	defer w.m.Unlock()

	for _, p := range c.pushers {
		Emmiter.UnSubscribe(p.SubPid)
		p.eventUnSubscribe <- true
		delete(w.key2Pusher, p.SubKey)
	}
}

// HandleHeartBeat handle websocket heart beat
func (w *WsServer) HandleHeartBeat(equation_ms int64) {
	clients := w.Clients
	for _, v := range clients {
		if time.Now().UnixNano()/1e6-v.Timestamp > equation_ms{
			w.DelCli <- v
			w.unSubscribe(v)
		}
	}
}
