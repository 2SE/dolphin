package main

import (
	"fmt"
	"os"
	"path"
)

const (
	Major   = 0
	Minor   = 1
	PackVer = 0

	Usage       = "distributed api gateway"
	UsageText   = "dolphin [-c|--config value] [--pprof value] [--loglvl value]"
	ConfigUsage = "Path to config file. if empty, will finding in ${HOME}/.config/dolphin.d/config.toml and /etc/dolphin.d/config.toml"
	PprofUsage  = "File name to save profiling info to. Disabled if not set."
	LoglvlUsage = "debug,info,warn"

	FlagConfigKey = "config"
	FlagPprofKey  = "pprof"
	FlagLoglvlKey = "loglvl"
)

/// go build -ldflags "-X main.VERSION=`date -u +.%Y%m%d.%H%M%S`" main.go
var (
	STAGE     = "dev"
	VERSION   string
	GITCOMMIT string

	DisplayVersion  = fmt.Sprintf("%d.%d.%d", Major, Minor, PackVer)
	DisplayName     = path.Base(os.Args[0])
	DescriptionText = fmt.Sprintf("internal version: %s-%s-%s", STAGE, VERSION, GITCOMMIT)
)
