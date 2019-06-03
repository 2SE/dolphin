package router

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
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
	s.conns[address] = conn
	s.clients[address] = appCli
	log.Infof("grpc client %s startup", address)
	return nil
}

func (s *resourcesPool) RemoveClient(address string) {
	delete(s.clients, address)
	delete(s.conns, address)
	log.Infof("grpc client %s shutdown", address)
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
	ch := make(chan struct{}, 1)
	for {
		ticker.AfterFunc(s.recycle, func() {
			for k, v := range s.connErr {
				if v > s.threshold {
					//todo removeByApp
					s.UnRegisterApp(s.addrPR[k])
					log.WithFields(log.Fields{
						logFieldKey: "errRecovery",
					}).Tracef("appclient %s was removed\n", k)
					s.connErrRemove(k)
				} else {
					s.connErrClear(k)
				}
			}
			ch <- struct{}{}
		})
		<-ch
	}
}
func (s *resourcesPool) healthCheck() {
	ch := make(chan struct{}, 1)
	for {
		ticker.AfterFunc(s.heartBeat, func() {
			for k, v := range s.conns {
				if v.GetState() == connectivity.TransientFailure {
					s.connErrInc(k)
				}
			}
			ch <- struct{}{}
		})
		<-ch
	}
}

func (s *resourcesPool) connErrInc(key string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.connErr[key]++
}

func (s *resourcesPool) connErrClear(key string) {
	s.m.Lock()
	defer s.m.Unlock()
	s.connErr[key] = 0
}

func (s *resourcesPool) connErrRemove(key string) {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.connErr, key)
}
