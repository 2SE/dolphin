package route

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func (p *resourcesPool) TryAddClient(address string) error {
	if _, ok := p.clients[address]; ok {
		return ErrClientExists
	}
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	appCli := NewAppServeClient(conn)
	p.clients[address] = appCli
	return nil
}

func (p *resourcesPool) RemoveClient(address string) {
	delete(p.clients, address)
}

func (p *resourcesPool) callAppAction(address string, request *ClientComRequest) (*ServerComResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return p.clients[address].Request(ctx, request)
}
