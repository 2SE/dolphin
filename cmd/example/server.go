package main

import (
	"context"
	"github.com/2se/dolphin/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/url"
)

type ExampleService struct {
}

func (service *ExampleService) Request(context.Context, *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	return nil, nil
}

type ExampleService2 struct {
}

func (service *ExampleService2) Request(context.Context, *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	return nil, nil
}
func main() {
	data := make(url.Values)
	data.Set("name", "ok")
	//resp, err := http.PostForm("http://127.0.0.1:16071", data)

	go S1Start()
	go S2Start()
	select {}
}
func S1Start() {
	s1 := new(ExampleService)
	l, err := net.Listen("tcp", "127.0.0.1:16012")
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}
	defer l.Close()
	svc := grpc.NewServer()
	pb.RegisterAppServeServer(svc, s1)
	if err := svc.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
func S2Start() {
	s2 := new(ExampleService2)
	l, err := net.Listen("tcp", "127.0.0.1:16013")
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}
	defer l.Close()
	svc := grpc.NewServer()
	pb.RegisterAppServeServer(svc, s2)
	if err := svc.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
