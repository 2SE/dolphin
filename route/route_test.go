package route

import (
	"fmt"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/config"
	"testing"
)

type mockRouter interface {
	listTopicPeers() map[string]*common.PeerRouters
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	route := InitRouter("node1", &config.RouteConfig{
		Recycle:   10,
		Threshold: 10,
		Timeout:   config.Duration{20},
	})
	route.Register([]common.MethodPath{
		common.NewMethodPath("1", "2", "3"),
		common.NewMethodPath("1", "3", "3"),
		common.NewMethodPath("1", "4", "3"),
	}, "50", "", "0.0.0.0:0000")

	route.Register([]common.MethodPath{
		common.NewMethodPath("1", "2", "3"),
		common.NewMethodPath("1", "3", "3"),
		common.NewMethodPath("1", "4", "3"),
	}, "50", "node2", "0.0.0.0:0000")
	for k, v := range route.(mockRouter).listTopicPeers() {
		fmt.Println("key:", k)
		fmt.Printf("peer:%s app:%s ", (*v)[0][0], (*v)[0][1])
	}
}
