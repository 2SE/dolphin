package config

import (
	"fmt"
	"time"
)

type Config struct {
	WsCnf       *WebsocketConfig `toml:"websocket"`
	ClusterCnf  *ClusterConfig   `toml:"cluster"`
	EventBusCnf *EventBusConfig  `toml:"event_bus"`
	PluginsCnf  []*PluginConfig  `toml:"plugins"`
}

type ClusterConfig struct {
	Self      string
	Connction *ClusterConnectionConfig
	Nodes     []*ClusterNodeConfig
	Failover  *ClusterFailoverConfig
}

type ClusterConnectionConfig struct {
	DialTimeout   duration
	MaxDelay      duration
	BaseDelay     duration
	Factor        float64
	Jitter        float64
	DisableTimeout bool     `toml:"disable_timeout"`
	WaitAfter     duration `toml:"wait_after"`
}

type ClusterNodeConfig struct {
	Name    string
	Address string
}

type ClusterFailoverConfig struct {
	Enabled       bool
	Heartbeat     duration
	VoteAfter     int
	NodeFailAfter int
}

type EventBusConfig struct {
	Listen   string
	KafkaCnf *KafkaConfig `toml:"kafka"`
}

type WebsocketConfig struct {
	Listen       string
	ReadBufSize  int
	WriteBufSize int
	Expvar       string
	Tls          *WsTlsConfig
}

type WsTlsConfig struct {
	Enabled      bool
	HTTPRedirect string `toml:"http_redirect"`
	CertFile     string `toml:"cert_file"`
	KeyFile      string `toml:"key_file"`
	Autocert     *WsAutoCertConfig
}

type WsAutoCertConfig struct {
	CertCache string `toml:"cert_cache"`
	Email     string
	Domains   []string
}

type KafkaConfig struct {
	Brokers      []string
	WaitWindow   duration `toml:"waitWindow"`
	ConsumeGroup string   `toml:"consumeGroup"`
	MinBytes     int      `toml:"minBytes"`
	MaxBytes     int      `toml:"maxBytes"`
	StartOffset  int      `toml:"startOffset"`
}

type PluginConfig struct {
	Enabled    bool
	Name       string
	ServerAddr string `toml:"server_addr"`
}

func (cnf *Config) String() string {
	if cnf.ClusterCnf != nil {
		return fmt.Sprintf("\n%s\n%s\n%s\n%s\n",
			cnf.WsCnf, cnf.ClusterCnf.String(), cnf.EventBusCnf.String(), cnf.PluginsCnf)
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

func (cnf *Config) GetEventBusConfig() *EventBusConfig {
	return cnf.EventBusCnf
}

func (cnf *Config) GetPluginConfigs() []*PluginConfig {
	return cnf.PluginsCnf
}

func (wscnf *WebsocketConfig) String() string {
	return fmt.Sprintf("[websocket] listen: %s, (read buf size): %d, (write buf size): %d, expvar: %s, tls: %s",
		wscnf.Listen, wscnf.ReadBufSize, wscnf.WriteBufSize, wscnf.Expvar, wscnf.Tls)
}

func (tcnf *WsTlsConfig) String() string {
	return fmt.Sprintf("\n [WS TLS](enabled): %v, (redirect http): %s, (cert file): %s, (key file): %s, \n[autocert]: %s",
		tcnf.Enabled, tcnf.HTTPRedirect, tcnf.CertFile, tcnf.KeyFile, tcnf.Autocert)
}

func (acc *WsAutoCertConfig) String() string {
	return fmt.Sprintf("  cache: %s, domains: %v, email: %s", acc.CertCache, acc.Domains, acc.Email)
}

func (ccnf *ClusterConfig) String() string {
	return fmt.Sprintf("\nCluster Config:\n Self(name): %s, \nConnection: %s \n Nodes(ClusterNodeConfig): %s\n Failover: %s",
		ccnf.Self, ccnf.Connction, ccnf.Nodes, ccnf.Failover)
}

func (cfc *ClusterFailoverConfig) String() string {
	return fmt.Sprintf("\n  failover enabled: %v, \n  heartbeat: %s, \n  vote_after: %d, \n  node_fail_after: %d",
		cfc.Enabled, cfc.Heartbeat, cfc.VoteAfter, cfc.NodeFailAfter)
}

func (cnc *ClusterNodeConfig) String() string {
	return fmt.Sprintf("\n{name: \"%s\", address: \"%s\"}", cnc.Name, cnc.Address)
}

func (ccc *ClusterConnectionConfig) String() string {
	return fmt.Sprintf("[connection] dial timeout: %s, (backoff) max delay: %s, " +
		"base delay: %s, factor: %f, jitter: %f, disable_timeout: %v, wait_after: %s",
		ccc.DialTimeout, ccc.MaxDelay, ccc.BaseDelay, ccc.Factor, ccc.Jitter, ccc.DisableTimeout, ccc.WaitAfter)
}

func (ebc *EventBusConfig) String() string {
	return fmt.Sprintf("[event bus] address: %s\n [kafka config]: %s\n", ebc.Listen, ebc.KafkaCnf)
}

func (kcnf *KafkaConfig) String() string {
	return fmt.Sprintf("brokers: %v, consume group: %s, min bytes: %d, max bytes: %d, start offset: %d, wait window: %s",
		kcnf.Brokers, kcnf.ConsumeGroup, kcnf.MinBytes,
		kcnf.MaxBytes, kcnf.StartOffset, kcnf.WaitWindow)
}

func (pcnf *PluginConfig) String() string {
	return fmt.Sprintf("\n[plugin] name: %s, enabled: %v, address: %s", pcnf.Name, pcnf.Enabled, pcnf.ServerAddr)
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
