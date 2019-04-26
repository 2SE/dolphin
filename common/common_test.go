package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"

	"sort"
	"testing"
)

func TestPeersRoute_Remove(t *testing.T) {
	psr := PeerRouters{peerRoute{"1", "2"}, peerRoute{"2", "2"}, peerRoute{"3", "2"}, peerRoute{"4", "2"}, peerRoute{"5", "2"}}
	psr.RemoveByPeer("5")
	psr.RemoveByPeer("1")
	psr.RemoveByPeer("3")

	psrCmp := PeerRouters{peerRoute{"2", "2"}, peerRoute{"4", "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Append(t *testing.T) {
	psr := PeerRouters{peerRoute{"1", "2"}, peerRoute{"2", "2"}}
	psr.Append(peerRoute{"3", "2"})
	psrCmp := PeerRouters{peerRoute{"1", "2"}, peerRoute{"2", "2"}, peerRoute{"3", "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Sort(t *testing.T) {
	psr := PeerRouters{peerRoute{"5", "2"}, peerRoute{"1", "2"}, peerRoute{"3", "2"}, peerRoute{"4", "2"}, peerRoute{"2", "2"}}
	sort.Sort(psr)
	fmt.Println(psr)
}
