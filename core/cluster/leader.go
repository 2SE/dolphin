package cluster

import (
	"github.com/2se/dolphin/config"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/rpc"
	"time"
)

// Failover config
type clusterFailover struct {
	// Current leader
	leader string
	// Current election term
	term int
	// Hearbeat interval
	heartBeat time.Duration
	// Vote timeout: the number of missed heartbeats before a new election is initiated.
	voteTimeout int

	// The list of peers the leader considers active
	activePeers []*PeerValue
	// The number of heartbeats a peer can fail before being declared dead
	peerFailCountLimit int

	// Channel for processing leader pings
	leaderPing chan *PingRequest
	// Channel for processing election votes
	electionVote chan *voteReqResp
	// Channel for stopping the failover runner
	done chan bool
}

func (c *Cluster) failoverInit(config *config.ClusterFailoverConfig) bool {
	log.Debug("cluster: initing failover")
	if config == nil || !config.Enabled {
		log.Debug("cluster: not enable failover")
		return false
	}

	if len(c.peers) < 2 {
		log.Warnf("cluster: failover disabled; need at least 3 peers, got %d", len(c.peers)+1)
		return false
	}

	// Generate ring hash on the assumption that all peers are alive and well.
	// This minimizes rehashing during normal operations.
	var peerValues []*PeerValue
	for _, peer := range c.peers {
		peerValues = append(peerValues, &PeerValue{peer.name, peer.address})
	}
	peerValues = append(peerValues, &PeerValue{c.thisName, c.listenOn})
	c.checkPeers(true, peerValues)

	log.WithField("config_heartbeat", config.Heartbeat.Get()).Debug("heartbeat value from config")

	// Random heartbeat ticker: 0.75 * config.HeartBeat + random(0, 0.5 * config.HeartBeat)
	rand.Seed(time.Now().UnixNano())
	hb := config.Heartbeat.Get()
	hb = (hb >> 1) + (hb >> 2) + time.Duration(rand.Intn(int(hb>>1)))

	voteAfter := config.VoteAfter
	failAfter := config.NodeFailAfter

	if voteAfter <= 0 {
		voteAfter = defaultVoteAfter
	}

	if failAfter <= 0 {
		failAfter = defaultFailAfter
	}

	c.fo = &clusterFailover{
		activePeers:        peerValues,
		heartBeat:          hb,
		voteTimeout:        voteAfter,
		peerFailCountLimit: failAfter,
		leaderPing:         make(chan *PingRequest, voteAfter),
		electionVote:       make(chan *voteReqResp, len(c.peers)),
		done:               make(chan bool, 1)}

	log.WithField("heartBeat", hb).Info("cluster: failover mode enabled.")
	return true
}

func (c *Cluster) sendPings() {
	recalculate := false
	ping := pingPool.Get().(*PingRequest)
	ping.Leader = c.thisName
	ping.Term = c.fo.term
	ping.Signature = c.signature
	ping.PeerValues = c.fo.activePeers

	for _, peer := range c.peers {
		unused := false
		err := peer.call("Cluster.Ping", ping, &unused)
		if err != nil {
			peer.failCount++
			if peer.failCount == c.fo.peerFailCountLimit {
				// peer failed too many times
				recalculate = true
			}
		} else {
			if peer.failCount >= c.fo.peerFailCountLimit {
				// peer has recovered
				recalculate = true
			}
			peer.failCount = 0
		}
	}

	pingPool.Put(ping)

	if recalculate {
		var activePeers []*PeerValue
		for _, peer := range c.peers {
			if peer.failCount < c.fo.peerFailCountLimit {
				activePeers = append(activePeers, &PeerValue{peer.name, peer.address})
			}
		}
		activePeers = append(activePeers, &PeerValue{c.thisName, c.listenOn})
		c.fo.activePeers = activePeers
		c.checkPeers(false, activePeers)
		log.Info("cluster: initiating failover check for peers. active peers: ", activePeers)
	}
}

func (c *Cluster) electLeader() {
	// Increment the term (voting for myself in this term) and clear the leader
	c.fo.term++
	c.fo.leader = ""

	log.Info("cluster: leading new election for term ", c.fo.term)

	peerCount := len(c.peers)
	// Number of votes needed to elect the leader
	expectVotes := (peerCount+1)>>1 + 1
	done := make(chan *rpc.Call, peerCount)

	request := &VoteRequest{
		PeerValue: &PeerValue{
			PeerName: c.thisName,
			PeerAddr: c.listenOn,
		},
		Term: c.fo.term,
	}

	// Send async requests for votes to other peers
	for _, peer := range c.peers {
		response := VoteResponse{}
		peer.callAsync("Cluster.Vote", request, &response, done)
	}

	// Number of votes received (1 vote for self)
	voteCount := 1
	timeout := time.NewTimer(c.fo.heartBeat>>1 + c.fo.heartBeat)
	log.WithField("timeout", c.fo.heartBeat>>1+c.fo.heartBeat).
		WithField("expectVotes", expectVotes).
		WithField("failover term", c.fo.term).
		Debug("cluster: start another leader election")

	// Wait for one of the following
	// 1. More than half of the peers voting in favor
	// 2. All peers responded.
	// 3. Timeout.
	for i := 0; i < peerCount && voteCount < expectVotes; {
		select {
		case call := <-done:
			if call.Error == nil {
				if call.Reply.(*VoteResponse).Result {
					// Vote in my favor
					voteCount++
				} else if c.fo.term < call.Reply.(*VoteResponse).Term {
					// Vote against me. Abandon vote: this node's term is behind the cluster
					i = peerCount
					voteCount = 0
				}
			}

			i++
		case <-timeout.C:
			// break the loop
			i = peerCount
		}
	}

	if voteCount >= expectVotes {
		// Current node elected as the leader
		c.fo.leader = c.thisName
		log.WithField("my_term", c.fo.term).Info("Elected myself as a new leader")
	}
}
