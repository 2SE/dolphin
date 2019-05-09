package outbox

import (
	"bytes"
	"context"
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	tw "github.com/RussellLuo/timingwheel"
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

var ticker *tw.TimingWheel

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
func ConsumersInit(cnfs []*config.KafkaConfig, pusher core.Hub, twl *tw.TimingWheel) {
	ticker = twl
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

func consumerTopic(reader *kafka.Reader, pusher core.Hub) error {
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
		pusher.Publish(&core.KV{
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
	ch := make(chan struct{})
	for {
		ticker.AfterFunc(time.Second, func() {
			ch <- struct{}{}
			for k, v := range o.topicOffset {
				buff.WriteString(k)
				buff.WriteRune(':')
				buff.WriteString(strconv.FormatInt(v, 10))
				buff.WriteString("\n")
			}
			ioutil.WriteFile("./kfk.log", buff.Bytes(), 0644)
			buff.Reset()
		})
		<-ch
	}
}
