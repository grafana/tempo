// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package timing is used to aggregate timing calls within hotpaths to avoid using
// repeated statsd calls. The package has a default set that reports at 10 second
// intervals and can be used directly. If a different behaviour or reporting pattern
// is desired, a custom Set may be created.
package timing

import (
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
)

// AutoreportInterval specifies the interval at which the default set reports.
const AutoreportInterval = 10 * time.Second

var (
	defaultSet = NewSet()
	stopReport = defaultSet.Autoreport(AutoreportInterval)
)

// Since records the duration for the given metric name as time passed since start.
// It uses the default set which is reported at 10 second intervals.
func Since(name string, start time.Time) { defaultSet.Since(name, start) }

// Stop permanently stops the default set from auto-reporting and flushes any remaining
// metrics. It can be useful to call when the program exits to ensure everything is
// submitted.
func Stop() { stopReport() }

// NewSet returns a new, ready to use Set.
func NewSet() *Set {
	return &Set{c: make(map[string]*counter)}
}

// Set represents a set of metrics that can be used for timing. Use NewSet to initialize
// a new Set. Use Report (or Autoreport) to submit metrics. Set is safe for concurrent use.
type Set struct {
	mu sync.RWMutex        // guards c
	c  map[string]*counter // maps names to their aggregates
}

// Autoreport enables autoreporting of the Set at the given interval. It returns a
// cancellation function.
func (s *Set) Autoreport(interval time.Duration) (cancelFunc func()) {
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				s.Report()
			case <-stop:
				s.Report()
				return
			}
		}
	}()
	var once sync.Once // avoid panics
	return func() {
		once.Do(func() {
			stop <- struct{}{}
			<-stop
		})
	}
}

// Since records the duration for the given metric name as *time passed since start*.
// If name does not exist, as defined by NewSet, it creates it.
func (s *Set) Since(name string, start time.Time) {
	ms := time.Since(start) / time.Millisecond
	s.getCounter(name).add(float64(ms))
}

// getCounter returns the counter with the given name, initializing any uninitialized
// fields of s.
func (s *Set) getCounter(name string) *counter {
	s.mu.RLock()
	c, ok := s.c[name]
	s.mu.RUnlock()
	if !ok {
		// initialize a new counter
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.c[name]; ok {
			// another goroutine already did it
			return c
		}
		s.c[name] = newCounter(name)
		c = s.c[name]
	}
	return c
}

// Report reports all of the Set's metrics to the statsd client.
func (s *Set) Report() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.c {
		c.flush()
	}
}

type counter struct {
	// name specifies the name of this counter
	name string

	// mu guards the below field from changes during flushing.
	mu    sync.RWMutex
	sum   *atomic.Float64
	count *atomic.Float64
	max   *atomic.Float64
}

func newCounter(name string) *counter {
	return &counter{
		name:  name,
		count: atomic.NewFloat64(0),
		max:   atomic.NewFloat64(0),
		sum:   atomic.NewFloat64(0),
	}
}

func (c *counter) add(v float64) {
	c.mu.RLock()
	if v > c.max.Load() {
		c.max.Store(v)
	}
	c.count.Add(1)
	c.sum.Add(v)
	c.mu.RUnlock()
}

func (c *counter) flush() {
	c.mu.Lock()
	count := c.count.Swap(0)
	sum := c.sum.Swap(0)
	max := c.max.Swap(0)
	c.mu.Unlock()
	metrics.Count(c.name+".count", int64(count), nil, 1)
	metrics.Gauge(c.name+".max", max, nil, 1)
	metrics.Gauge(c.name+".avg", sum/count, nil, 1)
}
