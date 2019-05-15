package main

import (
	"github.com/2se/dolphin/pb"
	"github.com/google/uuid"
	"testing"
)

var (
	addr = "http://127.0.0.1:8080"
	req  = &pb.ClientComRequest{}
)

func getRequests(num int) []*pb.ClientComRequest {
	//v1Map["getUser"] = &route{Resource: "v1.0", Reversion: "user", Method: service.GetUser}
	res := make([]*pb.ClientComRequest, 0, num)
	for i := 0; i < num; i++ {
		res = append(res, &pb.ClientComRequest{
			TraceId: uuid.New().String(),
			Qid:     string(i),
			Id:      string(i),
			MethodPath: &pb.MethodPath{
				Revision: "v1.0",
				Action:   "getUser",
				Resource: "user",
			},
		})
	}
	return res
}

func sendRequest() {

}
func BenchmarkRequest(b *testing.B) {
	getRequests(1000000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {

	}
}
