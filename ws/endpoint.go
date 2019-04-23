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
	W = NewWsServer()
	go W.Start()
	go W.HandleHeartBeat(HeartBeatEquation)
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
	var p Param
	var cli *Client
	err := ParamBind(&p, req)
	if err != nil {
		log.Errorln("Ws: get client_id error", err)
		// todo http response
	}

	if p.ClientID == "" {
		log.Errorf("cannot find the client by nil")
		// todo http response
	}
	if _, ok := W.Clients[p.ClientID]; ok {
		cli = W.Clients[p.ClientID]
	} else {
		conn, _, _, err := ws.UpgradeHTTP(req, w)
		if err != nil {
			log.Errorf("%v", err)
			// TODO http response
		}
		cli = &Client{conn: &conn, ID: p.ClientID}
		W.AddCli <- cli
	}

	go func() {
		for {
			conn := *cli.conn
			msg, opCode, err := wsutil.ReadClientData(conn)
			if err != nil {
				// todo handle error
				log.Error("Ws: read client data error", err)
			}

			switch opCode {
			case ws.OpClose:
				conn.Close()
				W.DelCli <- cli
			case ws.OpPing:
				wsutil.WriteServerMessage(conn, ws.OpPong, nil)
				cli.Timestamp = time.Now().UnixNano() / 1e6
			default:
				cli.Timestamp = time.Now().UnixNano() / 1e6
			}

			log.Printf("got msg from [%s], message is : %v", p.ClientID, string(msg))
			W.handleClientData(cli, msg)
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
