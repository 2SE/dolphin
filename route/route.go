package route

import (
	"bytes"
	"errors"
	"sync"
)

// fisrt byte  : version
// second byte : resource
// third byte  : action
type MethodPath [3]byte

type Router interface {
	RouteIn(mp MethodPath) (interface{}, error)
	RouteOut(mp MethodPath) (interface{}, error)
	//注册单个app上所有资源,peer 为 0 是默认本地
	Register(mps []MethodPath, app, peer byte)
	//注销app下所有
	UnRegisterApp(appKey byte)
	//注销peer下所有
	UnRegisterPeer(peer byte)

	//for unit test
	listTopicPeers() map[MethodPath]*PeersRoute
}

// 初始化本地route
// peer 本地cluster 编号
func NewRoute(peer byte) Router {
	route := &resourcesPool{
		curPeer:    peer,
		topicPeers: make(map[MethodPath]*PeersRoute),
	}
	return route
}

type resourcesPool struct {
	curPeer    byte
	topicPeers map[MethodPath]*PeersRoute
	m          sync.RWMutex
}

type PeerRoute [2]byte

type PeersRoute []PeerRoute

func (s PeersRoute) Len() int      { return len(s) }
func (s PeersRoute) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PeersRoute) Less(i, j int) bool {
	return s[i][0] < s[j][0]
}
func (s *PeersRoute) append(pr PeerRoute) {
	*s = append(*s, pr)
}
func (s *PeersRoute) removeByPeer(peer byte) {
	for k, v := range *s {
		if v[0] == peer {
			if k == 0 {
				*s = (*s)[1:]
			} else if k == s.Len() {
				*s = (*s)[:k-1]
			} else {
				*s = append((*s)[:k], (*s)[k+1:]...)
			}
			break
		}
	}
}
func (s *PeersRoute) removeByApp(app byte) {
	for k, v := range *s {
		if v[1] == app {
			if k == 0 {
				*s = (*s)[1:]
			} else if k == s.Len() {
				*s = (*s)[:k-1]
			} else {
				*s = append((*s)[:k], (*s)[k+1:]...)
			}
			break
		}
	}
}

func (s PeerRoute) equals(pr PeerRoute) bool {
	return bytes.Equal(s[:], pr[:])
}

func (s *resourcesPool) Register(mps []MethodPath, app, peer byte) {
	s.m.Lock()
	defer s.m.Unlock()
	if peer == 0 {
		peer = s.curPeer
	}
	pr := PeerRoute{s.curPeer, app}

	for _, mp := range mps {
		if s.topicPeers[mp] == nil {
			s.topicPeers[mp] = &PeersRoute{}
		}
		flag := true
		for _, peerRoute := range *s.topicPeers[mp] {
			if pr.equals(peerRoute) {
				flag = false
				break
			}
		}
		if flag {
			s.topicPeers[mp].append(pr)
		}
	}
}

func (s *resourcesPool) UnRegisterPeer(peer byte) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.removeByPeer(peer)
	}
}
func (s *resourcesPool) UnRegisterApp(app byte) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.removeByApp(app)
	}
}

func (s *resourcesPool) RouteIn(mp MethodPath) (interface{}, error) {
	_, ok := s.topicPeers[mp]
	if !ok {
		return nil, errors.New("MethodPath does not exist")
	}
	return nil, nil
}
func (s *resourcesPool) RouteOut(mp MethodPath) (interface{}, error) {
	return nil, nil
}
func (s *resourcesPool) listTopicPeers() map[MethodPath]*PeersRoute {
	return s.topicPeers
}
