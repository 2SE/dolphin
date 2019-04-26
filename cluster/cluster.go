package cluster

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/2se/dolphin/common"
	"github.com/2se/dolphin/config"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"time"
)

const (
	// 默认的在首次重连节点前超时等待时间
	// 如果多次重连节点，连接等待超时时间根据backoff算法会不断延长
	defaultClusterReconnect = 200 * time.Millisecond
	// 默认的net/rpc客户端连接超时时间
	defaultDialTimeout = 5 * time.Second

	LocalPeer        = "local"
	tcpNetwork       = "tcp"
	ReportMethod     = "Cluster.Report"
	RequestMethod    = "Cluster.Invoke"
	defaultVoteAfter = 8
	defaultFailAfter = 16
)

var (
	NoConfigErr        = errors.New("no configuration")
	NoPeersErr         = errors.New("no peers defined")
	PartitionedErr     = errors.New("cluster partitioned")
	PeerUnavailableErr = errors.New("peer unavailable")

	gwCluster *Cluster

	// peerConnCnf 集群内节点之间连接的配置
	// 连接配置需要在集群模块初始化时进行初始化
	// 如果该配置为nil，与连接配置相关的功能将使用默认参数
	peerConnCnf *config.ClusterConnectionConfig
)

type delegatedCluster struct {
	router common.Router
}

func (d *delegatedCluster) SetRouter(router common.Router) {
	d.router = router
}

func (d *delegatedCluster) Name() string {
	if gwCluster != nil {
		return gwCluster.thisName
	}

	return LocalPeer
}

// Partitioned checks if the cluster is partitioned due to network or other failure and if the
// current peer is a part of the smaller partition.
func (d *delegatedCluster) Partitioned() bool {
	if gwCluster == nil || gwCluster.fo == nil {
		// Cluster not initialized or failover disabled therefore not partitioned.
		return false
	}

	return (len(gwCluster.peers)+1)/2 >= len(gwCluster.fo.activePeers)
}

// Recv call remote Cluster.Recv
// The request from remote peer`s broadcast.
func (d *delegatedCluster) Notify(mps []common.MethodPath, pr common.PeerRouter) {
	if gwCluster == nil {
		return
	}

	if d.Partitioned() {
		log.Warnf("Notify: %v", PartitionedErr)
		return
	}

	req := reqPool.Get().(*RequestPkt)
	req.PeerName = gwCluster.thisName
	//req.Pkt = message

	peerCount := len(gwCluster.peers)
	done := make(chan *rpc.Call, peerCount)
	for _, peer := range gwCluster.peers {
		resp := &RespPkt{}
		peer.callAsync(ReportMethod, req, resp, done)
	}
	reqPool.Put(req)

	// TODO change timeout time
	timeout := time.NewTimer(defaultTimeout)
	for i := 0; i < peerCount; i++ {
		select {
		case call := <-done:
			if call.Error != nil {
				log.Warnf("cluster: call remote method failed. cause: %v", call.Error)
			}
		case <-timeout.C:
			i = peerCount
		}
	}

	if !timeout.Stop() {
		<-timeout.C
	}
}

// Request call remote Cluster.Request
// the request from client
// it will choose one peer to send request and reecived call back response
func (d *delegatedCluster) Request(value common.PeerRouter, message proto.Message) (response proto.Message, err error) {
	if gwCluster == nil {
		return d.router.RouteOut(value, message)
	}

	if d.Partitioned() {
		return nil, PartitionedErr
	}

	if peer, ok := gwCluster.peers[value.PeerName()]; ok {
		req := reqPool.Get().(*RequestPkt)
		defer reqPool.Put(req)

		req.PeerName = gwCluster.thisName
		req.AppName = value.AppName()
		req.Signature = gwCluster.signature
		req.Pkt = message

		resp := respPool.Get().(*RespPkt)
		defer respPool.Put(resp)
		if err = peer.call(RequestMethod, req, resp); err != nil {
			return nil, err
		}

		response = resp.Pkt
		return
	}

	return nil, PeerUnavailableErr
}

