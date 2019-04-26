package main

import (
	"github.com/2se/dolphin/cluster"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/event"
	"github.com/2se/dolphin/outbox"
	"github.com/2se/dolphin/route"
	"github.com/2se/dolphin/routehttp"
	"github.com/2se/dolphin/ws"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
)

//go:generate protoc --proto_path=../../pb/ --go_out=plugins=grpc:../../pb/ ../../pb/appserve.proto

func main() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	app := newApp()
	app.Run(os.Args)
}

func newApp() (app *cli.App) {
	app = cli.NewApp()
	app.Version = DisplayVersion
	app.Name = DisplayName
	app.Usage = Usage
	app.UsageText = UsageText
	app.Description = DescriptionText
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

func run(cliCtx *cli.Context) error {
	setLogLevel(cliCtx)

	configPath := cliCtx.String(FlagConfigKey)
	log.Printf("in param config path: [%s]", configPath)
	cnf, err := config.Load(configPath)
	if err != nil || cnf == nil {
		log.Fatalf("failed to load config file. may be error here: %v or else config is nil", err)
	}
	log.Debugf("loaded config info: %s", cnf)

	pprof := cliCtx.String(FlagPprofKey)
	if len(pprof) > 0 {
		runPprof(pprof)
	}

	localName, err := cluster.Init(cnf.ClusterCnf)
	if err != nil {
		log.Fatalf("failed to initial cluster. cause: %v", err)
	}
	//init route
	router := route.InitRoute(cluster.Name(), cnf.RouteCnf)
	//init event emitter
	emiter := event.NewEmitter(256)
	//init kafka consumers to push message into event
	outbox.ConsumersInit(cnf.KafkaCnf, emiter)

	cluster.Start(router)
	defer cluster.Shutdown()
	go routehttp.Start(localName, cnf.RouteHttpCnf.Address, cluster.Notify)

	ws.Init(cnf.WsCnf)
	if err = ws.ListenAndServe(signalHandler()); err != nil {
		log.Errorf("%v\n", err)
	}

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
