package mock

import (
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core/router"
	tw "github.com/RussellLuo/timingwheel"
	"time"
)

var (
	Ticker    = tw.NewTimingWheel(100*time.Millisecond, 650)
	MockRoute = router.Init(MockCluster, &config.RouteConfig{
		Recycle:   config.Duration{time.Second * 60},
		Threshold: 10,
		Timeout:   config.Duration{time.Second * 2},
	}, Ticker)
)
