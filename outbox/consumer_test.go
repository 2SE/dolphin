package outbox

import (
	"fmt"
	"github.com/2se/dolphin/event"
	"testing"
)

type TestTopic struct {
	Key string
}

func (t *TestTopic) GetTopic() string {
	return t.Key
}

var tpc = &TestTopic{"hashhash"}

func TestConsumerEvent(t *testing.T) {
	emitter := event.NewEmitter(2)
	_, eve := emitter.Subscribe(tpc)
	go func() {
		consumerTopic("www.rennbon.com:9092", "ttt1", "", 0, emitter)

	}()
	for c := range eve {
		fmt.Printf("key:%s val:%s\n", c.GetMetaData(), c.GetData())
	}
}
