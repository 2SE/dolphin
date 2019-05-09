package config

import (
	"fmt"
	"time"
)

type Config struct {
	WsCnf        *WebsocketConfig `toml:"websocket"`
	ClusterCnf   *ClusterConfig   `toml:"cluster"`
	KafkaCnf     []*KafkaConfig   `toml:"kafkas"`
	PluginsCnf   []*PluginConfig  `toml:"plugins"`
	RouteCnf     *RouteConfig     `toml:"route""`
	RouteHttpCnf *RouteHttpConfig `toml:"routehttp"`
	SchedulerCnf *SchedulerConfig `toml:"scheduler"`
}

type ClusterConfig struct {
	Self       string                   `toml:"self"`
	Connection *ClusterConnectionConfig `toml:"connection"`
	Nodes      []*ClusterNodeConfig     `toml:"nodes"`
	Failover   *ClusterFailoverConfig   `toml:"failover"`
}

type ClusterConnectionConfig struct {
	DialTimeout       Duration `toml:"dial_timeout"`
	MaxDelay          Duration `toml:"max_delay"`
	BaseDelay         Duration `toml:"base_delay"`
	Factor            float64  `toml:"factor"`
	Jitter            float64  `toml:"jitter"`
	DisableReqTimeout bool     `toml:"disable_request_timeout"`
	ReqWaitAfter      Duration `toml:"request_wait_after"`
}

type ClusterNodeConfig struct {
	Name    string `toml:"name"`
	Address string `toml:"address"`
}

type ClusterFailoverConfig struct {
	Enabled       bool     `toml:"enabled"`
	Heartbeat     Duration `toml:"heartbeat"`
	VoteAfter     int      `toml:"vote_after"`
	NodeFailAfter int      `toml:"node_fail_after"`
}

type WebsocketConfig struct {
	Listen             string       `toml:"listen"`
	ReadBufSize        int          `toml:"read_buf_size"`
	WriteBufSize       int          `toml:"write_buf_size"`
	GrpcListen         string       `toml:"grpc_listen"`
	Expvar             string       `toml:"expvar"`
	Tls                *WsTlsConfig `toml:"tls"`
	WriteWait          Duration     `toml:"write_time_wait"`
	ReadWait           Duration     `toml:"read_time_wait"`
	IdleSessionTimeout Duration     `toml:"idle_session_timeout"`
	SessionQueueSize   int          `toml:"session_queue_size"`
	QueueOutTimeout    Duration     `toml:"queue_out_timeout"`
	IDSalt             string       `toml:"id_salt"`
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
	Brokers   []string `toml:"brokers"`
	Topic     string   `toml:"topic"`
	Offset    int64    `toml:"offset"`
	GroupID   string   `toml:"groupId"`
	Partition int      `toml:"partition"`
	MinBytes  int      `toml:"minBytes"`
	MaxBytes  int      `toml:"maxBytes"`
	MaxWait   Duration `toml:"maxWait"`
}

type PluginConfig struct {
	Enabled    bool   `toml:"enabled"`
	Name       string `toml:"name"`
	ServerAddr string `toml:"server_addr"`
}
type RouteHttpConfig struct {
	Address string `toml:"address"`
}
type SchedulerConfig struct {
	Address string `toml:"address"`
}
type RouteConfig struct {
	Recycle   Duration `toml:"recycle"`
	Threshold int16    `toml:"threshold"`
	Timeout   Duration `toml:"timeout"`
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

func (cnf *Config) GetKafkaConfig() []*KafkaConfig {
	return cnf.KafkaCnf
}

func (cnf *Config) GetPluginConfigs() []*PluginConfig {
	return cnf.PluginsCnf
}
func (cnf *Config) GetRouteConfig() *RouteConfig {
	return cnf.RouteCnf
}
func (cnf *Config) GetRouteHttpConfig() *RouteHttpConfig {
	return cnf.RouteHttpCnf
}

func (wscnf *WebsocketConfig) String() string {
	return fmt.Sprintf("[websocket]\nlisten: %s | read buffer size: %d | write buffer size: %d "+
		"| expvar: %s\nidle session timeout: %s | session queue size: %d | queue out timeout: %s\n%s\n",
		wscnf.Listen, wscnf.ReadBufSize, wscnf.WriteBufSize, wscnf.Expvar,
		wscnf.IdleSessionTimeout, wscnf.SessionQueueSize, wscnf.QueueOutTimeout, wscnf.Tls)
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
	return fmt.Sprintf("\n[kafka]\nbrokers: %v\n"+
		"topic:%s\n"+
		"consume group: %s\n"+
		"partition: %d\n"+
		"min bytes: %d\n"+
		"max bytes: %d\n"+
		"start offset: %d\n"+
		"wait window: %s",
		kcnf.Brokers, kcnf.Topic, kcnf.GroupID,
		kcnf.Partition, kcnf.MinBytes,
		kcnf.MaxBytes, kcnf.Offset, kcnf.MaxWait)
}

func (pcnf *PluginConfig) String() string {
	return fmt.Sprintf("\n[%s]\nenabled: %v\naddress: %s", pcnf.Name, pcnf.Enabled, pcnf.ServerAddr)
}

type Duration struct {
	time.Duration
}

func (d *Duration) Get() time.Duration {
	return d.Duration
}

func (d *Duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
