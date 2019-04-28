package router

import (
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
	"testing"
	"time"
)

type mockCluster struct {
}

func (*mockCluster) Name() string {
	return "mock"
}
func (*mockCluster) SetRouter(core.Router) {

}
func (*mockCluster) Notify(core.PeerRouter, ...core.MethodPath) {

}
func (*mockCluster) Request(core.PeerRouter, proto.Message) (proto.Message, error) {
	return nil, nil
}

type mockRouter interface {
	ListTopicPeers() map[string]*core.PeerRouters
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	//run dolphin/cmd/example/server.go fisrt
	mockPeer := &mockCluster{}
	route := Init(mockPeer, &config.RouteConfig{
		Recycle:   config.Duration{time.Second * 60},
		Threshold: 10,
		Timeout:   config.Duration{time.Second * 2},
	})
	err := route.Register([]core.MethodPath{
		core.NewMethodPath("1", "2", "3"),
		core.NewMethodPath("1", "3", "3"),
		core.NewMethodPath("1", "4", "3"),
	},
		core.NewPeerRouter("", "app1"),
		"127.0.0.1:16012")
	if err != nil {
		fmt.Printf("register1 failed err:%s \n", err)
	} else {
		fmt.Println("register1 successed")
	}
	err = route.Register([]core.MethodPath{
		core.NewMethodPath("1", "2", "3"),
		core.NewMethodPath("1", "3", "3"),
		core.NewMethodPath("1", "4", "3"),
	},
		core.NewPeerRouter("", "app2"),
		"127.0.0.1:16013")
	if err != nil {
		fmt.Printf("register2 failed err:%s \n", err)
	} else {
		fmt.Println("register1 successed")
	}
	for k, v := range route.(mockRouter).ListTopicPeers() {
		fmt.Println("key:", k)
		for _, vi := range *v {
			fmt.Printf("peerRouter:%s \n", vi.String())
		}
	}
}
