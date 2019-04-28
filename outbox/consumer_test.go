package outbox

import (
	"fmt"
	"github.com/2se/dolphin/event"
	"github.com/segmentio/kafka-go"
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
	offsetRecoder.start()
	_, eve := emitter.Subscribe(tpc)
	go func() {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{"127.0.0.1:9092"},
			Topic:   "ttt1",
		})
		consumerTopic(reader, emitter)
	}()
	for c := range eve {
		fmt.Printf("key:%s val:%s\n", c.GetMetaData(), c.GetData())
		break
	}
}
