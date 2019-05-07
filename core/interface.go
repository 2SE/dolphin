package core

import (
	"bytes"
	"errors"
	"github.com/golang/protobuf/proto"
	"io"
	"sort"
)

var (
	ErrPeerNotFound = errors.New("core: peer not found")
)

const (
	spilt = '/'
)

type LocalPeer interface {
	Name() string
	SetRouter(Router)
	Notify(PeerRouter, ...MethodPath)
	Request(PeerRouter, proto.Message) (proto.Message, error)
}

type Router interface {
	//获取路由分流指向
	RouteIn(mp MethodPath, id string, request proto.Message) (response proto.Message, err error)
	//根据peerRouter将request定向到指定Grpc服务并返回结果
	RouteOut(pr PeerRouter, request proto.Message) (response proto.Message, err error)
	//注册单个
	//appName 资源服务名称
	//peer 节点名称（空字符串为本地）
	//address  资源服务连接地址（ps:www.example.com:8080）
	Register(mps []MethodPath, pr PeerRouter, address string) error
	//注销app下所有
	UnRegisterApp(pr PeerRouter)
	//注销peer下所有
	UnRegisterPeer(peerName string)
}

type MethodPath interface {
	String() string
}
type PeerRouter interface {
	PeerName() string
	SetPeerName(name string)
	AppName() string
	Equals(pr PeerRouter) bool
	String() string
}
type PeerRouters []PeerRouter

func NewMethodPath(version, resource, action string) MethodPath {
	return &methodPath{Version: version, Resource: resource, Action: action}
}

type methodPath struct {
	Version  string
	Resource string
	Action   string
	buff     string
}

func (m *methodPath) String() string {
	if m.buff == "" {
		buffer := bytes.NewBuffer(nil)
		buffer.WriteString(m.Version)
		buffer.WriteRune(spilt)
		buffer.WriteString(m.Resource)
		buffer.WriteRune(spilt)
		buffer.WriteString(m.Action)
		m.buff = buffer.String()
	}
	return m.buff
}

func NewPeerRouter(peerName, appName string) PeerRouter {
	return &peerRoute{peer: peerName, app: appName}
}

type peerRoute struct {
	peer string
	app  string
	str  string
}

func (s *peerRoute) PeerName() string {
	return s.peer
}
func (s *peerRoute) SetPeerName(name string) {
	s.peer = name
}
func (s *peerRoute) AppName() string {
	return s.app
}
func (s *peerRoute) Equals(pr PeerRouter) bool {
	return s.peer == pr.PeerName() && s.app == pr.AppName()
}
func (s *peerRoute) String() string {
	if s.str == "" {
		buffer := bytes.NewBuffer(nil)
		buffer.WriteString(s.peer)
		buffer.WriteRune(spilt)
		buffer.WriteString(s.app)
		s.str = buffer.String()
	}
	return s.str
}

func (s PeerRouters) Len() int      { return len(s) }
func (s PeerRouters) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PeerRouters) Less(i, j int) bool {
	return s[i].PeerName() < s[j].PeerName()
}
func (s *PeerRouters) Append(pr PeerRouter) {
	*s = append(*s, pr)
}
func (s *PeerRouters) Sort() {
	sort.Sort(s)
}
func (s *PeerRouters) RemoveByPeer(peerName string) {
	for k, v := range *s {
		if v.PeerName() == peerName {
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
func (s *PeerRouters) RemoveByPeerRouter(pr PeerRouter) {
	for k, v := range *s {
		if v.Equals(pr) {
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
func (s PeerRouters) FindOne(peerName string) (PeerRouter, error) {
	for _, v := range s {
		if v.PeerName() == peerName {
			return v, nil
		}
	}
	return nil, ErrPeerNotFound
}

type Request []byte

func (req Request) Unmarshal(model proto.Message) error {
	return proto.Unmarshal(req, model)
}

func Marshal(model proto.Message) (req Request, err error) {
	req, err = proto.Marshal(model)
	return
}

type Session interface {
	io.WriteCloser
	GetID() string
	Send(message proto.Message) error
}

type Hub interface {
	Start() error
	Stop() error
	Subscribe(ssid string, sub Subscriber) (*Subscription, error)
	UnSubscribe(subscription *Subscription)
	Publish(kv *KV)
}

type Dispatcher interface {
	Dispatch(sess Session, req Request)
}

type HubDispatcher interface {
	Hub
	Dispatcher
}

type Subscriber = Session

type Subscription struct {
	Ssid string
	Sub  Subscriber
}

type KV struct {
	Key []byte
	Val []byte
}
