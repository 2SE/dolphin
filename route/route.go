package route

import (
	"errors"
	"github.com/2se/dolphin/common"
	"github.com/golang/protobuf/proto"
	"sync"

	"sort"

	"hash/crc32"

	"github.com/2se/dolphin/ringhash"
)

var r *resourcesPool

var (
	ErrPeerNotFound         = errors.New("route: peer not found")
	ErrMethodPathNotFound   = errors.New("route: methodPath not found")
	ErrClientExists         = errors.New("route: client exists")
	ErrAddressNotFound      = errors.New("route: address not fount")
	ErrGprcServerConnFailed = errors.New("route: connection to grpc server failed")
)

type Router interface {
	//获取路由分流指向
	RouteIn(mp common.MethodPath, id string) (pr PeerRoute, redirect bool, err error)
	//更具appName将request定向到指定Grpc服务并返回结果
	RouteOut(appName string, request proto.Message) (response proto.Message, err error)
	//注册单个
	//appName 资源服务名称
	//peer 节点名称（空字符串为本地）
	//address  资源服务连接地址（ps:www.example.com:8080）
	Register(mps []common.MethodPath, appName, peerName, address string) error
	//注销app下所有
	UnRegisterApp(appName string)
	//注销peer下所有
	UnRegisterPeer(peerName string)
	//for unit test
	listTopicPeers() map[common.MethodPath]*PeersRoute
}

func GetRouterInstance() Router {
	return r
}

// 初始化本地route
// peer 本地cluster 编号
func InitRoute(peer string) Router {
	r = &resourcesPool{
		curPeer:    peer,
		topicPeers: make(map[common.MethodPath]*PeersRoute),
		ring:       make(map[common.MethodPath]*ringhash.Ring),
		appAddr:    make(map[string]string),
	}
	return r
}

type resourcesPool struct {
	curPeer string
	appAddr map[string]string //key:appName val:address (only save local app)
	addrApp map[string]string //key:address val:appname
	connErr map[string]int16  //key address val:count the err count in a period time for client send request

	topicPeers map[common.MethodPath]*PeersRoute
	clients    map[string]AppServeClient //key:address val:grpcClient (only save local app)
	ring       map[common.MethodPath]*ringhash.Ring
	m          sync.RWMutex
}

type PeerRoute [2]string

type PeersRoute []PeerRoute

func (s PeerRoute) PeerName() string {
	return s[0]

}
func (s PeerRoute) AppName() string {
	return s[1]
}
func (s PeersRoute) Len() int      { return len(s) }
func (s PeersRoute) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PeersRoute) Less(i, j int) bool {
	return s[i][0] < s[j][0]
}
func (s *PeersRoute) append(pr PeerRoute) {
	*s = append(*s, pr)
}
func (s *PeersRoute) sort() {
	sort.Sort(s)
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
func (s PeersRoute) findOne(peerName string) (PeerRoute, error) {
	for _, v := range s {
		if v[0] == peerName {
			return v, nil
		}
	}
	return PeerRoute{}, ErrPeerNotFound
}
func (s PeerRoute) equals(pr PeerRoute) bool {
	return s == pr
}

func (s *resourcesPool) Register(mps []common.MethodPath, appName, peerName, address string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if peerName == "" {
		peerName = s.curPeer
		err := s.TryAddClient(address)
		if err != nil {
			return err
		}
		s.addrApp[address] = appName
		s.appAddr[appName] = address
	}
	pr := PeerRoute{peerName, appName}

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
			s.topicPeers[mp].sort()
		}
	}
	return nil
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
	if address, ok := s.appAddr[appName]; ok {
		delete(s.appAddr, appName)
		delete(s.addrApp, address)
		s.RemoveClient(address)
	}
}

func (s *resourcesPool) RouteIn(mp common.MethodPath, id string) (pr PeerRoute, redirect bool, err error) {
	psr, ok := s.topicPeers[mp]
	if !ok {
		return PeerRoute{}, false, ErrMethodPathNotFound
	}
	if _, ok := s.ring[mp]; !ok {
		keys := make([]string, 0, psr.Len())
		for _, v := range *psr {
			keys = append(keys, v[0])
		}
		ring := ringhash.New(psr.Len(), crc32.ChecksumIEEE)
		ring.Add(keys...)
		s.ring[mp] = ring
	}
	peer := s.ring[mp].Get(id)
	pa, err := psr.findOne(peer)
	if err != nil {
		return PeerRoute{}, false, err
	}
	if peer != s.curPeer {
		redirect = true
	}
	return pa, redirect, err
}

func (s *resourcesPool) RouteOut(appName string, request proto.Message) (response proto.Message, err error) {
	addr, ok := s.appAddr[appName]
	if !ok {
		return nil, ErrAddressNotFound
	}
	return s.callAppAction(addr, request)
}
func (s *resourcesPool) listTopicPeers() map[common.MethodPath]*PeersRoute {
	return s.topicPeers
}
