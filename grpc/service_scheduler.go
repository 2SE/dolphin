package grpc

import (
	"context"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/pb"
	"time"
)

type SchedulerService struct {
}

func (s *SchedulerService) Request(ctx context.Context, req *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*30))
	defer cancel()
	res, err := router.RouteIn(core.NewMethodPath(req.Meta.Revision, req.Meta.Resource, req.Meta.Action), req.Id, req)
	return res.(*pb.ServerComResponse), err
}
