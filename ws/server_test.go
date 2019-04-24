package ws

import (
	"context"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/route"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestListenAndServe(t *testing.T) {
	//p := Param{"test_client_id"}
	cfg := &config.WebsocketConfig{Listen: "127.0.0.1:8081", ReadBufSize: 1024, WriteBufSize: 1024, Expvar: "/debug/vars"}

	Init(cfg)
	// client
	metaData := &route.ClientComMeta{Resource: "user", Revision: "v1", Action: "getInfo", Subscription: false, Key: "subscirbe_key", Uuid: "test_uuid"}
	metaDataByte, _ := proto.Marshal(metaData)
	go client(metaDataByte)
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
	var conn net.Conn
	for {
		dialer := ws.DefaultDialer

		conn, _, _, _ = dialer.Dial(context.Background(), "ws://127.0.0.1:8081/ws?client_id=test_client_id")
		if conn != nil {
			log.Println("connect success to server:", "127.0.0.1:8081")
			break
		}
	}

	err := wsutil.WriteClientMessage(conn, ws.OpBinary, data)
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