// Init 初始化集群，返回本节点在集群中的名称
// 如果是单实例模式，节点名称为local
// 如果是集群模式，节点名称为配置文件中定义的本节点的名称
// 如果未传入配置文件，返回错误，节点名称为空字符串
func Init(cnf *config.ClusterConfig) (localCluster common.LocalCluster, err error) {
	log.Debug("cluster: starting init")
	if cnf == nil {
		log.Errorf("cluster: %s", NoConfigErr)
		return nil, NoConfigErr
	}

	// Name of the current peer is not specified - disable clustering
	if cnf.Self == "" {
		log.Debug("cluster: config with a singleton mode.")
		localCluster = &delegatedCluster{}
		return
	}

	gwCluster = &Cluster{
		thisName: cnf.Self,
		peers:    make(map[string]*peer),
		delegate: &delegatedCluster{},
	}

	peerConnCnf = cnf.Connection

	log.WithField("peer config", peerConnCnf).Debug("cluster: peer config check")
	var peerNames []string
	for _, host := range cnf.Nodes {
		peerNames = append(peerNames, host.Name)

		if host.Name == cnf.Self {
			gwCluster.listenOn = host.Address
			// Don't create a cluster member for this local instance
			continue
		}

		gwCluster.peers[host.Name] = &peer{
			address: host.Address,
			name:    host.Name,
			done:    make(chan bool, 1),
		}
	}

	if len(gwCluster.peers) == 0 {
		log.Error("cluster: no configuration for cluster")
		return nil, NoPeersErr
	}

	if !gwCluster.failoverInit(cnf.Failover) {
		gwCluster.checkPeers(true, nil)
	}

	log.Debug("cluster: cluster inited")
	return gwCluster.delegate, nil
}

func Start(router common.Router) {
	log.Debug("cluster: starting now...")
	if gwCluster == nil {
		log.Info("cluster: runing in singleton mode")
		return
	}

	if router == nil {
		log.Fatalf("cluster: no router")
	}

	gwCluster.delegate.SetRouter(router)
	addr, err := net.ResolveTCPAddr(tcpNetwork, gwCluster.listenOn)
	if err != nil {
		// 启动失败，打印日志并退出
		log.WithError(err).Fatal("cluster: parse ip address failed! exit(1)")
	}

	if gwCluster.inbound, err = net.ListenTCP(tcpNetwork, addr); err != nil {
		log.WithError(err).Fatal("cluster: starting tcp listener failed! exit(1)")
	}

	go listen()

	for _, n := range gwCluster.peers {
		go n.reconnect()
	}

	log.Debug("cluster: all remote peers connected successful")
	if gwCluster.fo != nil {
		go gwCluster.run()
	}

	err = rpc.Register(gwCluster)
	if err != nil {
		log.WithError(err).Fatal("cluster: register net/rpc service failed! exit(1)")
	}

	log.Infof("cluster: %d peers initialized, peer '%s' listening on [%s]",
		len(gwCluster.peers)+1,
		gwCluster.thisName,
		gwCluster.listenOn,
	)
}

func listen() {
	if peerConnCnf != nil && peerConnCnf.DisableReqTimeout {
		log.Info("cluster: using default net/rpc server to handle request logic")
		rpc.Accept(gwCluster.inbound)
		log.Debug("cluster: net/rpc server stopped")
		return
	}

	log.Info("cluster: using net/rpc server with timeout mechanism to handle request logic")
	for {
		conn, err := gwCluster.inbound.Accept()
		if err != nil {
			log.Warnf("cluster: %v", err)
			return
		}

		go func(conn net.Conn) {
			buf := bufio.NewWriter(conn)
			srv := &gobServerCodec{
				rwc:    conn,
				dec:    gob.NewDecoder(conn),
				enc:    gob.NewEncoder(buf),
				encBuf: buf,
			}

			rpc.ServeCodec(srv)
		}(conn)
	}
}

