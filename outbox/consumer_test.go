package outbox

import (
	"github.com/2se/dolphin/core/dispatcher"
	"github.com/2se/dolphin/mock"
	_ "github.com/2se/dolphin/mock"
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
	ticker = mock.Ticker
	despatcher := dispatcher.New()
	despatcher.Start()
	defer despatcher.Stop()
	offsetRecoder.start()
	//go func() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"www.rennbon.com:9092"},
		Topic:   "ttt1",
	})
	consumerTopic(reader, despatcher)
	//}()
}
