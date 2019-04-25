package outbox

import (
	"bytes"
	"context"
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/util"
	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
	"time"
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
	go offsetRecoder.start()
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
	var (
		logSend = offsetRecoder.sender()
		m       kafka.Message
		err     error
	)
	for {
		m, err = reader.ReadMessage(context.Background())
		if err != nil {
			log.WithFields(log.Fields{
				"outbox": "consumerTopic",
			}).Errorln(err.Error())
			continue
		}
		logSend <- &m
		pusher.Emit(&KV{
			Key: m.Key,
			Val: m.Value,
		})
	}
}

type OffsetRecoder struct {
	topicOffset map[string]int64
	receive     chan *kafka.Message
}

var (
	offsetRecoder = &OffsetRecoder{
		topicOffset: make(map[string]int64),
		receive:     make(chan *kafka.Message, 64),
	}
	tick = util.NewTimingWheel(time.Second, 2)
)

func (o *OffsetRecoder) sender() chan<- *kafka.Message {
	return o.receive
}
func (o *OffsetRecoder) start() {
	go func() {
		for msg := range o.receive {
			o.topicOffset[msg.Topic] = msg.Offset
		}
	}()
	go o.logWrite()
}

func (o *OffsetRecoder) logWrite() {
	buff := bytes.NewBuffer(nil)
	for {
		select {
		case <-tick.After(time.Second):
			for k, v := range o.topicOffset {
				buff.WriteString(k)
				buff.WriteRune(':')
				buff.WriteString(strconv.FormatInt(v, 10))
				buff.WriteString("\n")
			}
			ioutil.WriteFile("./kfk.log", buff.Bytes(), 0644)
			buff.Reset()
		}
	}
}
