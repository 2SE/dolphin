package ws

import (
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/eventbus"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
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
	eventbus.ClientComMeta
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

func (w *WsServer) handleClientData(cli *Client, msg []byte) {
	// todo 确定topic
	var metaData eventbus.ClientComMeta
	err := proto.Unmarshal(msg, &metaData)
	if err != nil {
		log.Error("Ws: proto unmarsh msg error", err)
	}

	log.Println("unmashal msg", metaData)

	wssub := &SubscribeTopicer{metaData}
	if metaData.Key != "" {
		// subscribe
		subPid, event := Emmiter.Subscribe(wssub)
		log.Println("Ws: subscribe success", subPid)
		subscribe := &Subscribe{event, subPid, metaData.Key, cli.ID, cli.conn}
		cli.Subscribes = append(cli.Subscribes, *subscribe)
	} else {
		// todo handle data
		//log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
	}
}

//func createTopic(revision, resource, act string) *TestTopic {
//	return &TestTopic{revision, resource, act}
//}
