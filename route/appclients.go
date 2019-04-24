package route

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/util"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"log"
	"time"
)

var (
	tick = util.NewTimingWheel(time.Second, 60)
)

func (p *resourcesPool) TryAddClient(address string) error {
	if _, ok := p.clients[address]; ok {
		return ErrClientExists
	}
	ctx1, _ := context.WithTimeout(context.Background(), time.Second*60)
	//there can interceptor the data
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	//重试的机制 https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md
	resp, err := p.clients[address].Request(ctx, request.(*ClientComRequest))
	if err != nil {
		st := status.Convert(err)
		//todo log
		p.connErr[address]++
		fmt.Printf("the app server %s return an err:%s at %s \n", address, st.Message(), time.Now().String())
	}

	return resp, err
}
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	log.Printf("before invoker. method: %+v, request:%+v", method, req)
	err := invoker(ctx, method, req, reply, cc, opts...)
	log.Printf("after invoker. reply: %+v", reply)
	return err
}
func (p *resourcesPool) errRecovery() {
	for {
		select {
		case <-tick.After(20):
			for k, v := range p.connErr {
				if v > 5 {
					//todo removeByApp
					p.UnRegisterApp(p.addrApp[k])
				}
				p.connErr[k] = 0
			}
		}
	}
}
