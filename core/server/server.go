package server

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	tw "github.com/RussellLuo/timingwheel"
	ws "github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	idleSessionTimeout = 55 * time.Second
	writeWait          = 10 * time.Second
	defaultQueueSize   = 128
	defaultBufferSize  = 1024
	defaultIDSalt      = "2se.com"
)

var (
	NilConnErr             = errors.New("nil websocket connection")
	NilDispatcherErr       = errors.New("nil dispatcher, need invoke server.Init(dispatcher)")
	TimeoutErr             = errors.New("time out")
	WriteInClosedWriterErr = errors.New("writer has been already closed")

	server     *http.Server
	wsUpgrader *ws.Upgrader
	opt        *Opt
)

func newSession(conn *ws.Conn) (io.WriteCloser, error) {
	return NewSession(conn, opt)
}

func Init(cnf *config.WebsocketConfig, dispatcher core.HubDispatcher, ticker *tw.TimingWheel) {
	opt = &Opt{
		Tls:                cnf.Tls,
		HubDispatcher:      dispatcher,
		Ticker:             ticker,
		ReadBufferSize:     cnf.ReadBufSize,
		WriteBufferSize:    cnf.WriteBufSize,
		CheckOrigin:        func(*http.Request) bool { return true },
		IdleSessionTimeout: cnf.IdleSessionTimeout.Get(),
		WriteWait:          cnf.WriteWait.Get(),
		SessionQueueSize:   cnf.SessionQueueSize,
		QueueOutTimeout:    cnf.QueueOutTimeout.Get(),
		IDSalt:             cnf.IDSalt,
		SessionPool:        make(map[string]*session),
	}
	checkOpt(opt)
	wsUpgrader = &ws.Upgrader{
		ReadBufferSize:  opt.ReadBufferSize,
		WriteBufferSize: opt.WriteBufferSize,
		CheckOrigin:     opt.CheckOrigin,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	mux.HandleFunc("/", Serve404)

	server = &http.Server{
		Addr:    cnf.Listen,
		Handler: mux,
	}
	if cnf.Tls != nil && cnf.Tls.Enabled {
		tlsConfig, err := makeTls(cnf.Tls)
		if err != nil {
			log.WithError(err).Fatalln("server: cannot make tls")
		}

		server.TLSConfig = tlsConfig
	}
}

func ListenAndServe(stop <-chan bool) error {
	shuttingDown := false
	httpdone := make(chan bool)
	go func() {
		var err error
		if server.TLSConfig != nil {
			// If port is not specified, use default https port (443),
			// otherwise it will default to 80
			if server.Addr == "" {
				server.Addr = ":https"
			}

			if opt.Tls.HTTPRedirect != "" {
				log.Infof("Redirecting connections from HTTP at [%s] to HTTPS at [%s]",
					opt.Tls.HTTPRedirect, server.Addr)

			}
			// This is a second HTTP server listenning on a different port.
			go http.ListenAndServe(opt.Tls.HTTPRedirect, TlsRedirect(server.Addr))
			log.Infof("Listening for client HTTPS connections on [%s]", server.Addr)
			err = server.ListenAndServeTLS("", "")
		} else {
			log.Infof("Listening for client HTTP connections on [%s]", server.Addr)
			err = server.ListenAndServe()
		}
		if err != nil {
			if shuttingDown {
				log.Infof("HTTP server: stopped")
			} else {
				log.Infof("HTTP server: failed", err)
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
			if err := server.Shutdown(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				log.Infof("HTTP server failed to terminate gracefully", err)
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

func checkOpt(opt *Opt) {
	if opt.HubDispatcher == nil {
		panic(NilDispatcherErr)
	}

	if opt.IdleSessionTimeout <= 0 {
		opt.IdleSessionTimeout = idleSessionTimeout
		log.Warnf("idle session timeout value is incorrect, using default: %s", idleSessionTimeout)
	}
	opt.pongWait = opt.IdleSessionTimeout
	opt.pingPeriod = (opt.pongWait * 9) / 10
	log.WithField("pong wait", opt.pongWait).
		WithField("ping period", opt.pingPeriod).
		Info("set input options...")

	if opt.WriteWait <= 0 {
		opt.WriteWait = writeWait
		log.Warnf("write wait value is incorrect, using default: %s", writeWait)
	}

	if opt.SessionQueueSize <= 0 {
		opt.SessionQueueSize = defaultQueueSize
		log.Warnf("session queue size value is incorrect, using default: %d", defaultQueueSize)
	}

	if opt.QueueOutTimeout <= 0 {
		opt.QueueOutTimeout = time.Millisecond * 50
		log.Warnf("queue out timeout value is incorrect, using default: %s", opt.QueueOutTimeout)
	}

	if opt.ReadBufferSize <= 0 {
		opt.ReadBufferSize = defaultBufferSize
		log.Warnf("websocket read buffer size is incorrect, using default: %d", defaultBufferSize)
	}

	if opt.WriteBufferSize <= 0 {
		opt.WriteBufferSize = defaultBufferSize
		log.Warnf("websocket write buffer size is incorrect, using default: %d", defaultBufferSize)
	}

	if len(opt.IDSalt) == 0 {
		opt.IDSalt = defaultIDSalt
		log.Warnf("id salt value is incorrect, using default: %s", defaultIDSalt)
	}
}

type Opt struct {
	Tls *config.WsTlsConfig
	//Dispatcher         core.Dispatcher
	HubDispatcher      core.HubDispatcher //新加
	SessionPool        map[string]*session
	Ticker             *tw.TimingWheel
	IdleSessionTimeout time.Duration
	WriteWait          time.Duration
	SessionQueueSize   int
	QueueOutTimeout    time.Duration
	ReadBufferSize     int
	WriteBufferSize    int
	IDSalt             string
	CheckOrigin        func(*http.Request) bool

	sync.Mutex
	pongWait   time.Duration
	pingPeriod time.Duration
}

func (opt *Opt) RemoveSession(userId string) {
	opt.Lock()
	delete(opt.SessionPool, userId)
	opt.Unlock()
}
func (opt *Opt) SessionPoolAppend(userId string, sess *session) {
	opt.Lock()
	if opt.SessionPool[userId] != nil {
		opt.SessionPool[userId].closeWsSimple()
		delete(opt.SessionPool, userId)
	}
	opt.SessionPool[userId] = sess
	opt.Unlock()
}
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Errorf("Upgrade to the websocket failed.")
		http.Error(w, "upgrade to the websocket failed", http.StatusBadRequest)
		return
	}
	// TODO session store
	newSession(conn)
}

// Custom 404 response.
func Serve404(wrt http.ResponseWriter, req *http.Request) {
	wrt.Header().Set("Content-Type", "application/json; charset=utf-8")
	wrt.WriteHeader(http.StatusNotFound)
	// TODO
	//json.NewEncoder(wrt).Encode(
	//	&ServerComMessage{Ctrl: &MsgServerCtrl{
	//		Timestamp: time.Now().UTC().Round(time.Millisecond),
	//		Code:      http.StatusNotFound,
	//		Text:      "not found"}})
}

// Redirect HTTP requests to HTTPS
func TlsRedirect(toPort string) http.HandlerFunc {
	if toPort == ":443" || toPort == ":https" {
		toPort = ""
	} else if toPort != "" && toPort[:1] == ":" {
		// Strip leading colon. JoinHostPort will add it back.
		toPort = toPort[1:]
	}

	return func(wrt http.ResponseWriter, req *http.Request) {
		host, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			// If SplitHostPort has failed assume it's because :port part is missing.
			host = req.Host
		}

		target, _ := url.ParseRequestURI(req.RequestURI)
		target.Scheme = "https"

		// Ensure valid redirect target.
		if toPort != "" {
			// Replace the port number.
			target.Host = net.JoinHostPort(host, toPort)
		} else {
			target.Host = host
		}

		if target.Path == "" {
			target.Path = "/"
		}

		http.Redirect(wrt, req, target.String(), http.StatusTemporaryRedirect)
	}
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
