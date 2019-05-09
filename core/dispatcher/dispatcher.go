package dispatcher

import (
	"github.com/2se/dolphin/common/hash"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"sync"
)

func New() core.HubDispatcher {
	dispatcher := &defaultDispatcher{
		hub:   make(map[string]map[uint32]*core.Subscription),
		queue: make(chan *core.KV, 128),
		quit:  make(chan bool),
	}

	go dispatcher.pipeline()
	return dispatcher
}

type defaultDispatcher struct {
	sync.RWMutex
	hub   map[string]map[uint32]*core.Subscription // ssid: hash -> core.Subscriber
	queue chan *core.KV
	quit  chan bool
}

func (dis *defaultDispatcher) Start() error {
	go dis.pipeline()
	return nil
}

func (dis *defaultDispatcher) Stop() error {
	close(dis.quit)
	return nil
}

func (dis *defaultDispatcher) Subscribe(ssid string, sub core.Subscriber) (*core.Subscription, error) {
	dis.Lock()
	defer dis.Unlock()

	key := hash.OfString(sub.GetID())
	if list, ok := dis.hub[ssid]; !ok {
		list = make(map[uint32]*core.Subscription)
		list[key] = &core.Subscription{Ssid: ssid, Sub: sub}
		dis.hub[ssid] = list
		return list[key], nil
	}

	if _, ok := dis.hub[ssid][key]; !ok {
		dis.hub[ssid][key] = &core.Subscription{Ssid: ssid, Sub: sub}
		return dis.hub[ssid][key], nil
	}

	return nil, nil
}

func (dis *defaultDispatcher) UnSubscribe(subscription *core.Subscription) {
	if subscription == nil || subscription.Sub == nil || len(subscription.Ssid) == 0 {
		return
	}

	dis.Lock()
	defer dis.Unlock()

	key := hash.OfString(subscription.Sub.GetID())
	if _, ok := dis.hub[subscription.Ssid][key]; ok {
		delete(dis.hub[subscription.Ssid], key)
		if len(dis.hub[subscription.Ssid]) == 0 {
			delete(dis.hub, subscription.Ssid)
		}
	}
}

func (dis *defaultDispatcher) Publish(kv *core.KV) {
	if kv != nil && kv.Key != nil && len(kv.Key) > 0 && kv.Val != nil && len(kv.Val) > 0 {
		dis.queue <- kv
	}
}

func (dis *defaultDispatcher) pipeline() {
	for {
		select {
		case data := <-dis.queue:
			dis.RLock()
			if subs, ok := dis.hub[string(data.Key)]; ok {
				for _, item := range subs {
					go func() {
						if _, err := item.Sub.Write(data.Val); err != nil {
							log.WithError(err).Error("publish: failed")
						}
					}()
				}
			}
			dis.RUnlock()
		case <-dis.quit:
			return
		}
	}
}

func (dis *defaultDispatcher) Dispatch(sess core.Session, req core.Request) {
	ccr := new(pb.ClientComRequest)
	err := proto.Unmarshal(req, ccr)
	if err != nil {
		log.Errorf("Ws: proto unmarsh msg error: %v", err)
		// TODO response back sess.write()
	}

	// TODO handle client id
	mp := core.NewMethodPath(ccr.MethodPath.Revision, ccr.MethodPath.Resource, ccr.MethodPath.Action)

	res, err := router.RouteIn(mp, sess.GetID(), ccr)
	if err != nil {
		// TODO handle error
		log.Error("ws: router in error", err)
		return
	}

	data, err := core.Marshal(res)
	if err != nil {
		// todo handle error
		log.Error("ws: marshal ServerComResponse data error", err)
	}

	if _, err = sess.Write(data); err != nil {
		log.WithError(err).Error("")
		return
	}

	if len(ccr.FrontEnd.Key) > 0 {
		dis.Subscribe(ccr.FrontEnd.Key, sess)
	}

	if len(ccr.FrontEnd.Key) > 0 {
		dis.UnSubscribe(&core.Subscription{Ssid: ccr.FrontEnd.Key, Sub: sess})
	}
}
