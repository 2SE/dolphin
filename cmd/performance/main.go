package main

import (
	"fmt"
	"github.com/2se/dolphin/cmd/serverexample/user"
	"github.com/2se/dolphin/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
)

var (
	//dolphin websocket地址
	//addr = "106.12.54.87:8080"
	addr = "192.168.9.130:8080"
)

func main() {
	conns := GetClients(10000) //设置生成客户端数量
	for v := range conns {
		if v != nil {
			go func(conn *websocket.Conn) {
				req := getRequests(1000) //设置单个客户端串行请求次数
				sendRequest(conn, req)
			}(v)
		}
	}
	//testPing()
	select {}
}
func testPing() {
	conn := GetClients(1)
	cli := <-conn
	for {

		typ, p, err := cli.ReadMessage()
		if err != nil {
			fmt.Println(typ)
			if typ == 9 {
				fmt.Println("ping get", string(p))
				cli.WriteMessage(10, []byte{})
			}
		}
	}
}

//[{"Reversion":"v1.0","Resource":"user","Action":"getUser"},
// {"Reversion":"v1.0","Resource":"user","Action":"addUser"},
// {"Reversion":"v.10","Resource":"user","Action":"removeUser"}]}
func getRequests(num int) (request chan []byte) {
	request = make(chan []byte, 20)
	go func() {
		for i := 0; i < num; i++ {
			/*p := &user.LoginReq{
				DeviceId:  "444444",
				LoginType: 1,
				SmsCode:   "123456",
				UserName:  "15903636764",
			}*/
			p := &user.GetUserRequest{
				UserId: 1,
			}
			obj, _ := ptypes.MarshalAny(p)
			req := &pb.ClientComRequest{
				TraceId: uuid.New().String(),
				Qid:     string(i),
				Id:      string(i),
				MethodPath: &pb.MethodPath{
					Revision: "v1",
					Action:   "GetUser",
					Resource: "Example",
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
			if err != nil {
				fmt.Println(err)
			} else {
				if res.Code != 200 {
					fmt.Println(res)
				} else {

				}
			}
			/*pmu := &pb.LoginResponse{}
			err = ptypes.UnmarshalAny(res.Body, pmu)
			if err != nil {
				fmt.Println("unmarshalAny=>err:", err)
				return
			}
			lrp := new(user.LoginRes)
			err = ptypes.UnmarshalAny(pmu.Params, lrp)
			if err != nil {
				fmt.Println("unmarshalAny inner=>err:", err)
			}
			fmt.Println("login value ", lrp.String())
			*/
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
func GetClients(num int) chan *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	conns := make(chan *websocket.Conn, num)
	go func() {
		for i := 0; i < num; i++ {
			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err == nil {
				conns <- c
			}
		}
	}()
	return conns
}
