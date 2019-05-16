package router

import (
	"errors"
	"github.com/2se/dolphin/common/ringhash"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/pb"
	tw "github.com/RussellLuo/timingwheel"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"hash/crc32"
	"sync"
	"time"
)

var (
	r      *resourcesPool
	ticker *tw.TimingWheel
)

var (
	ErrMethodPathNotFound   = errors.New("route: methodPath not found")
	ErrClientExists         = errors.New("route: client exists")
	ErrAddressNotFound      = errors.New("route: address not fount")
	ErrGprcServerConnFailed = errors.New("route: connection to grpc server failed")
)

const logFieldKey = "router"

// 初始化本地route
// peer 本地cluster 编号
func Init(localPeer core.LocalPeer, cnf *config.RouteConfig, twheel *tw.TimingWheel) core.Router {
	ticker = twheel
	r = &resourcesPool{
		localPeer:  localPeer,
		pRAddr:     make(map[string]string),
		addrPR:     make(map[string]core.PeerRouter),
		connErr:    make(map[string]int16),
		topicPeers: make(map[string]*core.PeerRouters),
		conns:      make(map[string]*grpc.ClientConn),
		clients:    make(map[string]pb.AppServeClient),
		ring:       make(map[string]*ringhash.Ring),
		recycle:    cnf.Recycle.Duration,
		threshold:  cnf.Threshold,
		timeout:    cnf.Timeout.Duration,
		heartBeat:  cnf.HeartBeat.Duration,
	}
	go r.errRecovery()
	go r.healthCheck()
	return r
}

type resourcesPool struct {
	localPeer  core.LocalPeer
	pRAddr     map[string]string            //key:core.PeerRouter val:address (only save local app)
	addrPR     map[string]core.PeerRouter   //key:address val:appname
	connErr    map[string]int16             //key address val:count the err count in a period time for client send request
	topicPeers map[string]*core.PeerRouters //key: core.MethodPath
	clients    map[string]pb.AppServeClient //key:address val:grpcClient (only save local app)
	conns      map[string]*grpc.ClientConn
	ring       map[string]*ringhash.Ring //key: core.MethodPath
	recycle    time.Duration
	threshold  int16
	timeout    time.Duration
	heartBeat  time.Duration
	m          sync.RWMutex
}

func (s *resourcesPool) Register(mps []core.MethodPath, pr core.PeerRouter, address string) error {
	s.m.Lock()
	defer s.m.Unlock()
	if pr.PeerName() == "" {
		pr.SetPeerName(s.localPeer.Name())
		err := s.TryAddClient(address)
		if err != nil {
			log.WithFields(log.Fields{
				logFieldKey: "Register",
			}).Errorln(err)
			return err
		}
		s.addrPR[address] = pr
		s.pRAddr[pr.String()] = address
	}
	for _, mp := range mps {
		if s.topicPeers[mp.String()] == nil {
			s.topicPeers[mp.String()] = &core.PeerRouters{}
		}
		flag := true
		for _, peerRoute := range *s.topicPeers[mp.String()] {
			if pr.String() == peerRoute.String() {
				flag = false
				break
			}
		}
		if flag {
			s.topicPeers[mp.String()].Append(pr)
			s.topicPeers[mp.String()].Sort()
		}
	}
	s.localPeer.Notify(pr, mps...)
	return nil
}

func (s *resourcesPool) UnRegisterPeer(peerName string) {
	s.m.Lock()
	defer s.m.Unlock()
	for _, prs := range s.topicPeers {
		prs.RemoveByPeer(peerName)
	}
}

func (s *resourcesPool) UnRegisterApp(pr core.PeerRouter) {
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
	s.localPeer.Notify(pr)
}

func (s *resourcesPool) RouteIn(mp core.MethodPath, id string, request proto.Message) (response proto.Message, err error) {
	s.m.RLock()
	defer s.m.RUnlock()
	psr, ok := s.topicPeers[mp.String()]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteIn",
		}).Warnf("methodpath %s not found\n", mp.String())
		return nil, ErrMethodPathNotFound
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
		return nil, err
	}
	return s.localPeer.Request(pa, request)
}

func (s *resourcesPool) RouteOut(pr core.PeerRouter, request proto.Message) (response proto.Message, err error) {
	addr, ok := s.pRAddr[pr.String()]
	if !ok {
		log.WithFields(log.Fields{
			logFieldKey: "RouteOut",
		}).Warnf("peerRouter %s not found\n", pr.String())
		return nil, ErrAddressNotFound
	}
	return s.callAppAction(addr, request)
}

func (s *resourcesPool) ListTopicPeers() map[string]*core.PeerRouters {
	return s.topicPeers
}
