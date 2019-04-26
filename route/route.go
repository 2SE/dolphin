package route

import (
	"errors"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"

	"hash/crc32"

	"github.com/2se/dolphin/common/ringhash"
)

var r *resourcesPool

var (
	ErrPeerNotFound         = errors.New("route: peer not found")
	ErrMethodPathNotFound   = errors.New("route: methodPath not found")
	ErrClientExists         = errors.New("route: client exists")
	ErrAddressNotFound      = errors.New("route: address not fount")
	ErrGprcServerConnFailed = errors.New("route: connection to grpc server failed")
)

const logFieldKey = "route"

type Router interface {
	//获取路由分流指向
	RouteIn(mp common.MethodPath, id string) (pr common.PeerRouter, redirect bool, err error)
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
	listTopicPeers() map[common.MethodPath]*common.PeerRouters
}

func GetRouterInstance() Router {
	return r
}

// 初始化本地route
// peer 本地cluster 编号
func InitRoute(peer string, cnf *config.RouteConfig) Router {
	r = &resourcesPool{
		curPeer:    peer,
		topicPeers: make(map[common.MethodPath]*common.PeerRouters),
		ring:       make(map[common.MethodPath]*ringhash.Ring),
		appAddr:    make(map[string]string),
		recycle:    cnf.Recycle.Duration,
		threshold:  cnf.Threshold,
		timeout:    cnf.Timeout.Duration,
	}
	return r
}

type resourcesPool struct {
	curPeer string
	appAddr map[string]string //key:appName val:address (only save local app)
	addrApp map[string]string //key:address val:appname
	connErr map[string]int16  //key address val:count the err count in a period time for client send request

	topicPeers map[common.MethodPath]*common.PeerRouters
	clients    map[string]pb.AppServeClient //key:address val:grpcClient (only save local app)
	ring       map[common.MethodPath]*ringhash.Ring
	recycle    time.Duration
	threshold  int16
	timeout    time.Duration
	m          sync.RWMutex
}

func (s *resourcesPool) Register(mps []common.MethodPath, appName, peerName, address string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if peerName == "" {
		peerName = s.curPeer
		err := s.TryAddClient(address)
		if err != nil {
			log.WithFields(log.Fields{
				logFieldKey: "Register",
			}).Errorln(err)
			return err
		}
		s.addrApp[address] = appName
		s.appAddr[appName] = address
	}
	pr := common.NewPeerRouter(peerName, appName)

	for _, mp := range mps {
		if s.topicPeers[mp] == nil {
			s.topicPeers[mp] = &common.PeerRouters{}
		}
		flag := true
		for _, peerRoute := range *s.topicPeers[mp] {
			if pr == peerRoute {
				flag = false
				break
			}
		}
		if flag {
			s.topicPeers[mp].Append(pr)
			s.topicPeers[mp].Sort()
		}
	}
	return nil
}

func (s *resourcesPool) UnRegisterPeer(peerName string) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.RemoveByPeer(peerName)
	}
}
func (s *resourcesPool) UnRegisterApp(appName string) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.RemoveByApp(appName)
	}
	if address, ok := s.appAddr[appName]; ok {
		delete(s.appAddr, appName)
		delete(s.addrApp, address)
		s.RemoveClient(address)
	}
}

func (s *resourcesPool) RouteIn(mp common.MethodPath, id string) (pr common.PeerRouter, redirect bool, err error) {
	psr, ok := s.topicPeers[mp]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteIn",
		}).Warnf("methodpath %s not found\n", mp.String())
		return nil, false, ErrMethodPathNotFound
	}
	if _, ok := s.ring[mp]; !ok {
		keys := make([]string, 0, psr.Len())
		for _, v := range *psr {
			keys = append(keys, v.PeerName())
		}
		ring := ringhash.New(psr.Len(), crc32.ChecksumIEEE)
		ring.Add(keys...)
		s.ring[mp] = ring
	}
	peer := s.ring[mp].Get(id)
	pa, err := psr.FindOne(peer)
	if err != nil {
		log.WithFields(log.Fields{
			logFieldKey: "RouteIn",
		}).Warnf("peer %s not exists\n", peer)
		return nil, false, err
	}
	if peer != s.curPeer {
		redirect = true
	}
	return pa, redirect, nil
}

func (s *resourcesPool) RouteOut(appName string, request proto.Message) (response proto.Message, err error) {
	addr, ok := s.appAddr[appName]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteOut",
		}).Warnf("appName %s not found\n", appName)
		return nil, ErrAddressNotFound
	}
	return s.callAppAction(addr, request)
}
func (s *resourcesPool) listTopicPeers() map[common.MethodPath]*common.PeerRouters {
	return s.topicPeers
}
