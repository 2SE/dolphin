package router

import (
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	"testing"
)

type mockRouter interface {
	ListTopicPeers() map[string]*core.PeerRouters
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	route := Init("node1", &config.RouteConfig{
		Recycle:   10,
		Threshold: 10,
		Timeout:   config.Duration{20},
	})
	route.Register([]core.MethodPath{
		core.NewMethodPath("1", "2", "3"),
		core.NewMethodPath("1", "3", "3"),
		core.NewMethodPath("1", "4", "3"),
	}, "50", "", "0.0.0.0:0000")

	route.Register([]core.MethodPath{
		core.NewMethodPath("1", "2", "3"),
		core.NewMethodPath("1", "3", "3"),
		core.NewMethodPath("1", "4", "3"),
	}, "50", "node2", "0.0.0.0:0000")
	for k, v := range route.(mockRouter).ListTopicPeers() {
		fmt.Println("key:", k)
		fmt.Printf("peer:%s app:%s ", (*v)[0][0], (*v)[0][1])
	}
}
