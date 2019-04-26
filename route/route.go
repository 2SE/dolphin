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
	ErrMethodPathNotFound   = errors.New("route: methodPath not found")
	ErrClientExists         = errors.New("route: client exists")
	ErrAddressNotFound      = errors.New("route: address not fount")
	ErrGprcServerConnFailed = errors.New("route: connection to grpc server failed")
)

const logFieldKey = "route"

type Router interface {
	//获取路由分流指向
	RouteIn(mp common.MethodPath, id string) (pr common.PeerRouter, redirect bool, err error)
	//根据peerRouter将request定向到指定Grpc服务并返回结果
	RouteOut(pr common.PeerRouter, request proto.Message) (response proto.Message, err error)
	//注册单个
	//appName 资源服务名称
	//peer 节点名称（空字符串为本地）
	//address  资源服务连接地址（ps:www.example.com:8080）
	Register(mps []common.MethodPath, appName, peerName, address string) error
	//注销app下所有
	UnRegisterApp(pr common.PeerRouter)
	//注销peer下所有
	UnRegisterPeer(peerName string)
	//for unit test
	listTopicPeers() map[string]*common.PeerRouters
}

func GetRouterInstance() Router {
	return r
}

// 初始化本地route
// peer 本地cluster 编号
func InitRoute(peer string, cnf *config.RouteConfig) Router {
	r = &resourcesPool{
		curPeer:    peer,
		topicPeers: make(map[string]*common.PeerRouters),
		ring:       make(map[string]*ringhash.Ring),
		//appAddr:    make(map[string]string),
		recycle:   cnf.Recycle.Duration,
		threshold: cnf.Threshold,
		timeout:   cnf.Timeout.Duration,
	}
	return r
}

type resourcesPool struct {
	curPeer string
	pRAddr  map[string]string            //key:common.PeerRouter val:address (only save local app)
	addrPR  map[string]common.PeerRouter //key:address val:appname
	connErr map[string]int16             //key address val:count the err count in a period time for client send request

	topicPeers map[string]*common.PeerRouters //key: common.MethodPath
	clients    map[string]pb.AppServeClient   //key:address val:grpcClient (only save local app)
	ring       map[string]*ringhash.Ring      //key: common.MethodPath
	recycle    time.Duration
	threshold  int16
	timeout    time.Duration
	m          sync.RWMutex
}

func (s *resourcesPool) Register(mps []common.MethodPath, appName, peerName, address string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if peerName == "" || peerName == s.curPeer {
		peerName = s.curPeer
		err := s.TryAddClient(address)
		if err != nil {
			log.WithFields(log.Fields{
				logFieldKey: "Register",
			}).Errorln(err)
			return err
		}

	}
	pr := common.NewPeerRouter(peerName, appName)
	s.addrPR[address] = pr
	s.pRAddr[pr.String()] = address
	for _, mp := range mps {
		if s.topicPeers[mp.String()] == nil {
			s.topicPeers[mp.String()] = &common.PeerRouters{}
		}
		flag := true
		for _, peerRoute := range *s.topicPeers[mp.String()] {
			if pr == peerRoute {
				flag = false
				break
			}
		}
		if flag {
			s.topicPeers[mp.String()].Append(pr)
			s.topicPeers[mp.String()].Sort()
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

func (s *resourcesPool) UnRegisterApp(pr common.PeerRouter) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.RemoveByPeerRouter(pr)
	}
	if address, ok := s.pRAddr[pr.String()]; ok {
		delete(s.pRAddr, pr.String())
		delete(s.addrPR, address)
		s.RemoveClient(address)
	}
}

func (s *resourcesPool) RouteIn(mp common.MethodPath, id string) (pr common.PeerRouter, redirect bool, err error) {
	psr, ok := s.topicPeers[mp.String()]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteIn",
		}).Warnf("methodpath %s not found\n", mp.String())
		return nil, false, ErrMethodPathNotFound
	}
	if _, ok := s.ring[mp.String()]; !ok {
		keys := make([]string, 0, psr.Len())
		for _, v := range *psr {
			keys = append(keys, v.PeerName())
		}
		ring := ringhash.New(psr.Len(), crc32.ChecksumIEEE)
		ring.Add(keys...)
		s.ring[mp.String()] = ring
	}
	peer := s.ring[mp.String()].Get(id)
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

func (s *resourcesPool) RouteOut(pr common.PeerRouter, request proto.Message) (response proto.Message, err error) {
	addr, ok := s.pRAddr[pr.String()]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteOut",
		}).Warnf("peerRouter %s not found\n", pr.String())
		return nil, ErrAddressNotFound
	}
	return s.callAppAction(addr, request)
}
func (s *resourcesPool) listTopicPeers() map[string]*common.PeerRouters {
	return s.topicPeers
}
