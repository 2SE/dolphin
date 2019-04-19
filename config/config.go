package config

import (
	"fmt"
	"time"
)

type Config struct {
	WsCnf      *WebsocketConfig `toml:"websocket"`
	ClusterCnf *ClusterConfig   `toml:"cluster"`
	KafkaCnf   *KafkaConfig     `toml:"kafka"`
	PluginsCnf []*PluginConfig  `toml:"plugins"`
}

type ClusterConfig struct {
	Self       string                   `toml:"self"`
	Connection *ClusterConnectionConfig `toml:"connection"`
	Nodes      []*ClusterNodeConfig     `toml:"nodes"`
	Failover   *ClusterFailoverConfig   `toml:"failover"`
}

type ClusterConnectionConfig struct {
	DialTimeout       duration `toml:"dial_timeout"`
	MaxDelay          duration `toml:"max_delay"`
	BaseDelay         duration `toml:"base_delay"`
	Factor            float64  `toml:"factor"`
	Jitter            float64  `toml:"jitter"`
	DisableReqTimeout bool     `toml:"disable_request_timeout"`
	ReqWaitAfter      duration `toml:"request_wait_after"`
}

type ClusterNodeConfig struct {
	Name    string `toml:"name"`
	Address string `toml:"address"`
}

type ClusterFailoverConfig struct {
	Enabled       bool     `toml:"enabled"`
	Heartbeat     duration `toml:"heartbeat"`
	VoteAfter     int      `toml:"vote_after"`
	NodeFailAfter int      `toml:"node_fail_after"`
}

type WebsocketConfig struct {
	Listen       string       `toml:"listen"`
	ReadBufSize  int          `toml:"read_buf_size"`
	WriteBufSize int          `toml:"write_buf_size"`
	GrpcListen   string       `toml:"grpc_listen"`
	Expvar       string       `toml:"expvar"`
	Tls          *WsTlsConfig `toml:"tls"`
}

type WsTlsConfig struct {
	Enabled      bool              `toml:"enabled"`
	HTTPRedirect string            `toml:"http_redirect"`
	CertFile     string            `toml:"cert_file"`
	KeyFile      string            `toml:"key_file"`
	Autocert     *WsAutoCertConfig `toml:"autocert"`
}

type WsAutoCertConfig struct {
	CertCache string   `toml:"cert_cache"`
	Email     string   `toml:"email"`
	Domains   []string `toml:"domains"`
}

type KafkaConfig struct {
	Brokers      []string `toml:"brokers"`
	WaitWindow   duration `toml:"waitWindow"`
	ConsumeGroup string   `toml:"consumeGroup"`
	MinBytes     int      `toml:"minBytes"`
	MaxBytes     int      `toml:"maxBytes"`
	StartOffset  int      `toml:"startOffset"`
}

type PluginConfig struct {
	Enabled    bool   `toml:"enabled"`
	Name       string `toml:"name"`
	ServerAddr string `toml:"server_addr"`
}

func (cnf *Config) String() string {
	if cnf.ClusterCnf != nil {
		return fmt.Sprintf("\n%s\n%s\n%s\n\n[plugin]: %s\n",
			cnf.WsCnf, cnf.ClusterCnf, cnf.KafkaCnf, cnf.PluginsCnf)
	} else {
		return "-"
	}
}

func (cnf *Config) GetWebsocketConfig() *WebsocketConfig {
	return cnf.WsCnf
}

func (cnf *Config) GetClusterConfig() *ClusterConfig {
	return cnf.ClusterCnf
}

func (cnf *Config) GetKafkaConfig() *KafkaConfig {
	return cnf.KafkaCnf
}

func (cnf *Config) GetPluginConfigs() []*PluginConfig {
	return cnf.PluginsCnf
}

func (wscnf *WebsocketConfig) String() string {
	return fmt.Sprintf("[websocket]\nlisten: %s | read buffer size: %d | write buffer size: %d | expvar: %s\n%s",
		wscnf.Listen, wscnf.ReadBufSize, wscnf.WriteBufSize, wscnf.Expvar, wscnf.Tls)
}

func (tcnf *WsTlsConfig) String() string {
	return fmt.Sprintf("\n[websocket.tls]\nenabled: %v | redirect http: %s\ncert file: %s\nkey file: %s\n%s",
		tcnf.Enabled, tcnf.HTTPRedirect, tcnf.CertFile, tcnf.KeyFile, tcnf.Autocert)
}

func (acc *WsAutoCertConfig) String() string {
	return fmt.Sprintf("\n[websocket.tls.autocert]\ncache: %s\ndomains: %v\nemail: %s", acc.CertCache, acc.Domains, acc.Email)
}

func (ccnf *ClusterConfig) String() string {
	return fmt.Sprintf("\n[cluster]\nself name: \"%s\"\n%s\n[cluster.nodes]: %s\n%s",
		ccnf.Self, ccnf.Connection, ccnf.Nodes, ccnf.Failover)
}

func (cfc *ClusterFailoverConfig) String() string {
	return fmt.Sprintf("\n[cluster.failover]\nfailover enabled: %v | heartbeat: %s | vote_after: %d | node_fail_after: %d",
		cfc.Enabled, cfc.Heartbeat, cfc.VoteAfter, cfc.NodeFailAfter)
}

func (cnc *ClusterNodeConfig) String() string {
	return fmt.Sprintf("\n{name: \"%s\", address: \"%s\"}", cnc.Name, cnc.Address)
}

func (ccc *ClusterConnectionConfig) String() string {
	return fmt.Sprintf("[cluster.connection]\ndial timeout: %s | (backoff)max delay: %s"+
		" | base delay: %s | factor: %f | jitter: %f\n(net/rpc dial)disable_timeout: %v | wait_after: %s",
		ccc.DialTimeout, ccc.MaxDelay, ccc.BaseDelay, ccc.Factor, ccc.Jitter, ccc.DisableReqTimeout, ccc.ReqWaitAfter)
}

func (kcnf *KafkaConfig) String() string {
	return fmt.Sprintf("\n[kafka]\nbrokers: %v\nconsume group: %s\nmin bytes: %d\nmax bytes: %d\nstart offset: %d\nwait window: %s",
		kcnf.Brokers, kcnf.ConsumeGroup, kcnf.MinBytes,
		kcnf.MaxBytes, kcnf.StartOffset, kcnf.WaitWindow)
}

func (pcnf *PluginConfig) String() string {
	return fmt.Sprintf("\n[%s]\nenabled: %v\naddress: %s", pcnf.Name, pcnf.Enabled, pcnf.ServerAddr)
}

type duration struct {
	time.Duration
}

func (d *duration) Get() time.Duration {
	return d.Duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
