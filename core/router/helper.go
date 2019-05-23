package router

import (
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
)

func Register(mps []core.MethodPath, pr core.PeerRouter, address string) error {
	return r.Register(mps, pr, address)
}

func RouteIn(mp core.MethodPath, id string, request proto.Message) (response proto.Message, err error) {
	return r.RouteIn(mp, id, request)
}
