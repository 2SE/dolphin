package outbox

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/event"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

/*
func getKafkaReader(kafkaURL, topic, groupID string, partition int) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaURL},
		GroupID:   groupID,
		Topic:     topic,
		Partition: partition,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})
}*/

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
func ConsumersInit(cnfs []*config.KafkaConfig, pusher event.Emitter) {
	for _, v := range cnfs {
		go func(cnf *config.KafkaConfig) {
			reader := kafka.NewReader(kafka.ReaderConfig{
				Brokers:   cnf.Brokers,
				GroupID:   cnf.GroupID,
				Topic:     cnf.Topic,
				Partition: cnf.Partition,
				MinBytes:  cnf.MinBytes,
				MaxBytes:  cnf.MaxBytes, // 10MB
				MaxWait:   cnf.MaxWait.Duration,
			})
			err := reader.SetOffset(cnf.Offset)
			if err != nil {
				panic(fmt.Errorf("kafka topic %s consumer err:%s", cnf.Topic, err.Error()))
			}
			consumerTopic(reader, pusher)
		}(v)
	}
}

func consumerTopic(reader *kafka.Reader, pusher event.Emitter) error {
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.WithFields(log.Fields{
				"outbox": "consumerTopic",
			}).Errorln(err.Error())
			continue
		}
		pusher.Emit(&KV{
			Key: m.Key,
			Val: m.Value,
		})
	}
}
