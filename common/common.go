package common

import "bytes"

const spilt = '/'

type MethodPath interface {
	String() string
}

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
