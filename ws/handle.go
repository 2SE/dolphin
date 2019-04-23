package ws

import (
	"bytes"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/eventbus"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

var (
	W *WsServer
	Emmiter = event.NewEmitter(2)
	wg = new(sync.WaitGroup)
)


type TestTopic struct {
	Version  string
	Resource string
	Action   string
}

func (s *TestTopic) GetTopic() string {
	buff := bytes.NewBuffer(nil)
	buff.WriteString(s.Version)
	buff.WriteString("_")
	buff.WriteString(s.Action)
	buff.WriteString("_")
	buff.WriteString(s.Resource)
	return buff.String()
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

func (w *WsServer)handleClientData(cli *Client, msg []byte) {
	// todo 确定topic
	var metaData eventbus.ClientComMeta
	err := proto.Unmarshal(msg, &metaData)
	if err != nil {
		log.Error("Ws: proto unmarsh msg error", err)
	}

	log.Println("unmashal msg", metaData)
	wsTopic := createTopic(metaData.Revision, metaData.Resource, metaData.Action)
	if metaData.Key != "" {
		// subscribe
		subPid, event := Emmiter.Subscribe(wsTopic)
		log.Println("Ws: subscribe success", subPid)
		subscribe := &Subscribe{event, subPid, metaData.Key}
		cli.Subscribes = append(cli.Subscribes, *subscribe)
		w.Subscribe <- cli
	} else {
		// todo handle data
		//log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
	}
}

func createTopic(revision, resource, act string) *TestTopic {
	return &TestTopic{revision, resource, act}
}
