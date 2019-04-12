package config

import (
	"fmt"
	"time"
)

type Config struct {
	ClusterCnf  *ClusterConfig  `toml:"cluster"`
	EventBusCnf *EventBusConfig `toml:"event_bus"`
}

type ClusterConfig struct {
	Self     string
	Nodes    []*ClusterNodeConfig
	Failover *ClusterFailoverConfig
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
	Address string
}

func (cnf *Config) String() string {
	if cnf.ClusterCnf != nil {
		return fmt.Sprintf("%s\n%s",
			cnf.ClusterCnf.String(),
			cnf.EventBusCnf.String(),
		)
	} else {
		return "-"
	}
}

func (cnf *Config) GetClusterConfig() *ClusterConfig {
	return cnf.ClusterCnf
}

func (ccnf *ClusterConfig) String() string {
	return fmt.Sprintf("\nCluster Config:\n Self(name): %s, \n Nodes(ClusterNodeConfig): %s\n Failover: %s",
		ccnf.Self,
		ccnf.Nodes,
		ccnf.Failover,
	)
}

func (cfc *ClusterFailoverConfig) String() string {
	return fmt.Sprintf("\n  failover enabled: %v, \n  heartbeat: %s, \n  vote_after: %d, \n  node_fail_after: %d",
		cfc.Enabled,
		cfc.Heartbeat,
		cfc.VoteAfter,
		cfc.NodeFailAfter,
	)
}

func (cnc *ClusterNodeConfig) String() string {
	return fmt.Sprintf("\n{name: \"%s\", address: \"%s\"}", cnc.Name, cnc.Address)
}

func (ebc *EventBusConfig) String() string {
	return fmt.Sprintf("[event bus] address: %s", ebc.Address)
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
