package main

import (
	"github.com/2se/dolphin/cluster"
	"github.com/2se/dolphin/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
)

const (
	ConfigUsage   = "Path to config file. if empty, will finding in ${HOME}/.config/dolphin.d/config.toml and /etc/dolphin.d/config.toml"
	FlagConfigKey = "config"
	FlagPprofKey  = "pprof"
	FlagLoglvlKey = "loglvl"

	PprofUsage  = "File name to save profiling info to. Disabled if not set."
	LoglvlUsage = "debug,info,warn"
)

func main() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	app := newApp()
	app.Run(os.Args)
}

func newApp() (app *cli.App) {
	app = cli.NewApp()
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "c,config",
			Usage: ConfigUsage,
		},

		cli.StringFlag{
			Name:  FlagPprofKey,
			Usage: PprofUsage,
		},

		cli.StringFlag{
			Name:  FlagLoglvlKey,
			Usage: LoglvlUsage,
		},
	}
	return app
}

func run(context *cli.Context) error {
	setLogLevel(context)

	configPath := context.String(FlagConfigKey)
	cnf, err := config.Load(configPath)
	log.Infof("%s", cnf)
	if err != nil || cnf == nil {
		log.Fatalf("failed to load config file. may be error here: %v or else config is nil", err)
	}

	pprof := context.String(FlagPprofKey)
	if len(pprof) > 0 {
		runPprof(pprof)
	}

	_, err = cluster.Init(cnf.GetClusterConfig())
	if err != nil {
		log.Fatalf("初始化集群出错! %v", err)
	}

	cluster.Start()
	<-signalHandler()
	cluster.Shutdown()

	return nil
}

func runPprof(pprofFile string) {
	log.Infof("pprof enabled. and it is path: %s", pprofFile)
	var err error

	cpuf, err := os.Create(pprofFile + ".cpu")
	if err != nil {
		log.Fatal("Failed to create CPU pprof file: ", err)
	}
	defer cpuf.Close()

	memf, err := os.Create(pprofFile + ".mem")
	if err != nil {
		log.Fatal("Failed to create Mem pprof file: ", err)
	}
	defer memf.Close()

	pprof.StartCPUProfile(cpuf)
	defer pprof.StopCPUProfile()
	defer pprof.WriteHeapProfile(memf)

	log.Infof("Profiling info saved to '%s.(cpu|mem)'", pprofFile)
}

func setLogLevel(cliCtx *cli.Context) {
	lglvl := cliCtx.String(FlagLoglvlKey)
	switch strings.ToLower(lglvl) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func signalHandler() <-chan bool {
	stop := make(chan bool)

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		// Wait for a signal. Don't care which signal it is
		sig := <-signchan
		log.Infof("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}
