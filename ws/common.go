package ws

import "time"

const (
	HeartBeatEquation = 3000
	ReadBufferSize = 1024
	WriteBufferSize = 1024
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

)

// websocket的Message对象
type WSMessage struct {
	MsgType int
	MsgData []byte
}