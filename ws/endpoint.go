package ws

import (
	"context"
	"crypto/tls"
	"github.com/2se/dolphin/config"
	dhttp "github.com/2se/dolphin/http"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"time"
)

var (
	Endpoint *WebsocketEndpoint
)

type WebsocketEndpoint struct {
	server       *http.Server
	httpRedirect string
}

func Init(cnf *config.WebsocketConfig) {
	// Set up HTTP server. Must use non-default mux because of expvar.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", serveWebsocket)
	mux.HandleFunc("/", dhttp.Serve404)

	server := &http.Server{
		Addr:    cnf.Listen,
		Handler: mux,
	}

	if cnf.Tls != nil && cnf.Tls.Enabled {
		tlsConfig, err := makeTls(cnf.Tls)
		if err != nil {
			log.Fatalln(err)
		}

		server.TLSConfig = tlsConfig
	}

	Endpoint = &WebsocketEndpoint{server: server}
}

func ListenAndServe(stop <-chan bool) error {
	shuttingDown := false

	httpdone := make(chan bool)

	go func() {
		var err error
		if Endpoint.server.TLSConfig != nil {
			// If port is not specified, use default https port (443),
			// otherwise it will default to 80
			if Endpoint.server.Addr == "" {
				Endpoint.server.Addr = ":https"
			}

			if Endpoint.httpRedirect != "" {
				log.Printf("Redirecting connections from HTTP at [%s] to HTTPS at [%s]",
					Endpoint.httpRedirect, Endpoint.server.Addr)

				// This is a second HTTP server listenning on a different port.
				go http.ListenAndServe(Endpoint.httpRedirect, dhttp.TlsRedirect(Endpoint.server.Addr))
			}

			log.Printf("Listening for client HTTPS connections on [%s]", Endpoint.server.Addr)
			err = Endpoint.server.ListenAndServeTLS("", "")
		} else {
			log.Printf("Listening for client HTTP connections on [%s]", Endpoint.server.Addr)
			err = Endpoint.server.ListenAndServe()
		}
		if err != nil {
			if shuttingDown {
				log.Println("HTTP server: stopped")
			} else {
				log.Println("HTTP server: failed", err)
			}
		}

		httpdone <- true
	}()

	// Wait for either a termination signal or an error
Loop:
	for {
		select {
		case <-stop:
			// Flip the flag that we are terminating and close the Accept-ing socket, so no new connections are possible.
			shuttingDown = true
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := Endpoint.server.Shutdown(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				log.Println("HTTP server failed to terminate gracefully", err)
			}

			// Wait for http server to stop Accept()-ing connections.
			<-httpdone
			cancel()
			break Loop
		case <-httpdone:
			break Loop
		}
	}
	return nil
}

func serveWebsocket(w http.ResponseWriter, req *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		log.Errorf("%v", err)
		// TODO http response
	}

	// TODO
	go func() {
		defer conn.Close()

		for {
			msg, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				// handle error
			}
			err = wsutil.WriteServerMessage(conn, op, msg)
			if err != nil {
				// handle error
			}
		}
	}()
}

func makeTls(cnf *config.WsTlsConfig) (*tls.Config, error) {
	if cnf.Autocert != nil {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(cnf.Autocert.Domains...),
			Cache:      autocert.DirCache(cnf.Autocert.CertCache),
			Email:      cnf.Autocert.Email,
		}

		return certManager.TLSConfig(), nil
	}

	// Otherwise try to use static keys.
	cert, err := tls.LoadX509KeyPair(cnf.CertFile, cnf.KeyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}
