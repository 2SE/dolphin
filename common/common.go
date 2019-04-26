package common

import (
	"bytes"
	"errors"
	"sort"
)

var (
	ErrPeerNotFound = errors.New("common: peer not found")
)

const spilt = '/'

type MethodPath interface {
	String() string
}
type PeerRouter interface {
	PeerName() string
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
		buffer.WriteString(m.Action)
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
