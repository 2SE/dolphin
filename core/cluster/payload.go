package cluster

import (
	"encoding/ascii85"
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
	"hash/fnv"
	"sort"
	"sync"
)

type PktType uint8

const (
	RequestPktType PktType = iota + 1
	OnlinePktType
	OfflinePktType
)

// maybe protobuf 这里需要定义交互需要的协议
type RequestPkt struct {
	// 发送这条请求的节点名称
	PeerName string
	AppName  string
	Paths    []core.MethodPather
	PktType  PktType
	// Ring hash signature of the node sending this request
	// Signature must match the signature of the receiver, otherwise the
	// Cluster is desynchronized.
	Signature string
	Pkt       proto.Message
}

// maybe protobuf
type RespPkt struct {
	Code      int
	PeerValue *PeerValue // response from which peer
	Pkt       proto.Message
}

// --- 以下是内部请求消息体 ---

// voteReqResp is a vote request and a response in leader election.
type voteReqResp struct {
	req  *VoteRequest
	resp chan VoteResponse
}

// PingRequest 是主节点ping跟随节点时发送的数据
type PingRequest struct {
	// 主节点名称
	Leader string
	// 选举的轮次
	Term int
	// Ring hash signature that represents the cluster
	Signature string
	// 当前活动的集群名称集合
	PeerValues []*PeerValue
}

type PeerValue struct {
	PeerName string
	PeerAddr string
}

func (p *PeerValue) String() string {
	return p.PeerName
}

// ClusterVoteRequest is a request from a leader candidate to a node to vote for the candidate.
type VoteRequest struct {
	// 发出本次投票的候选节点名称
	PeerValue *PeerValue
	// 当前选举的轮次
	Term int
}

// ClusterVoteResponse is a vote from a node.
type VoteResponse struct {
	// 已对发出的投票请求投票
	Result bool
	// 节点投票后的轮次
	Term int
}

type digestPeerValue []*PeerValue

func (d digestPeerValue) Len() int           { return len(d) }
func (d digestPeerValue) Less(i, j int) bool { return d[i].PeerName < d[j].PeerName }
func (d digestPeerValue) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d digestPeerValue) Signature() string {
	sort.Sort(d)
	hash := fnv.New32a()
	var items []byte
	for _, item := range d {
		hash.Write([]byte(item.PeerName))
		items = append(items, byte(hash.Sum32()))
	}

	hash.Reset()
	b := make([]byte, 4)
	for i, item := range items {
		b[0] = byte(item)
		b[1] = byte(item >> 8)
		b[2] = byte(item >> 16)
		b[3] = byte(item >> 24)
		hash.Write(b)
		hash.Write([]byte(d[i].PeerName))
	}

	b = []byte{}
	b = hash.Sum(b)
	dst := make([]byte, ascii85.MaxEncodedLen(len(b)))
	ascii85.Encode(dst, b)
	return string(dst)
}

var pingPool = &sync.Pool{
	New: func() interface{} {
		return &PingRequest{}
	},
}

var reqPool = &sync.Pool{
	New: func() interface{} {
		return &RequestPkt{}
	},
}

var respPool = &sync.Pool{
	New: func() interface{} {
		return &RespPkt{}
	},
}
