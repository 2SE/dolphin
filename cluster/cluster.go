package cluster

import (
	"errors"
	"github.com/2se/dolphin/config"
	rh "github.com/2se/dolphin/ringhash"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"sort"
	"time"
)

const (
	// Default timeout before attempting to reconnect to a node
	defaultClusterReconnect = 200 * time.Millisecond
	// Number of replicas in ringhash
	clusterHashReplicas = 20
)

var (
	NoConfigErr = errors.New("no configuration")
	NoNodesErr  = errors.New("no peers defined")

	gwCluster  *Cluster
	singleMode bool
)

func Init(cnf *config.ClusterConfig) (workerId int, err error) {
	if cnf == nil {
		return 0, NoConfigErr
	}

	// Name of the current node is not specified - disable clustering
	if cnf.Self == "" {
		log.Println("Running as a standalone server.")
		singleMode = true
		return 1, nil
	}

	gwCluster = &Cluster{
		thisName: cnf.Self,
		peers:    make(map[string]*peer),
	}

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
		return 0, NoNodesErr
	}

	if gwCluster.failoverInit(cnf.Failover) {
		gwCluster.rehash(nil)
	}

	sort.Strings(nodeNames)
	workerId = sort.SearchStrings(nodeNames, cnf.Self) + 1

	return workerId, nil
}

func Start() {
	if singleMode {
		return
	}

	addr, err := net.ResolveTCPAddr("tcp", gwCluster.listenOn)
	if err != nil {
		// 启动失败，打印日志并退出
		log.Fatal(err)
	}

	if gwCluster.inbound, err = net.ListenTCP("tcp", addr); err != nil {
		log.Fatal(err)
	}

	for _, n := range gwCluster.peers {
		go n.reconnect()
	}

	if gwCluster.fo != nil {
		go gwCluster.run()
	}

	err = rpc.Register(gwCluster)
	if err != nil {
		log.Fatal(err)
	}

	go rpc.Accept(gwCluster.inbound)

	log.Printf("Cluster of %d peers initialized, node '%s' listening on [%s]",
		len(gwCluster.peers)+1,
		gwCluster.thisName,
		gwCluster.listenOn,
	)
}

func Shutdown() {
	if singleMode {
		return
	}

	gwCluster.inbound.Close()

	if gwCluster.fo != nil {
		gwCluster.fo.done <- true
	}

	for _, n := range gwCluster.peers {
		n.done <- true
	}

	log.Println("Cluster shut down")
}

func Route() {
	if singleMode {
		// TODO 本节点处理
	}

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
	// 节点一致性哈希环
	ring *rh.Ring
	// Failover parameters. Could be nil if failover is not enabled
	fo *clusterFailover
}

// Master
// Called by a remote peer.
func (c *Cluster) Master(msg *RequestPkt, rejected *bool) error {
	log.Printf("cluster: Master request received from node '%s'", msg.Node)
	if msg.Signature == c.ring.Signature() {
		// TODO 处理具体的业务逻辑
	} else {
		*rejected = true
	}
	return nil
}

// Proxy receives messages from the master node addressed to a specific local memory.
// Called by remote peer
func (c *Cluster) Proxy(msg *RespPkt, unused *bool) error {
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

// Recalculate the ring hash using provided list of peers or only peers in a non-failed state.
// Returns the list of peers used for ring hash.
func (c *Cluster) rehash(nodes []string) []string {
	ring := rh.New(clusterHashReplicas, nil)

	var ringKeys []string

	if nodes == nil {
		for _, node := range c.peers {
			ringKeys = append(ringKeys, node.name)
		}
		ringKeys = append(ringKeys, c.thisName)
	} else {
		ringKeys = append(ringKeys, nodes...)
	}
	ring.Add(ringKeys...)

	c.ring = ring

	return ringKeys
}

// run
// 1. 心跳检测
// 作为主节点，向从节点发送ping
// 或者作为从节点，当心跳检测周期到达时检查心跳关系，如果接收ping的次数超过了阀值，则开启新一轮主从选举
// 2. 作为从节点，接收来自主节点的ping消息
// 3. 接收来自投票选举的消息
func (c *Cluster) run() {
	// 集群中每个节点的心跳频率是不一样的，来确保大家不会在同一时间开启选举
	hbTicker := time.NewTicker(c.fo.heartBeat)

	missed := 0
	// Don't rehash immediately on the first ping. If this node just came onlyne, leader will
	// account it on the next ping. Otherwise it will be rehashing twice.
	rehashSkipped := false

	for {
		select {
		case <-hbTicker.C:
			if c.fo.leader == c.thisName {
				// I'm the leader, send pings
				c.sendPings()
			} else {
				missed++
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
				log.Println("cluster: ping from a stale leader", ping.Term, c.fo.term, ping.Leader, c.fo.leader)
				continue
			}

			if ping.Term > c.fo.term {
				// 切换本地的主节点信息
				c.fo.term = ping.Term
				c.fo.leader = ping.Leader
				log.Printf("cluster: leader '%s' elected", c.fo.leader)
			} else if ping.Leader != c.fo.leader {
				if c.fo.leader != "" {
					// Wrong leader. It's a bug, should never happen!
					log.Printf("cluster: wrong leader '%s' while expecting '%s'; term %d",
						ping.Leader, c.fo.leader, ping.Term)
				} else {
					log.Printf("cluster: leader set to '%s'", ping.Leader)
				}
				c.fo.leader = ping.Leader
			}

			missed = 0
			if ping.Signature != c.ring.Signature() {
				if rehashSkipped {
					log.Println("cluster: rehashing at a request of",
						ping.Leader, ping.Nodes, ping.Signature, c.ring.Signature())
					c.rehash(ping.Nodes)
					rehashSkipped = false

				} else {
					rehashSkipped = true
				}
			}

		case vreq := <-c.fo.electionVote:
			if c.fo.term < vreq.req.Term {
				// This is a new election. This node has not voted yet. Vote for the requestor and
				// clear the current leader.
				log.Printf("Voting YES for %s, my term %d, vote term %d", vreq.req.Node, c.fo.term, vreq.req.Term)
				c.fo.term = vreq.req.Term
				c.fo.leader = ""
				vreq.resp <- VoteResponse{Result: true, Term: c.fo.term}
			} else {
				// This node has voted already or stale election, reject.
				log.Printf("Voting NO for %s, my term %d, vote term %d", vreq.req.Node, c.fo.term, vreq.req.Term)
				vreq.resp <- VoteResponse{Result: false, Term: c.fo.term}
			}
		case <-c.fo.done:
			return
		}
	}
}
