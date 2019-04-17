// Copyright (c) 2018 alpha. All rights reserved.
// Use of this source code is governed by a Apache License Version 2.0 found in the LICENSE file.

package event

import (
	"fmt"
	"sync"
	"testing"
)

const (
	fireTimes = 10
)

func TestOn(t *testing.T) {
	emitter := NewEmitter(2)
	wg := new(sync.WaitGroup)
	wg.Add(fireTimes)

	emitter.On(fireEvent, response(wg, t))
	event := &GenericEvent{
		Topic: &Topic{
			Version: 0x01,
			Source:  0x02,
			Action:  0x03,
		},
	}

	for i := 0; i < fireTimes; i++ {
		event.Data = []byte(fmt.Sprintf("fire in the hole %d", i+1))
		emitter.Emit(event)
	}

	t.Log("event emitted.")
	wg.Wait()
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
