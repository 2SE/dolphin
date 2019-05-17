package main

import (
	"fmt"
	"github.com/2se/dolphin/pb"
	"github.com/gogo/protobuf/proto"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
)

var (
	addr = "127.0.0.1:8080"
)

func main() {
	conns := GetClients(1)
	for _, v := range conns {
		go func(conn *websocket.Conn) {
			req := getRequests(1000000)
			sendRequest(conn, req)
		}(v)
	}
	select {}
}
func getRequests(num int) (request chan []byte) {
	//v1Map["getUser"] = &route{Resource: "v1.0", Reversion: "user", Method: service.GetUser}
	request = make(chan []byte, 20)
	go func() {
		for i := 0; i < num; i++ {
			//p := &GetUserRequest{UserId:1}
			//obj:= ptypes.MarshalAny(p)
			req := &pb.ClientComRequest{
				TraceId: uuid.New().String(),
				Qid:     string(i),
				Id:      string(i),
				MethodPath: &pb.MethodPath{
					Revision: "v1.0",
					Action:   "getUser",
					Resource: "user",
				},
				FrontEnd: &pb.FrontEnd{
					Uuid: uuid.New().String(),
				},
			}
			buff, _ := proto.Marshal(req)
			request <- buff
		}
	}()
	return request
}

func sendRequest(conn *websocket.Conn, request <-chan []byte) {
	done := make(chan struct{}, 1)
	go func() {
		for i := 0; ; i++ {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			<-done
			res := new(pb.ServerComResponse)
			err = proto.Unmarshal(p, res)
			fmt.Println(i, res.Code)
		}
	}()
	for ch := range request {
		done <- struct{}{}
		err := conn.WriteMessage(websocket.BinaryMessage, ch)
		if err != nil {
			return
		}

	}
}
func GetClients(num int) []*websocket.Conn {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	conns := make([]*websocket.Conn, num)
	for i := 0; i < num; i++ {
		fmt.Println(u.String())
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		conns[i] = c
	}
	return conns
}
