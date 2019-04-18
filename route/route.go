package route

import (
	"errors"
	"sync"
)

// fisrt byte  : version
// second byte : resource
// third byte  : action
type MethodPath [3]byte

var r *resourcesPool

type Router interface {
	RouteIn(mp MethodPath) (interface{}, error)
	RouteOut(mp MethodPath) (interface{}, error)
	//注册单个app上所有资源,peer 为 0 是默认本地
	Register(mps []MethodPath, appName, peerName string)
	//注销app下所有
	UnRegisterApp(appName string)
	//注销peer下所有
	UnRegisterPeer(peerName string)
	//for unit test
	listTopicPeers() map[MethodPath]*PeersRoute
}

func GetRouterInstance() Router {
	return r
}

// 初始化本地route
// peer 本地cluster 编号
func InitRoute(peer string) Router {
	route := &resourcesPool{
		curPeer:    peer,
		topicPeers: make(map[MethodPath]*PeersRoute),
	}
	return route
}

type resourcesPool struct {
	curPeer    string
	topicPeers map[MethodPath]*PeersRoute
	m          sync.RWMutex
}

type PeerRoute [2]string

type PeersRoute []PeerRoute

func (s PeersRoute) Len() int      { return len(s) }
func (s PeersRoute) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PeersRoute) Less(i, j int) bool {
	return s[i][0] < s[j][0]
}
func (s *PeersRoute) append(pr PeerRoute) {
	*s = append(*s, pr)
}
func (s *PeersRoute) removeByPeer(peer string) {
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
func (s *PeersRoute) removeByApp(app string) {
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
	return s == pr
}

func (s *resourcesPool) Register(mps []MethodPath, appName, peerName string) {
	s.m.Lock()
	defer s.m.Unlock()
	if peerName == "" {
		peerName = s.curPeer
	}
	pr := PeerRoute{s.curPeer, appName}

	for _, mp := range mps {
		if s.topicPeers[mp] == nil {
			s.topicPeers[mp] = &PeersRoute{}
		}
		flag := true
		for _, peerRoute := range *s.topicPeers[mp] {
			if pr == peerRoute {
				flag = false
				break
			}
		}
		if flag {
			s.topicPeers[mp].append(pr)
		}
	}
}

func (s *resourcesPool) UnRegisterPeer(peerName string) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.removeByPeer(peerName)
	}
}
func (s *resourcesPool) UnRegisterApp(appName string) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.removeByApp(appName)
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
