package mock

import (
	"github.com/2se/dolphin/core"
	"github.com/golang/protobuf/proto"
)

var (
	//用于引用cluster包，但是执行逻辑中不依赖cluster的模拟测试数据
	MockCluster = new(mockCluster)
)

type mockCluster struct {
}

func (*mockCluster) Name() string {
	return "mock"
}
func (*mockCluster) SetRouter(core.Router) {

}
func (*mockCluster) Notify(core.PeerRouter, ...core.MethodPath) {

}
func (*mockCluster) Request(core.PeerRouter, proto.Message) (proto.Message, error) {
	return nil, nil
}
