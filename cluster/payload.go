package cluster

// maybe protobuf 这里需要定义交互需要的协议
type RequestPkt struct {
	// 发送这条请求的节点名称
	Node string
	// 一致性哈希算法签名
	// Ring hash signature of the node sending this request
	// Signature must match the signature of the receiver, otherwise the
	// Cluster is desynchronized.
	Signature string

	Pkt interface{}

	// 超级用户可以发送消息代表其他用户
	OnBehalfOf string

	// Expanded (routable) topic name
	RcptTo string
	// Originating session
	//Sess *ClusterSess
	// True if the original session has disconnected
	SessGone bool
}

// maybe protobuf
type RespPkt struct {
	FromSID string
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
	Nodes []string
}

// ClusterVoteRequest is a request from a leader candidate to a node to vote for the candidate.
type VoteRequest struct {
	// 发出本次投票的候选节点名称
	Node string
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
