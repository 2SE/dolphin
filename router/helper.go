package router

import (
	"github.com/2se/dolphin/common"
	"github.com/golang/protobuf/proto"
)

func Register(mps []common.MethodPath, pr common.PeerRouter, address string) error {
	return r.Register(mps, pr, address)
}

func RouteIn(mp common.MethodPath, id string) (pr common.PeerRouter, redirect bool, err error) {
	return r.RouteIn(mp, id)
}

func RouteOut(pr common.PeerRouter, request proto.Message) (response proto.Message, err error) {
	return r.RouteOut(pr, request)
}
