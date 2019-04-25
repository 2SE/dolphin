package main

import (
	"context"
	"github.com/2se/dolphin/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/url"
)

type ExampleService struct {
}

func (service *ExampleService) Request(context.Context, *pb.ClientComRequest) (*pb.ServerComResponse, error) {
	return nil, nil
}

func main() {
	data := make(url.Values)
	data.Set("name", "ok")
	resp, err := http.PostForm("http://127.0.0.1:16071", data)
	l, err := net.Listen("tcp", "127.0.0.1:16012")
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}
	defer l.Close()

	svc := grpc.NewServer()
	pb.RegisterAppServeServer(svc, &ExampleService{})

	svc.Serve(l)
}
