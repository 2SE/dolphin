package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/2se/dolphin/cmd/serverexample/user"
	"github.com/2se/dolphin/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

type AppInfo struct {
	PeerName string
	AppName  string
	Address  string
	Methods  []*MP
}
type MP struct {
	Reversion string
	Resource  string
	Action    string
}

var (
	m     sync.Mutex
	id    int64 = 2
	users       = []*user.User{
		{UserId: 1, Age: 20, UserName: "Mr Cai", Face: "face1.jpg"},
		{UserId: 2, Age: 25, UserName: "Miss Cai", Face: "face2.jpg"},
	}
	appInfo     = new(AppInfo)
	v1Map       = make(map[string]*route)
	service     = &ExampleService{}
	dolphinAddr string
)

type route struct {
	Resource  string
	Reversion string
	Method    func(*pb.ClientComRequest) (*pb.ServerComResponse, error)
}

func getId() int64 {
	m.Lock()
	defer m.Unlock()
	id++
	return id
}
func init() {
	//总线服务地址
	dolphinAddr = "http://192.168.0.12:9527"
	appInfo.PeerName = ""
	appInfo.AppName = "app1"
	//本地服务地址
	appInfo.Address = "192.168.0.14:10086"
	appInfo.Methods = make([]*MP, 0, 3)

	v1Map["getUser"] = &route{Resource: "user", Reversion: "v1.0", Method: service.GetUser}
	v1Map["addUser"] = &route{Resource: "user", Reversion: "v1.0", Method: service.AddUser}
	v1Map["removeUser"] = &route{Resource: "user", Reversion: "v.10", Method: service.RmoveUser}

	for k, v := range v1Map {
		appInfo.Methods = append(appInfo.Methods, &MP{
			Reversion: v.Reversion,
			Resource:  v.Resource,
			Action:    k,
		})
	}
}

type ExampleService struct {
}

func handle(request *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	m, ok := v1Map[request.MethodPath.Action]
	if !ok {
		return nil, errors.New("action not found")
	}
	return m.Method(request)
}

func (service *ExampleService) Request(ctx context.Context, req *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return handle(req)
}

func (service *ExampleService) AddUser(c *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	req := &user.User{}
	err := ptypes.UnmarshalAny(c.Params, req)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("param error")
	}
	req.UserId = getId()
	users = append(users, req)
	object, err := ptypes.MarshalAny(req)
	if err != nil {
		return nil, err
	}
	return &pb.ServerComResponse{
		Code: 200,
		Body: object,
	}, nil
}
func (service *ExampleService) RmoveUser(c *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	req := &user.GetUserRequest{}
	err := ptypes.UnmarshalAny(c.Params, req)
	if err != nil {
		return nil, err
	}
	for k, v := range users {
		if v.UserId == req.UserId {
			if k == 0 {
				users = users[1:]
			} else if k == len(users) {
				users = users[:k-1]
			} else {
				users = append(users[:k], users[k+1:]...)
			}
			break
		}
	}
	return &pb.ServerComResponse{}, nil
}
func (service *ExampleService) GetUser(c *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	logrus.Info(c)
	req := &user.GetUserRequest{}
	err := ptypes.UnmarshalAny(c.Params, req)
	if err != nil {
		return nil, err
	}
	for _, v := range users {
		if v.UserId == req.UserId {
			object, err := ptypes.MarshalAny(v)
			if err != nil {
				return nil, err
			}
			return &pb.ServerComResponse{
				Code: 200,
				Body: object,
			}, nil
		}
	}
	return nil, errors.New("not found")
}

func main() {
	go func() {
		s1 := new(ExampleService)
		l, err := net.Listen("tcp", appInfo.Address)
		if err != nil {
			panic(fmt.Errorf("tpc listen err:%v ", err))
		}
		defer l.Close()
		svc := grpc.NewServer()
		pb.RegisterAppServeServer(svc, s1)
		if err := svc.Serve(l); err != nil {
			panic(fmt.Errorf("failed to serve: %v", err))
		}
	}()
	appJson, err := json.Marshal(appInfo)
	if err != nil {
		panic(fmt.Errorf("json marshal err:%s ", err.Error()))
	}

	resp, err := http.Post(dolphinAddr, "application/json; charset=utf-8", bytes.NewReader(appJson))
	if err != nil {
		panic(fmt.Errorf("Service registration failed err:%v ", err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("Service registration failed err:%v ", err))
	}
	fmt.Println(string(body))
	fmt.Print(string(appJson))
	select {}
}
