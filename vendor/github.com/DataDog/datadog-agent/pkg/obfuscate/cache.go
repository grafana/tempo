// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package obfuscate

import (
	"fmt"
	"time"

	"github.com/outcaste-io/ristretto"
)

// measuredCache is a wrapper on top of *ristretto.Cache which additionally
// sends metrics (hits and misses) every 10 seconds.
type measuredCache struct {
	*ristretto.Cache

	// close allows sending shutdown notification.
	close  chan struct{}
	statsd StatsClient
}

// Close gracefully closes the cache when active.
func (c *measuredCache) Close() {
	if c.Cache == nil {
		return
	}
	c.close <- struct{}{}
	<-c.close
}

func (c *measuredCache) statsLoop() {
	defer func() {
		c.close <- struct{}{}
	}()
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	mx := c.Cache.Metrics
	for {
		select {
		case <-tick.C:
			c.statsd.Gauge("datadog.trace_agent.ofuscation.sql_cache.hits", float64(mx.Hits()), nil, 1)     //nolint:errcheck
			c.statsd.Gauge("datadog.trace_agent.ofuscation.sql_cache.misses", float64(mx.Misses()), nil, 1) //nolint:errcheck
		case <-c.close:
			c.Cache.Close()
			return
		}
	}
}

type cacheOptions struct {
	On     bool
	Statsd StatsClient
}

// newMeasuredCache returns a new measuredCache.
func newMeasuredCache(opts cacheOptions) *measuredCache {
	if !opts.On {
		// a nil *ristretto.Cache is a no-op cache
		return &measuredCache{}
	}
	cfg := &ristretto.Config{
		// We know that the maximum allowed resource length is 5K. This means that
		// in 5MB we can store a minimum of 1000 queries.
		MaxCost: 5000000,

		// An appromixated worst-case scenario when the cache is filled with small
		// queries averaged as being of length 11 ("LOCK TABLES"), we would be able
		// to fit 476K of them into 5MB of cost.
		//
		// We average it to 500K and multiply 10x as the documentation recommends.
		NumCounters: 500000 * 10,

		BufferItems: 64,   // default recommended value
		Metrics:     true, // enable hit/miss counters
	}
	cache, err := ristretto.NewCache(cfg)
	if err != nil {
		panic(fmt.Errorf("Error starting obfuscator query cache: %v", err))
	}
	c := measuredCache{
		close:  make(chan struct{}),
		statsd: opts.Statsd,
		Cache:  cache,
	}
	go c.statsLoop()
	return &c
}
