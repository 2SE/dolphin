package outbox

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/event"
	"github.com/segmentio/kafka-go"
)

func getKafkaReader(kafkaURL, topic, groupID string, partition int) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaURL},
		GroupID:   groupID,
		Topic:     topic,
		Partition: partition,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})
}

type KV struct {
	Key []byte
	Val []byte
}

func (kv *KV) GetTopic() string {
	return string(kv.Key)
}
func (kv *KV) GetMetaData() []byte {
	return kv.Key
}
func (kv *KV) GetData() []byte {
	return kv.Val
}

func consumerTopic(kafkaURL, topic, groupID string, partition int, pusher event.Emitter) {
	reader := getKafkaReader(kafkaURL, topic, groupID, partition)
	defer reader.Close()
	fmt.Println("start consuming ... !!")
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("err :", err)
		}
		pusher.Emit(&KV{
			Key: m.Key,
			Val: m.Value,
		})
		fmt.Printf("message at topic:%v partition:%v offset:%v	%s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
	}
}
