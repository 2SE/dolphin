package ws

const (
	HeartBeatEquation = 3000
)

// WSMessage websocket的Message对象
type WSMessage struct {
	MsgType int
	MsgData []byte
}
