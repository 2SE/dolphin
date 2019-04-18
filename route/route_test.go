package route

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"sort"
	"testing"
)

func TestPeersRoute_Remove(t *testing.T) {
	psr := PeersRoute{{1, 2}, {2, 2}, {3, 2}, {4, 2}, {5, 2}}
	psr.removeByPeer(5)
	psr.removeByPeer(1)
	psr.removeByPeer(3)

	psrCmp := PeersRoute{{2, 2}, {4, 2}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Append(t *testing.T) {
	psr := PeersRoute{{1, 2}, {2, 2}}
	psr.append(PeerRoute{3, 2})
	psrCmp := PeersRoute{{1, 2}, {2, 2}, {3, 2}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Sort(t *testing.T) {
	psr := PeersRoute{{2, 2}, {3, 2}, {1, 2}, {4, 2}, {5, 2}}
	sort.Sort(psr)
	fmt.Println(psr)
}

func TestResourcesPool_RegiserSubResources(t *testing.T) {
	route := NewRoute(1)
	route.Register([]MethodPath{{1, 2, 3}, {1, 2, 4}, {2, 3, 4}}, 50, 0)
	for k, v := range route.listTopicPeers() {
		fmt.Println("key:", k)
		fmt.Printf("peer:%d app:%d ", (*v)[0][0], (*v)[0][1])
	}
}
