// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
)

// rateLimiter keeps track of the number of traces passing through the API. It
// takes a target rate via SetTargetRate and drops traces until that rate is met.
// For example, setting a target rate of 0.5 will ensure that only 50% of traces
// go through.
//
// The rateLimiter also uses a decay mechanism to ensure that older entries have
// lesser impact on the rate computation.
type rateLimiter struct {
	mu sync.RWMutex
	// stats keeps track of all the internal counters used by the rate limiter.
	stats info.RateLimiterStats
	// decayPeriod specifies the interval at which the counters should be decayed.
	decayPeriod time.Duration
	// decayFactor specifies the factor using which the counters are decayed. See
	// the documentation for (*rateLimiter).decayScore for more information.
	decayFactor float64
	// exit channel
	exit chan struct{}
}

// newRateLimiter returns an initialized rate limiter.
func newRateLimiter() *rateLimiter {
	decayFactor := 9.0 / 8.0
	return &rateLimiter{
		stats: info.RateLimiterStats{
			TargetRate: 1,
		},
		decayPeriod: 5 * time.Second,
		decayFactor: decayFactor,
		exit:        make(chan struct{}),
	}
}

// Run runs the rate limiter, occasionally decaying the score.
func (ps *rateLimiter) Run() {
	info.UpdateRateLimiter(*ps.Stats())
	t := time.NewTicker(ps.decayPeriod)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ps.decayScore()
		case <-ps.exit:
			return
		}
	}
}

// decayScore applies the decay to the rolling counters. It's purpose is to reduce the impact older
// traces have when computing the rate.
func (ps *rateLimiter) decayScore() {
	ps.mu.Lock()
	ps.stats.RecentPayloadsSeen /= ps.decayFactor
	ps.stats.RecentTracesSeen /= ps.decayFactor
	ps.stats.RecentTracesDropped /= ps.decayFactor
	ps.mu.Unlock()
}

// Stop stops the rate limiter.
func (ps *rateLimiter) Stop() { close(ps.exit) }

// SetTargetRate set the target limiting rate. The rateLimiter will not permit
// traces which would result in surpassing the rate.
func (ps *rateLimiter) SetTargetRate(rate float64) {
	ps.mu.Lock()
	ps.stats.TargetRate = rate
	ps.mu.Unlock()
}

// TargetRate returns the target rate. The value represents the percentage of traces
// that the rate limiter is trying to keep. It is the actual sampling rate. Depending
// on the traces received, it may differ from RealRate.
func (ps *rateLimiter) TargetRate() float64 {
	ps.mu.RLock()
	rate := ps.stats.TargetRate
	ps.mu.RUnlock()
	return rate
}

// RealRate returns the percentage of traces that the rate limiter has kept so far.
func (ps *rateLimiter) RealRate() float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.realRateLocked()
}

func (ps *rateLimiter) realRateLocked() float64 {
	if ps.stats.RecentTracesSeen <= 0 {
		// avoid division by zero
		return ps.stats.TargetRate
	}
	return 1 - (ps.stats.RecentTracesDropped / ps.stats.RecentTracesSeen)
}

// Active reports whether the rateLimiter is active. An inactive rateLimiter is one
// that has seen no traces (e.g. calls are always Permits(0))
func (ps *rateLimiter) Active() bool {
	ps.mu.RLock()
	active := ps.stats.RecentTracesSeen > 0
	ps.mu.RUnlock()
	return active
}

// Stats returns a copy of the currrent rate limiter's stats.
func (ps *rateLimiter) Stats() *info.RateLimiterStats {
	ps.mu.RLock()
	stats := ps.stats
	ps.mu.RUnlock()
	return &stats
}

// Permits reports wether the rate limiter should allow n more traces to
// enter the pipeline. Permits calls alter internal statistics which affect
// the result of calling RealRate(). It should only be called once per payload.
func (ps *rateLimiter) Permits(n int64) bool {
	if n <= 0 {
		return true // no sensible value in n, disable rate limiting
	}

	keep := true

	ps.mu.Lock()

	if ps.realRateLocked() > ps.stats.TargetRate {
		// we're keeping more than the target rate, drop
		keep = false
		ps.stats.RecentTracesDropped += float64(n)
	}

	// this should be done *after* testing the real rate against the target rate,
	// otherwise we could end up systematically dropping the first payload.
	ps.stats.RecentPayloadsSeen++
	ps.stats.RecentTracesSeen += float64(n)

	ps.mu.Unlock()

	if !keep {
		log.Debugf("Rate limiting at rate %.2f dropped payload with %d traces", ps.TargetRate(), n)
	}
	return keep
}

// computeRateLimitingRate gives us the new rate at which requests need to be rate limited. It is computed
// based on how much the [current] value surpasses the [max], and then combined with [rate]. The [current] and
// [max] values may be any values which have an impact on the allowed traffic, for example: a maximum amount
// of memory or CPU. [rate] is the current rate at which we may already be rate limiting.
//
// For example:
//
//	• If [max]=500 and [current]=700, the new rate will be 0.71 (500/700) to slow down the intake. Considering
//	  we are not rate limiting already ([rate]=1).
//
//	• If [max]=500 and [current]=700, and we are already rate-limiting at 50% ([rate]=0.5), then the new rate
//	  will be 0.71 * 0.5 = 0.35 to further reduce the intake.
//
// The formula also works backwards to gradually increase the intake once the [current] value is again <= [max].
func computeRateLimitingRate(max, current, rate float64) float64 {
	const (
		// deltaMin is a threshold that must be passed before changing the
		// rate limiting rate. If set to 0.1, for example, the new rate must be
		// below 90% or above 110% of the previous value, before we actually
		// adjust the sampling rate. This is to avoid over-adapting and jittering.
		deltaMin = float64(0.15) // +/- 15% change
		// rateMin is an absolute minimum rate, never sample more than this, it is
		// inefficient, the cost handling the payloads without even reading them
		// is too high anyway.
		rateMin = float64(0.05) // 5% hard-limit
	)
	if max <= 0 || current < 0 || rate < 0 || rate > 1 {
		// invalid values
		return 1
	}
	if current == 0 || rate == 0 {
		// not initialized yet; return now to avoid division by zero error
		return 1
	}

	// rate * (max / current)
	//      |      |
	//      |      1. Will give us the rate at which we will need to reduce the current
	//      |         intake to stay around [max].
	//      |
	//      2. We apply it to the current [rate].
	//
	// (1) The new rate is computed based on the percentage that our maximum allowed threshold [max]
	// represents from the [current] value (e.g. for [max]=500 and [current]=700 => 500/700 = 0.71).
	// (2) It is then applied to the current [rate].
	newRate := rate * max / current
	if newRate >= 1 {
		// no need to rate limit anything
		return 1
	}

	delta := (newRate - rate) / rate
	if delta > -deltaMin && delta < deltaMin {
		// no need to change, this is close enough to what we want (avoid jittering)
		return rate
	}

	// Taking the average of both values, it is going to converge in the long run,
	// but no need to hurry, wait for next iteration.
	newRate = (newRate + rate) / 2

	if newRate < rateMin {
		// Here, we would need a too-aggressive sampling rate to cope with
		// our objective, and rate limiting is not the right tool any more.
		return rateMin
	}
	return newRate
}