func Shutdown() {
	if gwCluster == nil {
		log.Debug("cluster: singelton mode, and exit quickly")
		return
	}

	if gwCluster.fo != nil {
		gwCluster.fo.done <- true
	}

	for _, n := range gwCluster.peers {
		if n.connected {
			n.endpoint.Close()
		} else if n.reconnecting {
			n.done <- true
		}
	}

	time.Sleep(time.Second)
	gwCluster.inbound.Close()

	log.Info("cluster: cluster stopped")
}

// 集群有两个角色一个是主节点角色，一个是从节点角色。在同一时间，集群的节点只承担一种角色
type Cluster struct {
	peers     map[string]*peer // 集群中其他的节点列表
	signature string           // signature for peers array(sort by charactor asc)
	thisName  string           // 本节点的名称
	listenOn  string           // 本节点的集群服务地址
	inbound   *net.TCPListener // 本节点的进群服务TCP
	fo        *clusterFailover // Failover parameters. Could be nil if failover is not enabled
	delegate  *delegatedCluster
}

// Invoke called by a remote peer.
func (c *Cluster) Invoke(msg *RequestPkt, resp *RespPkt) (err error) {
	log.Printf("cluster: Invoke request received from peer '%s'", msg.PeerName)

	// the cluster peers may be in election phase
	if msg.Signature != c.signature {
		log.Warnf("cluster.Invoke: the cluster peers may be in election phase")
		resp.Code = 500
		resp.PeerValue = &PeerValue{c.thisName, c.listenOn}
		return
	}

	var result proto.Message
	pr := common.NewPeerRouter(msg.PeerName, msg.AppName)
	if result, err = c.delegate.router.RouteOut(pr, msg.Pkt); err != nil {
		log.WithError(err).Errorf("cluster: error found when remote invoke RouteOut method")
		resp.Code = 500
	} else {
		resp.Code = 200
		resp.Pkt = result
	}

	resp.PeerValue = &PeerValue{c.thisName, c.listenOn}
	return
}

// Report 用于接收来自远端节点的内部广播，并调用本地Router.Register方法.
// 看作是内部节点对其他节点的广播，不对客户端开放
// Called by remote peer
func (c *Cluster) Report(msg *RespPkt, unused *bool) error {
	log.Println("cluster: response from Master for session ", msg.PeerValue)
	return c.delegate.router.Register()
}

// Ping 集群内部接口，供远端主节点调用rpc.Client.Call("Cluster.Ping"...
func (c *Cluster) Ping(ping *PingRequest, unused *bool) error {
	select {
	case c.fo.leaderPing <- ping:
	default:
	}
	return nil
}

// Vote 集群内部接口，用于接收远端节点发出的投票rpc.Client.Go("Cluster.Vote"...
func (c *Cluster) Vote(req *VoteRequest, resp *VoteResponse) error {
	respChan := make(chan VoteResponse, 1)

	c.fo.electionVote <- &voteReqResp{
		req:  req,
		resp: respChan}

	*resp = <-respChan

	return nil
}

