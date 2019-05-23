package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"

	"sort"
	"testing"
)

func TestPeersRoute_Remove(t *testing.T) {
	psr := PeerRouters{
		&peerRoute{peer: "1", app: "2"},
		&peerRoute{peer: "2", app: "2"},
		&peerRoute{peer: "3", app: "2"},
		&peerRoute{peer: "4", app: "2"},
		&peerRoute{peer: "5", app: "2"},
	}
	psr.RemoveByPeer("5")
	psr.RemoveByPeer("1")
	psr.RemoveByPeer("3")

	psrCmp := PeerRouters{
		&peerRoute{peer: "2", app: "2"},
		&peerRoute{peer: "4", app: "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Append(t *testing.T) {
	psr := PeerRouters{
		&peerRoute{peer: "1", app: "2"},
		&peerRoute{peer: "2", app: "2"}}
	psr.Append(&peerRoute{peer: "3", app: "2"})
	psrCmp := PeerRouters{
		&peerRoute{peer: "1", app: "2"},
		&peerRoute{peer: "2", app: "2"},
		&peerRoute{peer: "3", app: "2"}}
	assert.Equal(t, psr, psrCmp)
}

func TestPeersRoute_Sort(t *testing.T) {
	psr := PeerRouters{
		&peerRoute{peer: "5", app: "2"},
		&peerRoute{peer: "1", app: "2"},
		&peerRoute{peer: "3", app: "2"},
		&peerRoute{peer: "4", app: "2"},
		&peerRoute{peer: "2", app: "2"}}
	sort.Sort(psr)
	fmt.Println(psr)
}
