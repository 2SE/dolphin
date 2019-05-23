package outbox

import (
	"context"
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

func TestConsumerEvent(t *testing.T) {
	ticker = mock.Ticker
	despatcher := dispatcher.New()
	despatcher.Start()
	defer despatcher.Stop()
	offsetRecoder.start()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"192.168.10.189:9092"},
		Topic:    "dolphinhub",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	m, err := reader.ReadMessage(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(m.Key), string(m.Value))
}
