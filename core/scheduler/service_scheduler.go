package scheduler

import (
	"context"
	"fmt"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/pb"
	"github.com/sirupsen/logrus"
	"net/http"

	"google.golang.org/grpc"
	"net"
)

type SchedulerService struct {
}

var ss = new(SchedulerService)

func (s *SchedulerService) Request(ctx context.Context, req *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	res, err := router.RouteIn(core.NewMethodPath(req.MethodPath.Revision, req.MethodPath.Resource, req.MethodPath.Action), req.Id, req)
	if err != nil {
		if res == nil {
			res = &pb.ServerComResponse{
				Code: http.StatusInternalServerError,
				Text: err.Error(),
			}
		}
	}
	return res.(*pb.ServerComResponse), err
}

func SchedulerStart(address string) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		panic(fmt.Errorf("tpc listen err:%v ", err))
	}
	defer l.Close()
	svc := grpc.NewServer()
	pb.RegisterAppServeServer(svc, ss)
	logrus.Info("SchedulerService ready to start and listen on ", address)
	if err := svc.Serve(l); err != nil {
		panic(fmt.Errorf("failed to serve: %v", err))
	}
}
