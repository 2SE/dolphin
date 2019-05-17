package dispatcher

import (
	"errors"
	"fmt"
	"github.com/2se/dolphin/common/hash"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net/http"
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
	//part 1
	if err != nil {
		err = fmt.Errorf("Ws: proto unmarsh msg error: %v", err)
		response(sess, http.StatusBadRequest, err)
		return
	}
	//part1.5 check
	if ccr.FrontEnd == nil {
		err = errors.New("Request parameter validation failed")
		response(sess, http.StatusBadRequest, err)
		return
	}

	//part2
	bucket := fmt.Sprintf("%v%v%v", ccr.Qid, ccr.FrontEnd.Uuid)
	limited, _, err := limiter.RateLimit(bucket, 1)
	if err != nil {
		err = fmt.Errorf("limiter store error: %v", err)
		response(sess, http.StatusInternalServerError, err)
		return
	}

	if limited {
		err = errors.New("The request exceeded the current limit")
		response(sess, http.StatusForbidden, err)
		return
	}

	// TODO handle client id
	mp := core.NewMethodPath(ccr.MethodPath.Revision, ccr.MethodPath.Resource, ccr.MethodPath.Action)

	res, err := router.RouteIn(mp, sess.GetID(), ccr)
	if err != nil {
		err = fmt.Errorf("ws: router in error", err)
		response(sess, http.StatusInternalServerError, err)
		return
	}
	data, err := core.Marshal(res)
	if err != nil {
		// todo handle error
		err = fmt.Errorf("ws: marshal ServerComResponse data error", err)
		response(sess, http.StatusInternalServerError, err)
		return
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

func response(sess core.Session, code uint32, err error) {
	log.Error(err)
	res := &pb.ServerComResponse{
		Code: code,
		Text: err.Error(),
	}
	data, err := core.Marshal(res)
	if err != nil {
		// todo handle error
		log.Error("ws: marshal ServerComResponse data error", err)
	}
	if _, err = sess.Write(data); err != nil {
		log.WithError(err).Error("")
	}
}
