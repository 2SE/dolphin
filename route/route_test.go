package route

import (
	"fmt"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/config"
	"sort"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func TestPeersRoute_Remove(t *testing.T) {
	psr := PeersRoute{{"1", "2"}, {"2", "2"}, {"3", "2"}, {"4", "2"}, {"5", "2"}}
	psr.removeByPeer("5")
	psr.removeByPeer("1")
	psr.removeByPeer("3")

	psrCmp := PeersRoute{{"2", "2"}, {"4", "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Append(t *testing.T) {
	psr := PeersRoute{{"1", "2"}, {"2", "2"}}
	psr.append(PeerRoute{"3", "2"})
	psrCmp := PeersRoute{{"1", "2"}, {"2", "2"}, {"3", "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Sort(t *testing.T) {
	psr := PeersRoute{{"2", "2"}, {"3", "2"}, {"1", "2"}, {"4", "2"}, {"5", "2"}}
	sort.Sort(psr)
	fmt.Println(psr)
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	route := InitRoute("node1", &config.RouteConfig{
		Recycle:   10,
		Threshold: 10,
		Timeout:   60,
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
	for k, v := range route.listTopicPeers() {
		fmt.Println("key:", k)
		fmt.Printf("peer:%s app:%s ", (*v)[0][0], (*v)[0][1])
	}
}
