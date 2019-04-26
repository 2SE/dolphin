package router

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/common/timer"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

var (
	tick = timer.NewTimingWheel(time.Second, 30)
)

func (s *resourcesPool) TryAddClient(address string) error {
	if _, ok := s.clients[address]; ok {
		return ErrClientExists
	}
	ctx1, _ := context.WithTimeout(context.Background(), s.timeout)
	//there can interceptor the data,if you need to change the retry mechanism, modify the backoff
	conn, err := grpc.DialContext(ctx1, address, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	appCli := pb.NewAppServeClient(conn)
	s.clients[address] = appCli
	return ErrGprcServerConnFailed
}

func (s *resourcesPool) RemoveClient(address string) {
	delete(s.clients, address)
}

func (s *resourcesPool) callAppAction(address string, request proto.Message) (*pb.ServerComResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	//重试的机制 https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md
	resp, err := s.clients[address].Request(ctx, request.(*pb.ClientComRequest))
	if err != nil {
		st := status.Convert(err)
		//todo log
		//DeadlineExceeded 是否需要
		if st.Code() != codes.DeadlineExceeded {
			s.connErr[address]++
		}
		log.WithFields(log.Fields{
			logFieldKey: "callAppAction",
			"address":   address,
			"errCode":   st.Code(),
		}).Errorln(err.Error())
	}

	return resp, err
}

func (s *resourcesPool) errRecovery() {
	for {
		select {
		case <-tick.After(s.recycle):
			for k, v := range s.connErr {
				if v > s.threshold {
					//todo removeByApp
					s.UnRegisterApp(s.addrPR[k])
					log.WithFields(log.Fields{
						logFieldKey: "errRecovery",
					}).Tracef("appclient %s was removed\n", k)
				}
				s.connErr[k] = 0
			}
		}
	}
}
