package ws

import (
	"flag"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"
	_ "github.com/2se/dolphin/mock"
)

func TestListenAndServe(t *testing.T) {
	//p := Param{"test_client_id"}
	cfg := &config.WebsocketConfig{Listen: "127.0.0.1:8081", ReadBufSize: 1024, WriteBufSize: 1024, Expvar: "/debug/vars"}

	Init(cfg)
	// client
	reqData := &pb.ClientComRequest{
		Id:      "1",
		Qid:     "test_client_id",
		TraceId: "test_trace_id",
		Meta: &pb.ClientComMeta{
			Resource:     "user",
			Revision:     "v1",
			Action:       "getInfo",
			Subscription: false,
			Key:          "test_subscribe_key",
			Uuid:         "test_uuid",
		},
	}
	reqDataByte, _ := proto.Marshal(reqData)
	go client(reqDataByte)
	ListenAndServe(signalHandler())

}

func signalHandler() <-chan bool {

	stop := make(chan bool)

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		// Wait for a signal. Don't care which signal it is
		sig := <-signchan
		log.Infof("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}

func client(data []byte) {
	var addr = flag.String("addr", "127.0.0.1:8081", "http service address")
	u := url.URL{Scheme: "ws", Host: *addr, Path:"/ws"}
	var conn *websocket.Conn

	for {
		c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
		if c != nil {
			log.Println("connect success to server:", *addr)
			conn = c
			break
		}
	}


	err := conn.WriteMessage(websocket.TextMessage, []byte(data))
	if err != nil {
		log.Error("send msg error", err)
		return
	}

	//for {
	//	select {
	//	case <-done:
	//		return
	//	case t := <-ticker.C:
	//		err := wsutil.WriteClientMessage()
	//		if err != nil {
	//			log.Println("write:", err)
	//			return
	//		}
	//	case <-interrupt:
	//		log.Println("interrupt")
	//
	//		// Cleanly close the connection by sending a close message and then
	//		// waiting (with timeout) for the server to close the connection.
	//		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	//		if err != nil {
	//			log.Println("write close:", err)
	//			return
	//		}
	//		select {
	//		case <-done:
	//		case <-time.After(time.Second):
	//		}
	//		return
	//	}
	//}

}
