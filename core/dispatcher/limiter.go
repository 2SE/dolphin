package dispatcher

import (
	"github.com/2se/dolphin/config"
	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store/memstore"
)

var (
	limiter *Limiter
)

type Limiter struct {
	*throttled.GCRARateLimiter
}

func InitLimiter(c *config.LimitConfig) error {
	store, err := memstore.New(c.MaxNum)
	if err != nil {
		return err
	}
	quota := throttled.RateQuota{MaxRate: throttled.PerSec(c.MaxRate), MaxBurst: c.MaxBurst}
	l, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return err
	}
	limiter = &Limiter{
		l,
	}
	return nil
}
