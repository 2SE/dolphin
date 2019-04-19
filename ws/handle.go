package ws

import (
	"bytes"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/eventbus"
	log "github.com/sirupsen/logrus"
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

func (s *TestTopic) GetTopic() []byte {
	buff := bytes.NewBuffer(nil)
	buff.WriteString(s.Version)
	buff.WriteString("_")
	buff.WriteString(s.Action)
	buff.WriteString("_")
	buff.WriteString(s.Resource)
	return buff.Bytes()
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
//	return nil
//}

func (w *WsServer)handleClientData(cli *Client, metaData *eventbus.ClientComMeta) {
	// todo 确定topic
	wsTopic := createTopic(metaData.Revision, metaData.Resource, metaData.Action)
	if metaData.Key != "" {
		// subscribe
		subPid, event := Emmiter.Subscribe(wsTopic)
		subscribe := &Subscribe{event, subPid, metaData.Key}
		cli.Subscribes = append(cli.Subscribes, *subscribe)
		w.Subscribe <- cli
	} else {
		// todo handle data
		log.Printf("Ws: got data, client is %s, data is %v", cli.ID, metaData)
	}
}

func createTopic(revision, resource, act string) *TestTopic {
	return &TestTopic{revision, resource, act}
}
