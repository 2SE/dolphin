// Copyright (c) 2018 alpha. All rights reserved.
// Use of this source code is governed by a Apache License Version 2.0 found in the LICENSE file.

package event

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	fireTimes = 11
)

func TestOn(t *testing.T) {
	emitter := NewEmitter(2)
	//wg := new(sync.WaitGroup)
	//wg.Add(fireTimes)
	//response channel
	//emitter.On(fireEvent, response(wg, t))
	id, eve := emitter.Subscribe(fireEvent)

	go func() {
		for i := 0; i < 100000000000; i++ {
			time.Sleep(time.Millisecond * 100)
			event := &GenericEvent{
				Topic: fireEvent,
			}
			event.Data = []byte(fmt.Sprintf("fire in the hole %d", i+1))
			emitter.Emit(event)
		}
	}()
	go func() {
		for c := range eve {
			fmt.Println("print ", c)
			//wg.Done()
		}
	}()
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				emitter.UnSubscribe(id)
			}
		}
	}()
	t.Log("event emitted.")
	//wg.Wait()
	//close(ch)
	select {}
	t.Log("Testing finished.")
}

func response(wg *sync.WaitGroup, t *testing.T) Callback {
	ch := make(chan Event)
	go func() {
		for e := range ch {
			t.Logf("go data: %s\n", e.GetData())
		}
	}()

	return func(event Event) {
		ch <- event
		wg.Done()
	}
}

func TestAA(t *testing.T) {
	fmt.Println(2342)

	fmt.Println()
}
