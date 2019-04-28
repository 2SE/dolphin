package router

import (
	"fmt"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/mock"
	"testing"
)

type mockRouter interface {
	ListTopicPeers() map[string]*core.PeerRouters
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	//run dolphin/cmd/example/server.go fisrt
	err := mock.MockRoute.Register([]core.MethodPath{
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
	err = mock.MockRoute.Register([]core.MethodPath{
		core.NewMethodPath("1", "2", "3"),
		core.NewMethodPath("1", "3", "3"),
		core.NewMethodPath("2", "4", "3"),
	},
		core.NewPeerRouter("", "app2"),
		"127.0.0.1:16013")
	if err != nil {
		fmt.Printf("register2 failed err:%s \n", err)
	} else {
		fmt.Println("register1 successed")
	}
	for k, v := range mock.MockRoute.(mockRouter).ListTopicPeers() {
		fmt.Println("key:", k)
		for _, vi := range *v {
			fmt.Printf("peerRouter:%s \n", vi.String())
		}
	}
}
