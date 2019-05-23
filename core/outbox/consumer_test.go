package outbox

import (
	"context"
	_ "github.com/2se/dolphin/mock"
	"github.com/segmentio/kafka-go"
	"testing"
)

func TestConsumerEvent(t *testing.T) {
	cms = &Consumers{

		readers:     make([]*kafka.Reader, 1),
		topicOffset: make(map[string]int64),
		receive:     make(chan *kafka.Message, 64),
	}
	cms.readers[0] = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"192.168.10.189:9092"},
		Topic:    "dolphinhub",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	m, err := cms.readers[0].ReadMessage(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(m.Key), string(m.Value))
}
