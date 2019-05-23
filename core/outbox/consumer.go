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
	"sync"
	"time"
)

var cms *Consumers

type Consumers struct {
	ticker  *tw.TimingWheel
	readers []*kafka.Reader
	hub     core.Hub

	topicOffset map[string]int64
	receive     chan *kafka.Message
}

func InitConsumers(cnfs []*config.KafkaConfig, pusher core.Hub, twl *tw.TimingWheel) {
	cms = &Consumers{
		ticker:      twl,
		hub:         pusher,
		readers:     make([]*kafka.Reader, 0, len(cnfs)),
		topicOffset: make(map[string]int64),
		receive:     make(chan *kafka.Message, 64),
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(cnfs))
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
			cms.readers = append(cms.readers, reader)
			wg.Done()
		}(v)
	}
	wg.Wait()
	cms.asyncOffsetLogger()
	cms.consumerTopic()
}

func (c *Consumers) consumerTopic() {
	for _, reader := range c.readers {
		go func(reader *kafka.Reader) {
			var (
				m   kafka.Message
				err error
			)
			for {
				m, err = reader.ReadMessage(context.Background())
				if err != nil {
					log.WithFields(log.Fields{
						"outbox": "consumerTopic",
						"topic":  reader.Config().Topic,
					}).Errorln(err.Error())
					continue
				}
				c.receive <- &m
				c.hub.Publish(&core.KV{
					Key: m.Key,
					Val: m.Value,
				})
			}
		}(reader)
	}
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

func (o *Consumers) asyncOffsetLogger() {
	go func() {
		for msg := range o.receive {
			o.topicOffset[msg.Topic] = msg.Offset
		}
	}()
	go o.logWrite()
}

func (o *Consumers) logWrite() {
	buff := bytes.NewBuffer(nil)
	ch := make(chan struct{})
	for {
		o.ticker.AfterFunc(time.Second, func() {
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
