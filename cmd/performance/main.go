package main

import (
	"fmt"
	"github.com/2se/dolphin/cmd/performance/userpb"
	"github.com/2se/dolphin/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"

	"github.com/gorilla/websocket"
	"log"
	"net/url"
)

var (
	addr = "192.168.10.169:8080"
)

func main() {
	conns := GetClients(20)
	for _, v := range conns {
		go func(conn *websocket.Conn) {
			req := getRequests(10000)
			sendRequest(conn, req)
		}(v)
	}
	select {}
}

//	v1Map["getUser"] = &route{Resource: "user", Reversion: "v1.0", Method: service.GetUser}
func getRequests(num int) (request chan []byte) {
	//v1Map["getUser"] = &route{Resource: "v1.0", Reversion: "user", Method: service.GetUser}
	request = make(chan []byte, 20)
	go func() {
		for i := 0; i < num; i++ {
			p := &userpb.GetUserRequest{UserId: 1}
			obj, _ := ptypes.MarshalAny(p)
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
				Params: obj,
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
