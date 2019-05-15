package server

import (
	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store/memstore"
)

var (
	limiter *Limiter
)

type Limiter struct {
	*throttled.GCRARateLimiter
}

func initLimiter(maxNum, maxRate, maxBurst int) error {
	store, err := memstore.New(maxNum)
	if err != nil {
		return err
	}
	quota := throttled.RateQuota{MaxRate: throttled.PerHour(maxRate), MaxBurst: maxBurst}
	l, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return err
	}
	limiter = &Limiter{
		l,
	}
	return nil
}
