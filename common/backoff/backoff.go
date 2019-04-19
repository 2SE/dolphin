package backoff

import (
	"github.com/2se/dolphin/config"
	"math/rand"
	"time"
)

var (
	defaultMaxDelay  = 3.0 * time.Minute
	defaultBaseDelay = 1.0 * time.Second
	defaultFactor    = 1.6
	defaultJitter    = 0.2
)

type Backoff struct {
	MaxDelay  time.Duration
	baseDelay time.Duration
	factor    float64
	jitter    float64
}

func newDefault() *Backoff {
	return &Backoff{
		MaxDelay:  defaultMaxDelay,
		baseDelay: defaultBaseDelay,
		factor:    defaultFactor,
		jitter:    defaultJitter,
	}
}

func New(cnf *config.ClusterConnectionConfig) *Backoff {
	if cnf == nil {
		return newDefault()
	}

	var (
		factor    float64
		jitter    float64
		maxDelay  time.Duration
		baseDelay time.Duration
	)

	if cnf.Factor <= 0 {
		factor = defaultFactor
	} else {
		factor = cnf.Factor
	}

	if cnf.Jitter <= 0 {
		jitter = defaultJitter
	} else {
		jitter = cnf.Jitter
	}

	if cnf.MaxDelay.Get() <= 0 {
		maxDelay = defaultMaxDelay
	} else {
		maxDelay = cnf.MaxDelay.Get()
	}

	if cnf.BaseDelay.Get() <= 0 {
		baseDelay = defaultBaseDelay
	} else {
		baseDelay = cnf.BaseDelay.Get()
	}

	return &Backoff{
		MaxDelay:  maxDelay,
		baseDelay: baseDelay,
		factor:    factor,
		jitter:    jitter,
	}
}

func (bc *Backoff) Duration(retries int) time.Duration {
	if retries <= 0 {
		return bc.baseDelay
	}

	backoff, max := float64(bc.baseDelay), float64(bc.MaxDelay)
	for backoff < max && retries > 0 {
		backoff *= bc.factor
		retries--
	}

	if backoff > max {
		backoff = max
	}

	backoff *= 1 + bc.jitter*(rand.Float64()*2-1)
	if backoff < 0 {
		return 0
	}

	return time.Duration(backoff)
}
