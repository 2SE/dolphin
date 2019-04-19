package cluster

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/2se/dolphin/config"
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

	tcpNetwork = "tcp"

	LocalPeer = "local"

	defaultVoteAfter = 8
	defaultFailAfter = 16
)

var (
	NoConfigErr = errors.New("no configuration")
	NoNodesErr  = errors.New("no peers defined")

	// gwCluster 表示集群中的本节点。如果gwCluster为nil，表示当前以单节点运行，而非集群模式。
	gwCluster *Cluster

	// peerConnCnf 集群内节点之间连接的配置
	// 连接配置需要在集群模块初始化时进行初始化
	// 如果该配置为nil，与连接配置相关的功能将使用默认参数
	peerConnCnf *config.ClusterConnectionConfig
)

// Init 初始化集群，返回本节点在集群中的名称
// 如果是单实例模式，节点名称为local
// 如果是集群模式，节点名称为配置文件中定义的本节点的名称
// 如果未传入配置文件，返回错误，节点名称为空字符串
func Init(cnf *config.ClusterConfig) (peerName string, err error) {
	log.Debug("cluster: starting init")
	if cnf == nil {
		log.Errorf("cluster: %s", NoConfigErr)
		return "", NoConfigErr
	}

	// Name of the current node is not specified - disable clustering
	if cnf.Self == "" {
		log.Debug("cluster: config with a singleton mode.")
		return LocalPeer, nil
	}

	gwCluster = &Cluster{
		thisName: cnf.Self,
		peers:    make(map[string]*peer),
	}
	peerConnCnf = cnf.Connection
	log.WithField("peer config", peerConnCnf).Debug("cluster: peer config check")

	var nodeNames []string
	for _, host := range cnf.Nodes {
		nodeNames = append(nodeNames, host.Name)

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
		return "", NoNodesErr
	}

	gwCluster.failoverInit(cnf.Failover)
	log.Debug("cluster: cluster inited")
	return gwCluster.thisName, nil
}

func Start() {
	log.Debug("cluster: starting now...")
	if gwCluster == nil {
		log.Info("cluster: runing in singleton mode")
		return
	}

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

	log.Infof("[cluster] cluster of %d peers initialized, node '%s' listening on [%s]",
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
		n.done <- true
	}

	time.Sleep(time.Second)
	gwCluster.inbound.Close()

	log.Info("cluster: cluster stopped")
}

// Partitioned checks if the cluster is partitioned due to network or other failure and if the
// current node is a part of the smaller partition.
func Partitioned() bool {
	if gwCluster == nil || gwCluster.fo == nil {
		// Cluster not initialized or failover disabled therefore not partitioned.
		return false
	}

	return (len(gwCluster.peers)+1)/2 >= len(gwCluster.fo.activeNodes)
}

func Name() string {
	if gwCluster != nil {
		return gwCluster.thisName
	}

	return LocalPeer
}

func Recv() {

}

func Emit() {

}

// 集群有两个角色一个是主节点角色，一个是从节点角色。在同一时间，集群的节点只承担一种角色
type Cluster struct {
	// 集群中其他的节点列表
	peers map[string]*peer
	// 本节点的名称
	thisName string
	// 本节点的集群服务地址
	listenOn string
	// 本节点的进群服务TCP
	inbound *net.TCPListener
	// Failover parameters. Could be nil if failover is not enabled
	fo *clusterFailover
}

// Emit 用于接收来自远程节点的APP请求调用，并调用本地Router.RouteOut方法
// 看作是客户端发起的调用
// Called by a remote peer.
func (c *Cluster) Emit(msg *RequestPkt, rejected *bool) error {
	log.Printf("cluster: Master request received from node '%s'", msg.Node)
	// 调用Router模块的 RouteOut方法

	//if msg.Signature == c.ring.Signature() {
	//	// TODO 处理具体的业务逻辑
	//} else {
	//	*rejected = true
	//}
	return nil
}

// Recv 用于接收来自远端节点的内部广播，并调用本地Router.Register方法.
// 看作是内部节点对其他节点的广播，不对客户端开放
// Called by remote peer
func (c *Cluster) Recv(msg *RespPkt, unused *bool) error {
	log.Println("cluster: response from Master for session", msg.FromSID)
	// TODO
	return nil
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
				// 这个ping来自老的主节点，忽略。
				log.Info("cluster: ping from a stale leader", ping.Term, c.fo.term, ping.Leader, c.fo.leader)
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

		case vreq := <-c.fo.electionVote:
			log.WithField("recv term", vreq.req.Term).
				WithField("remote name", vreq.req.Node).
				WithField("this name", c.thisName).Debug("cluster: recv election vote...")

			if c.fo.term < vreq.req.Term {
				// This is a new election. This node has not voted yet. Vote for the requestor and
				// clear the current leader.
				log.Infof("Voting YES for %s, my term %d, vote term %d", vreq.req.Node, c.fo.term, vreq.req.Term)
				c.fo.term = vreq.req.Term
				c.fo.leader = ""
				vreq.resp <- VoteResponse{Result: true, Term: c.fo.term}
			} else {
				// This node has voted already or stale election, reject.
				log.Infof("Voting NO for %s, my term %d, vote term %d", vreq.req.Node, c.fo.term, vreq.req.Term)
				vreq.resp <- VoteResponse{Result: false, Term: c.fo.term}
			}
		case <-c.fo.done:
			return
		}
	}
}
