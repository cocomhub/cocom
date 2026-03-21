// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

// ErrNotObtained is returned when a lock cannot be obtained.
var ErrNotObtained = errors.New("lock not obtained")

// Strategy allows to customise the lock retry strategy.
type Strategy interface {
	// NextBackoff returns the next backoff duration.
	NextBackoff() time.Duration
}

func NewStrategy(name string) Strategy {
	var cfg Config
	err := viper.UnmarshalKey(name, &cfg)
	if err != nil {
		return nil
	}
	return NewStrategyByConfig(cfg)
}

func NewStrategyByConfig(cfg Config) Strategy {
	switch cfg.Type {
	case "linear":
		return LinearBackoff(cfg.LinearBackoff)
	case "no_retry":
		return NoRetry()
	case "limit":
		return LimitRetry(NewStrategy(cfg.LimitStrategyName), cfg.LimitMax)
	case "exponential":
		return ExponentialBackoff(cfg.ExponentialBackoffMin, cfg.ExponentialBackoffMax)
	}
	return NewStrategy("mutex.retry.default")
}

type Config struct {
	Type                  string        `mapstructure:"type" default:"linear"`
	LinearBackoff         time.Duration `mapstructure:"linear_backoff" default:"50ms"`
	NoRetry               bool          `mapstructure:"no_retry" default:"false"`
	LimitStrategyName     string        `mapstructure:"limit_strategy_name" default:"mutex.retry.default"`
	LimitMax              int           `mapstructure:"limit_max" default:"10"`
	ExponentialBackoffMin time.Duration `mapstructure:"exponential_backoff_min" default:"16ms"`
	ExponentialBackoffMax time.Duration `mapstructure:"exponential_backoff_max" default:"1000ms"`
}

type linearBackoff time.Duration

// LinearBackoff allows retries regularly with customized intervals
func LinearBackoff(backoff time.Duration) Strategy {
	return linearBackoff(backoff)
}

// NoRetry acquire the lock only once.
func NoRetry() Strategy {
	return linearBackoff(0)
}

func (r linearBackoff) NextBackoff() time.Duration {
	return time.Duration(r)
}

type limitedRetry struct {
	s   Strategy
	cnt int64
	max int64
}

// LimitRetry limits the number of retries to max attempts.
func LimitRetry(s Strategy, max int) Strategy {
	return &limitedRetry{s: s, max: int64(max)}
}

func (r *limitedRetry) NextBackoff() time.Duration {
	if atomic.LoadInt64(&r.cnt) >= r.max {
		return 0
	}
	atomic.AddInt64(&r.cnt, 1)
	return r.s.NextBackoff()
}

type exponentialBackoff struct {
	cnt uint64

	min, max time.Duration
}

// ExponentialBackoff strategy is an optimization strategy with a retry time of 2**n milliseconds (n means number of times).
// You can set a minimum and maximum value, the recommended minimum value is not less than 16ms.
func ExponentialBackoff(min, max time.Duration) Strategy {
	return &exponentialBackoff{min: min, max: max}
}

func (r *exponentialBackoff) NextBackoff() time.Duration {
	cnt := atomic.AddUint64(&r.cnt, 1)

	ms := 2 << 25
	if cnt < 25 {
		ms = 2 << cnt
	}

	if d := time.Duration(ms) * time.Millisecond; d < r.min {
		return r.min
	} else if r.max != 0 && d > r.max {
		return r.max
	} else {
		return d
	}
}
