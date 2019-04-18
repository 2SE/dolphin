package eventbus

import (
	"context"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/route"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

var (
	server *eventBusServer
	router route.Router
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
func (a *AppComunicator) Register(ctx context.Context, request *RegisterRequest) (*RegisterResponse, error) {
	mps := make([]route.MethodPath, 0, len(request.MethodPath))
	for _, v := range request.MethodPath {
		mps = append(mps, route.MethodPath{v[0], v[1], v[2]})
	}
	router.Register(mps, request.AppName, "")
	return &RegisterResponse{
		Accepted: true,
		Code:     1,
	}, nil
}

//
//// 双向发送数据
//func (a *AppComunicator) Pipe(stream Bus_PipeServer) error {
//	stream.Recv()
//	return nil
//}
