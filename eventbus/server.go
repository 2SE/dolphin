package eventbus

import (
	"context"
	"github.com/2se/dolphin/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

var (
	server *eventBusServer
)

type eventBusServer struct {
	address string
}

func Init(cnf *config.EventBusConfig) error {
	server = &eventBusServer{
		address: cnf.Listen,
	}
	return nil
}

func Start() {
	l, err := net.Listen("tcp", server.address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	comunicator := &AppComunicator{}
	server := grpc.NewServer()
	RegisterBusServer(server, comunicator)

	go func() {
		if err := server.Serve(l); err != nil {
			log.Fatalf("failed to serve: %v\n", err)
		}
	}()
}

func Shutdown() {

}

type AppComunicator struct {
}

// APP向event bug 注册
func (a *AppComunicator) Register(context.Context, *AppInfo) (*RegReply, error) {
	return nil, nil
}

// 双向发送数据
func (a *AppComunicator) Pipe(stream Bus_PipeServer) error {
	stream.Recv()
	return nil
}
