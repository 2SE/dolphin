package ws

import (
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	Emmiter = event.NewEmitter(2)
	wg      = new(sync.WaitGroup)
)

const HeartBeatEquation = 3000

type SubscribeTopicer struct {
	*pb.ClientComRequest
}

func (s *SubscribeTopicer) GetTopic() string {
	return s.Meta.Key + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

//func ParamBind(obj interface{}, r *http.Request) error {
//	if err := r.ParseForm(); err != nil {
//		return err
//		// Handle error
//	}
//	if err := schema.NewDecoder().Decode(obj, r.Form); err != nil {
//		return err
//		// Handle error
//	}
//
//	return nil
//}

func (w *WsServer) handleClientData(conn *net.Conn, msg []byte) {
	var cli = &Client{conn: conn}

	// login & get client id
	if _, ok := w.Conns[conn]; !ok {
		cli.ID = verifyData(msg)
		w.AddCli <- cli
	}
	req := new(pb.ClientComRequest)
	err := proto.Unmarshal(msg, req)
	if err != nil {
		log.Error("Ws: proto unmarsh msg error", err)
	}

	log.Println("unmashal msg", req)

	wssub := &SubscribeTopicer{req}
	// todo handle data
	//log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
	//todo
	mp := core.NewMethodPath(req.Meta.Revision, req.Meta.Resource, req.Meta.Action)
	res, err := router.RouteIn(mp, cli.ID, req)
	if err != nil {
		// TODO
		return
	}
	data, err := proto.Marshal(res)
	if err != nil {
		// todo handle error
		log.Error("Ws: marshal ServerComResponse data error", err)
	}
	cli.Message = data
	w.SendMsg <- cli

	if req.Meta.Key != "" {
		log.Printf("Ws: client [%s] subscribe...", cli.ID)
		// subscribe
		subPid, event := Emmiter.Subscribe(wssub)
		log.Printf("Ws: client [%s] subscribe success, subPid is [%s]", cli.ID, subPid)
		subscribe := &Subscribe{event, nil, subPid, req.Meta.Key, cli}
		w.Subscribe <- subscribe
	}
}

// todo =====================================
func verifyData(msg []byte) string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

//func createTopic(revision, resource, act string) *TestTopic {
//	return &TestTopic{revision, resource, act}
//}
