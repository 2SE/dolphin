package event

import (
	"unsafe"
)

//Type Event type. you can define custom event type
//Please don`t overide the following interal event type:
// subscribe Type = "subscribe"
// unSubscribe Type = "unSubscribe"
// unSubscribeAll Type = "unSubscribeAll"
type Type byte

type Topic [3]byte

type Topicer interface {
	GetTopic() Topic
	GetKey() byte
}

//Event ...
type Event interface {
	GetTopic() Topic
	GetMetaData() []byte
	GetData() []byte
}

//GenericEvent ...
type GenericEvent struct {
	Topic Topic
	Meta  []byte
	Data  []byte
}

//GetType ...
func (e *GenericEvent) GetTopic() Topic {
	return e.Topic
}

//GetMetaData ...
func (e *GenericEvent) GetMetaData() []byte {
	return e.Meta
}

//GetData ...
func (e *GenericEvent) GetData() []byte {
	return e.Data
}

//Callback ...
type Callback func(event Event)

//Emitter ...
type Emitter interface {
	On(topicer Topic, callback Callback) (identity uintptr)
	Once(topicer Topic) (identity uintptr, event <-chan Event)
	Subscribe(topicer Topic) (identity uintptr, event <-chan Event)
	UnSubscribe(identity ...uintptr)
	UnSubscribeAll()
	Emit(event ...Event)
}

//NewEmitter ...
// one event broadcast to multiple subscribers
// when unsubscribe event, the channel sending event will automate be closed
func NewEmitter(eventBufSize int) Emitter {
	emitter := new(eventEmitter)
	emitter.hub = make(map[Topic]*broadcaster)
	emitter.eventListener = make(chan Event)
	emitter.observer = make(chan subscriber)
	if eventBufSize <= 0 {
		eventBufSize = 256
	}
	emitter.eventBufSize = eventBufSize
	go emitter.dispatch()
	return emitter
}

type eventEmitter struct {
	hub           map[Topic]*broadcaster
	eventListener chan Event
	observer      chan subscriber
	eventBufSize  int
}

func (e *eventEmitter) dispatch() {
	for {
		select {
		case event := <-e.eventListener:
			if b, ok := e.hub[event.GetTopic()]; ok {
				b.broadcast(event)
			}
		case suber := <-e.observer:
			switch suber.subscribeAction {
			case subscribe:
				b, ok := e.hub[suber.topic]
				if !ok {
					b = new(broadcaster)
					b.subers = make(map[uintptr]subscriber)
					b.pipeline = make(chan Event)
					b.observer = make(chan subscriber)
					b.eventBufSize = e.eventBufSize
					b.emitter = e
					go b.start()
					e.hub[suber.topic] = b
				}
				b.dealRegister(suber)
			case unSubscribe:
				e.unSubscribe(suber)
			case unSubscribeAll:
				e.unSubscribe(suber)
			}
		}
	}
}

func (e *eventEmitter) unSubscribe(suber subscriber) {
	for _, b := range e.hub {
		go b.dealRegister(suber)
	}
}

func (e *eventEmitter) On(topic Topic, callback Callback) (identity uintptr) {
	identity, _ = e.doSubscribe(topic, fireAllways, callback)
	return
}

func (e *eventEmitter) Once(topic Topic) (uintptr, <-chan Event) {
	return e.doSubscribe(topic, fireOnce, nil)
}

func (e *eventEmitter) Subscribe(topic Topic) (uintptr, <-chan Event) {
	return e.doSubscribe(topic, fireAllways, nil)
}

func (e *eventEmitter) doSubscribe(topic Topic, subType subscribeType, callback Callback) (uintptr, <-chan Event) {
	//identity := uuid.New()
	response := make(chan chan Event)
	ptr := uintptr(unsafe.Pointer(&callback))
	e.observer <- subscriber{
		identity:        ptr,
		callback:        callback,
		subscribeAction: subscribe,
		topic:           topic,
		subscribeType:   subType,
		response:        response,
	}

	event, ok := <-response
	if !ok {
		return 0, nil
	}

	if callback != nil {
		close(event)
		return ptr, nil
	}

	return ptr, event
}

func (e *eventEmitter) UnSubscribe(identities ...uintptr) {
	for _, identity := range identities {
		e.observer <- subscriber{
			identity:        identity,
			subscribeAction: unSubscribe,
		}
	}
}

func (e *eventEmitter) UnSubscribeAll() {
	e.observer <- subscriber{
		subscribeAction: unSubscribeAll,
	}
}

func (e *eventEmitter) Emit(events ...Event) {
	for _, event := range events {
		e.eventListener <- event
	}
}

type subscribeType byte

const (
	fireOnce       subscribeType = 0x01
	fireAllways    subscribeType = 0x02
	subscribe      Type          = 0x11
	unSubscribe    Type          = 0x12
	unSubscribeAll Type          = 0x13
)

type subscriber struct {
	identity        uintptr //string
	callback        Callback
	topic           Topic
	subscribeType   subscribeType
	subscribeAction Type
	response        chan chan Event
	event           chan Event
}

type broadcaster struct {
	subers       map[uintptr]subscriber
	pipeline     chan Event
	observer     chan subscriber
	eventBufSize int
	emitter      *eventEmitter
}

func (b *broadcaster) start() {
	for {
		select {
		case e := <-b.pipeline:
			b.processEvent(e)
		case suber := <-b.observer:
			switch suber.subscribeAction {
			case subscribe:
				b.processSubscribe(suber)
			case unSubscribe:
				b.processUnSubscribe(suber)
			case unSubscribeAll:
				b.processUnSubscribeAll(suber)
			}
		}
	}
}

func (b *broadcaster) processEvent(event Event) {
	for identity, suber := range b.subers {
		if suber.subscribeType == fireOnce {
			delete(b.subers, identity)
			b.checkEmpty(suber.topic)
		}

		if suber.callback != nil {
			go suber.callback(event)
		} else {
			suber.event <- event
		}

		if suber.subscribeType == fireOnce {
			close(suber.event)
		}
	}
}

func (b *broadcaster) processSubscribe(suber subscriber) {
	if suber.subscribeType == fireAllways {
		suber.event = make(chan Event, b.eventBufSize)
	} else {
		suber.event = make(chan Event)
	}
	suber.response <- suber.event
	close(suber.response)
	b.subers[suber.identity] = suber
}

func (b *broadcaster) processUnSubscribe(suber subscriber) {
	if s, ok := b.subers[suber.identity]; ok {
		delete(b.subers, suber.identity)
		close(s.event)
		b.checkEmpty(s.topic)
	}
}

func (b *broadcaster) processUnSubscribeAll(suber subscriber) {
	for id, suber := range b.subers {
		if suber.subscribeType == fireAllways && suber.callback == nil {
			close(suber.event)
		}
		delete(b.subers, id)
		b.checkEmpty(suber.topic)
	}
}

func (b *broadcaster) checkEmpty(topic Topic) {
	if len(b.subers) == 0 {
		//clean the parent event type
		//will error?
		delete(b.emitter.hub, topic)
	}
}

func (b *broadcaster) broadcast(event Event) {
	b.pipeline <- event
}

func (b *broadcaster) dealRegister(suber subscriber) {
	b.observer <- suber
}
