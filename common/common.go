package common

import (
	"bytes"
	"sort"
)

const spilt = '/'

type MethodPath interface {
	String() string
}
type PeerRouter interface {
	PeerName() string
	AppName() string
	Equals(pr PeerRouter) bool
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
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(m.Version)
	buffer.WriteRune(spilt)
	buffer.WriteString(m.Action)
	buffer.WriteRune(spilt)
	buffer.WriteString(m.Action)
	return buffer.String()
}

func NewPeerRouter(peerName, appName string) PeerRouter {
	return &peerRoute{peerName, appName}
}

type peerRoute [2]string

func (s peerRoute) PeerName() string {
	return s[0]

}
func (s peerRoute) AppName() string {
	return s[1]
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
func (s *PeerRouters) RemoveByApp(appName string) {
	for k, v := range *s {
		if v.AppName() == appName {
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
	return peerRoute{}, nil // ErrPeerNotFound
}
func (s peerRoute) Equals(pr PeerRouter) bool {
	return s == pr
}
