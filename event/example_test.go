// Copyright (c) 2018 alpha. All rights reserved.
// Use of this source code is governed by a Apache License Version 2.0 found in the LICENSE file.

package event

import (
	"bytes"
	"fmt"
	"sync"
	"time"
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

var (
	fireEvent = &TestTopic{"v1", "user", "getUser"}
)

func Example_newEmitter() {

	emitter := NewEmitter(2)

	c := make(chan string, 1)
	go func() {
		identity, listener := emitter.Subscribe(fireEvent)
		if listener == nil {
			return
		}
		c <- identity

		for event := range listener {
			fmt.Printf("device 1: %s\n", event.GetData())
		}
	}()

	go func() {
		emitter.On(fireEvent, doEvent())
	}()

	go func() {
		_, listener := emitter.Once(fireEvent)
		if listener == nil {
			return
		}

		for event := range listener {
			fmt.Printf("device 2: %s\n", event.GetData())
		}
	}()

	times := 10
	counter := 0
	var wg sync.WaitGroup
	wg.Add(times)
	go func() {
		time.Sleep(time.Second)
		for {
			if counter >= times {
				break
			}

			if counter == 5 {
				emitter.UnSubscribe(<-c)
			}

			time.Sleep(2 * time.Millisecond)
			event := &GenericEvent{
				Topic: fireEvent,
				Data:  []byte(fmt.Sprintf("fire in the hole %d", counter)),
			}
			emitter.Emit(event)
			counter += 1
			wg.Done()
		}
	}()

	wg.Wait()
	time.Sleep(5 * time.Second)
}

func doEvent() Callback {
	ch := make(chan Event)
	go func() {
		for e := range ch {
			fmt.Printf("device 3: %s\n", e.GetData())
		}
	}()
	return func(event Event) {
		ch <- event
	}
}