// run
// 1. 心跳检测
// 作为主节点，向从节点发送ping
// 或者作为从节点，当心跳检测周期到达时检查心跳关系，如果接收ping的次数超过了阀值，则开启新一轮主从选举
// 2. 作为从节点，接收来自主节点的ping消息
// 3. 接收来自投票选举的消息
func (c *Cluster) run() {
	log.Debug("cluster: starting heartbeat and leader election")
	// 集群中每个节点的心跳频率是不一样的，来确保大家不会在同一时间开启选举
	hbTicker := time.NewTicker(c.fo.heartBeat)

	missed := 0
	rehashSkipped := false

	for {
		select {
		case <-hbTicker.C:
			if c.fo.leader == c.thisName {
				// I'm the leader, send pings
				c.sendPings()
			} else {
				missed++

				log.WithField("votetimeout", c.fo.voteTimeout).
					WithField("missed", missed).
					Debug("check missed...")

				if missed >= c.fo.voteTimeout {
					// Elect the leader
					missed = 0
					c.electLeader()
				}
			}
		case ping := <-c.fo.leaderPing:
			// Ping from a leader.
			if ping.Term < c.fo.term {
				// This is a ping from a stale leader. Ignore.
				log.WithField("ping_term", ping.Term).
					WithField("failOver_term", c.fo.term).
					WithField("ping_leader", ping.Leader).
					WithField("failOver_leader", c.fo.leader).
					Info("cluster: ping from a stale leader")
				continue
			}

			if ping.Term > c.fo.term {
				// 切换本地的主节点信息
				c.fo.term = ping.Term
				c.fo.leader = ping.Leader
				log.Infof("cluster: leader '%s' elected", c.fo.leader)
			} else if ping.Leader != c.fo.leader {
				if c.fo.leader != "" {
					// Wrong leader. It's a bug, should never happen!
					log.Infof("cluster: wrong leader '%s' while expecting '%s'; term %d",
						ping.Leader, c.fo.leader, ping.Term)
				} else {
					log.Infof("cluster: leader set to '%s'", ping.Leader)
				}
				c.fo.leader = ping.Leader
			}

			missed = 0
			if ping.Signature != c.signature {
				if rehashSkipped {
					log.WithField("ping_leader", ping.Leader).
						WithField("ping_peers", ping.PeerValues).
						WithField("ping_signature", ping.Signature).
						WithField("self_signature", c.signature).
						Info("cluster: recheck activted peers at a request of")
					c.checkPeers(false, ping.PeerValues)
					rehashSkipped = false
				} else {
					rehashSkipped = true
				}
			}

		case vreq := <-c.fo.electionVote:
			log.WithField("recv term", vreq.req.Term).
				WithField("remote name", vreq.req.PeerValue).
				WithField("this name", c.thisName).Debug("cluster: recv election vote...")

			if c.fo.term < vreq.req.Term {
				// This is a new election. This peer has not voted yet. Vote for the requestor and
				// clear the current leader.
				log.Infof("Voting YES for %s, my term %d, vote term %d", vreq.req.PeerValue, c.fo.term, vreq.req.Term)
				c.fo.term = vreq.req.Term
				c.fo.leader = ""
				vreq.resp <- VoteResponse{Result: true, Term: c.fo.term}
			} else {
				// This peer has voted already or stale election, reject.
				log.Infof("Voting NO for %s, my term %d, vote term %d", vreq.req.PeerValue, c.fo.term, vreq.req.Term)
				vreq.resp <- VoteResponse{Result: false, Term: c.fo.term}
			}
		case <-c.fo.done:
			return
		}
	}
}

func (c *Cluster) checkPeers(onInit bool, peerValues []*PeerValue) {
	var names []*PeerValue
	if peerValues == nil {
		for _, n := range peerValues {
			names = append(names, n)
		}
		names = append(names, &PeerValue{c.thisName, c.listenOn})
	} else {
		names = append(names, peerValues...)
	}

	c.signature = digestPeerValue(names).Signature()
	log.WithField("signature", c.signature).Info("check peers and set signature")
	if !onInit {
		var found bool
		// finding old peer not in new peer list
		for key, peer := range c.peers {
			for _, name := range names {
				if key == name.PeerName {
					found = true
					break
				}
			}

			if !found {
				peer.disconnect()
				delete(c.peers, key)
				c.delegate.router.UnRegisterPeer(peer.name)
			}

			found = false
		}

		for _, name := range names {
			// finding new peer not in peer list
			if _, ok := c.peers[name.PeerName]; !ok && name.PeerName != c.thisName {
				newPeer := &peer{
					address: name.PeerAddr,
					name:    name.PeerName,
					done:    make(chan bool, 1),
				}

				c.peers[name.PeerName] = newPeer
				go newPeer.reconnect()
			}
		}
	}
}
