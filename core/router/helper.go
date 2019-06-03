package router

import (
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
)

func Register(mps []core.MethodPather, pr core.PeerRouter, address string) error {
	return r.Register(mps, pr, address)
}

func RouteIn(mp core.MethodPather, id string, request proto.Message) (response proto.Message, err error) {
	return r.RouteIn(mp, id, request)
}
