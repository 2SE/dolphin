package route

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/util"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

var (
	tick = util.NewTimingWheel(time.Second, 30)
)

func (p *resourcesPool) TryAddClient(address string) error {
	if _, ok := p.clients[address]; ok {
		return ErrClientExists
	}
	ctx1, _ := context.WithTimeout(context.Background(), p.timeout)
	//there can interceptor the data,if you need to change the retry mechanism, modify the backoff
	conn, err := grpc.DialContext(ctx1, address, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	appCli := NewAppServeClient(conn)
	p.clients[address] = appCli
	return ErrGprcServerConnFailed
}

func (p *resourcesPool) RemoveClient(address string) {
	delete(p.clients, address)
}

func (p *resourcesPool) callAppAction(address string, request proto.Message) (*ServerComResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	//重试的机制 https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md
	resp, err := p.clients[address].Request(ctx, request.(*ClientComRequest))
	if err != nil {
		st := status.Convert(err)
		//todo log
		//DeadlineExceeded 是否需要
		if st.Code() != codes.DeadlineExceeded {
			p.connErr[address]++
		}
		log.WithFields(log.Fields{
			logFieldKey: "callAppAction",
			"address":   address,
			"errCode":   st.Code(),
		}).Errorln(err.Error())
	}

	return resp, err
}

func (p *resourcesPool) errRecovery() {
	for {
		select {
		case <-tick.After(p.recycle):
			for k, v := range p.connErr {
				if v > p.threshold {
					//todo removeByApp
					p.UnRegisterApp(p.addrApp[k])
					log.WithFields(log.Fields{
						logFieldKey: "errRecovery",
					}).Tracef("appclient %s was removed\n", k)
				}
				p.connErr[k] = 0
			}
		}
	}
}
