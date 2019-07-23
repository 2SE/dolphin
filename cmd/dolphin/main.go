package main

import (
	"fmt"
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core"
	"github.com/2se/dolphin/core/cluster"
	"github.com/2se/dolphin/core/dispatcher"
	"github.com/2se/dolphin/core/outbox"
	"github.com/2se/dolphin/core/router"
	"github.com/2se/dolphin/core/scheduler"
	"github.com/2se/dolphin/core/server"
	tw "github.com/RussellLuo/timingwheel"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
	"strconv"

	"os"
	"os/signal"

	"runtime/pprof"
	"runtime/trace"
	"strings"
	"syscall"
	"time"
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
		cli.StringFlag{
			Name:  "p,period",
			Usage: PeriodUsage,
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
	log.Infof("loaded config info: %s", cnf)

	pprof := cliCtx.String(FlagPprofKey)
	period := cliCtx.String(FlagPeriodKey)
	t := time.Second * 2
	if period != "" {
		ts, err := strconv.ParseInt(period, 10, 64)
		if err != nil {
			log.Fatalf("fail")
		}
		t = time.Second * time.Duration(ts)
	}
	if len(pprof) > 0 {
		go runPprof(t, pprof)
	}
	ticker := tw.NewTimingWheel(100*time.Millisecond, 550) // timer max wait 100ms * 550 ≈ 55sec.
	ticker.Start()
	defer ticker.Stop()

	dispatcher.InitLimiter(cnf.LimitCnf)

	despatcher := dispatcher.New()
	despatcher.Start()
	defer despatcher.Stop()

	//init kafka consumers to push message into event
	//outbox.ConsumersInit(cnf.KafkaCnf, despatcher, ticker)
	if cnf.KafkaCnf != nil && cnf.KafkaCnf.Enable {
		outbox.InitConsumers(cnf.KafkaCnf.Topics, despatcher, ticker)
	}

	localPeer, err := cluster.Init(cnf.ClusterCnf)
	if err != nil {
		log.Fatalf("failed to initial cluster. cause: %v", err)
	}
	if cnf.LoginMPCnf != nil {
		core.InitRequestCheck(cnf.LoginMPCnf, cnf.WhiteList)
	}
	//init router
	appRouter := router.Init(localPeer, cnf.RouteCnf, ticker)
	cluster.Start(appRouter)
	defer cluster.Shutdown()
	go scheduler.Start(cnf.RouteHttpCnf.Address)
	go scheduler.SchedulerStart(cnf.SchedulerCnf.Address)
	server.Init(cnf.WsCnf, despatcher, ticker)
	if err = server.ListenAndServe(signalHandler()); err != nil {
		log.WithError(err).Error("listen and serve websocket failed.")
	}

	return nil
}

func runPprof(period time.Duration, pprofFile string) {

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
	tracef, err := os.Create(pprofFile + ".trace")
	if err != nil {
		log.Fatal("Failed to create trace pprof file: ", err)
	}
	defer tracef.Close()

	pprof.StartCPUProfile(cpuf)
	trace.Start(tracef)
	defer pprof.StopCPUProfile()
	defer pprof.WriteHeapProfile(memf)
	defer trace.Stop()
	time.Sleep(period)
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

func traceProfile() {
	f, err := os.OpenFile("trace.out", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.Println("Trace started")
	trace.Start(f)
	defer trace.Stop()

	time.Sleep(60 * time.Second)
	fmt.Println("Trace stopped")
}
