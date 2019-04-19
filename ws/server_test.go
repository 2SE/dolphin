package ws

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

var addr = flag.String("listen", ":9001", "addr to listen")

func TestListenAndServe(t *testing.T) {
	flag.Parse()
	wsServer := NewWsServer()
	http.HandleFunc("/ws", serveWebsocket)
	go wsServer.Start()

	ln, err := net.Listen("tcp", *addr)

	if err != nil {
		t.Error("server listen error", err)
		return
	}
	t.Logf("listening %s (%q)", ln.Addr(), *addr)

	var (
		s     = new(http.Server)
		serve = make(chan error, 1)
		sig   = make(chan os.Signal, 1)
	)
	signal.Notify(sig, syscall.SIGTERM)
	go func() { serve <- s.Serve(ln) }()

	select {
	case err := <-serve:
		t.Fatal(err)
	case sig := <-sig:
		const timeout = 5 * time.Second

		t.Logf("signal %q received; shutting down with %s timeout", sig, timeout)

		ctx, _ := context.WithTimeout(context.Background(), timeout)
		if err := s.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
