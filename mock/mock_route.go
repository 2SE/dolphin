package mock

import (
	"github.com/2se/dolphin/config"
	"github.com/2se/dolphin/core/router"
	"time"
)

var (
	MockRoute = router.Init(MockCluster, &config.RouteConfig{
		Recycle:   config.Duration{time.Second * 60},
		Threshold: 10,
		Timeout:   config.Duration{time.Second * 2},
	})
)
