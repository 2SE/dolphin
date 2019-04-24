package ws

import (
	"github.com/2se/dolphin/cluster"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/route"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	W       *WsServer
	Emmiter = event.NewEmitter(2)
	wg      = new(sync.WaitGroup)
)

const HeartBeatEquation = 3000

type SubscribeTopicer struct {
	*route.ClientComMeta
}

func (s *SubscribeTopicer) GetTopic() string {
	return s.Key + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func ParamBind(obj interface{}, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
		// Handle error
	}
	if err := schema.NewDecoder().Decode(obj, r.Form); err != nil {
		return err
		// Handle error
	}

	return nil
}

func (w *WsServer) handleClientData(conn *net.Conn, msg []byte) {
	var cli = &Client{conn: conn}

	// login & get client id
	if _, ok := w.Conns[conn]; !ok {
		cli.ID = verifyData(msg)
		w.AddCli <- cli
	}
	req := new(route.ClientComRequest)
	err := proto.Unmarshal(msg, req)
	if err != nil {
		log.Error("Ws: proto unmarsh msg error", err)
	}

	log.Println("unmashal msg", req)

	wssub := &SubscribeTopicer{req.Meta}
	if req.Meta.Key != "" {
		// subscribe
		subPid, event := Emmiter.Subscribe(wssub)
		log.Println("Ws: subscribe success", subPid)
		subscribe := &Subscribe{event, nil, subPid, req.Meta.Key, cli}
		w.Subscribe <- subscribe
	} else {
		// todo handle data
		//log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
		//todo
		mp := common.NewMethodPath(req.Meta.Revision, req.Meta.Resource, req.Meta.Action)
		pr, redirect, err := route.GetRouterInstance().RouteIn(mp, cli.ID)
		if err != nil {

		}
		if redirect {
			res, err := cluster.Emit(pr.PeerName(), pr.AppName(), req)
			if err != nil {
				// todo handle error
				log.Error("Ws: handleClientData redirect error ", err)
			}

			// todo handle redirect
			log.Println("Ws: got redirect request", res.String())

		} else {
			rep, err := route.GetRouterInstance().RouteOut(pr.AppName(), req)
			if err != nil {
				// todo handle error
				log.Error("Ws: handleClientData router out error ", err)
			}
			data, err := proto.Marshal(rep)
			if err != nil {
				// todo handle error
				log.Error("Ws: marshal ServerComResponse data error", err)
			}
			cli.Message = data
			w.SendMsg <- cli
		}
	}
}

// todo =====================================
func verifyData(msg []byte) string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

//func createTopic(revision, resource, act string) *TestTopic {
//	return &TestTopic{revision, resource, act}
//}
