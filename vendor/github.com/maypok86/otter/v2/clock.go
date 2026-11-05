// Copyright (c) 2025 Alexey Mayshev and contributors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otter

import (
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/maypok86/otter/v2/internal/xmath"
)

// Clock is a time source that
//   - Returns a time value representing the number of nanoseconds elapsed since some
//     fixed but arbitrary point in time
//   - Returns a channel that delivers “ticks” of a clock at intervals.
type Clock interface {
	// NowNano returns the number of nanoseconds elapsed since this clock's fixed point of reference.
	// Entry times and the data of SaveCacheTo and LoadCacheFrom are in the clock's units, so a cache
	// that saves or loads its data must use a clock counting from the Unix epoch, as the default one does.
	//
	// By default, time.Now().UnixNano() is used.
	NowNano() int64
	// Tick returns a channel that delivers “ticks” of a clock at intervals.
	//
	// The cache uses this method only for proactive expiration and calls Tick(time.Second) in a separate goroutine.
	// A nil channel, or one that is closed, means no ticks: the cache then does without periodic clean-ups.
	//
	// By default, the ticks of a [time.Ticker] are used, and the ticker is stopped when that goroutine
	// returns.
	Tick(duration time.Duration) <-chan time.Time
}

type timeSource interface {
	Clock
	Init()
	Sleep(duration time.Duration)
	ProcessTick()
	// Ticker returns the ticks for the periodic clean-up and a function that stops them, or nil when
	// they cannot be stopped. The built-in clocks stop their ticker: with GODEBUG=asynctimerchan=1
	// (Go 1.24-1.26), an unstopped ticker is never collected and fires for the life of the process.
	Ticker(duration time.Duration) (tick <-chan time.Time, stopTick func())
}

func newStoppableTicker(duration time.Duration) (tick <-chan time.Time, stopTick func()) {
	t := time.NewTicker(duration)
	return t.C, t.Stop
}

func newTimeSource(clock Clock) timeSource {
	if clock == nil {
		return &realSource{}
	}
	if r, ok := clock.(*realSource); ok {
		return r
	}
	if f, ok := clock.(*fakeSource); ok {
		return f
	}
	if c, ok := clock.(*CoarseClock); ok {
		return coarseSource{CoarseClock: c}
	}
	return newCustomSource(clock)
}

type customSource struct {
	clock         Clock
	isInitialized atomic.Bool
}

func newCustomSource(clock Clock) *customSource {
	return &customSource{
		clock: clock,
	}
}

func (cs *customSource) Init() {
	if !cs.isInitialized.Load() {
		cs.isInitialized.Store(true)
	}
}

func (cs *customSource) NowNano() int64 {
	if !cs.isInitialized.Load() {
		return 0
	}
	return cs.clock.NowNano()
}

func (cs *customSource) Tick(duration time.Duration) <-chan time.Time {
	return cs.clock.Tick(duration)
}

func (cs *customSource) Ticker(duration time.Duration) (tick <-chan time.Time, stopTick func()) {
	return cs.clock.Tick(duration), nil
}

func (cs *customSource) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (cs *customSource) ProcessTick() {}

type realSource struct {
	initMutex     sync.Mutex
	isInitialized atomic.Bool
	start         time.Time
	startNanos    atomic.Int64
}

func (c *realSource) Init() {
	if !c.isInitialized.Load() {
		c.initMutex.Lock()
		if !c.isInitialized.Load() {
			now := time.Now()
			c.start = now
			c.startNanos.Store(now.UnixNano())
			c.isInitialized.Store(true)
		}
		c.initMutex.Unlock()
	}
}

func (c *realSource) NowNano() int64 {
	if !c.isInitialized.Load() {
		return 0
	}
	return xmath.SaturatedAdd(c.startNanos.Load(), time.Since(c.start).Nanoseconds())
}

func (c *realSource) Tick(duration time.Duration) <-chan time.Time {
	return time.Tick(duration)
}

func (c *realSource) Ticker(duration time.Duration) (tick <-chan time.Time, stopTick func()) {
	return newStoppableTicker(duration)
}

func (c *realSource) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (c *realSource) ProcessTick() {}

// CoarseClock is a Clock that reads the time once per resolution in a separate goroutine and
// returns the time it read last. A cache with expiration or refresh reads its clock on every
// operation, and reading the precise time is a large part of a read from such a cache. Reading a
// CoarseClock costs about as much as reading a variable.
//
// In exchange, the time lags behind by up to the resolution, and by more when the goroutine
// cannot run in time. So an entry may expire or become refreshable up to that much earlier or
// later than its duration says. The goroutine also wakes up once per resolution for as long as
// the clock runs.
//
// One CoarseClock can be shared by several caches. The goroutine stops once the clock is no
// longer reachable, or when Stop is called.
//
// With testing/synctest, create the clock inside the bubble and stop it before the bubble's
// function returns: the goroutine notices that the clock is gone only at a tick, and time in the
// bubble no longer moves once the function returns. A clock created outside the bubble reads
// the real time, not the bubble's.
//
// Create a CoarseClock with NewCoarseClock.
type CoarseClock struct {
	precise realSource
	// now is the time the goroutine read last, or 0 when no goroutine updates it.
	now      atomic.Int64
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// NewCoarseClock returns a CoarseClock that reads the time once per resolution. It panics if the
// resolution is not positive.
func NewCoarseClock(resolution time.Duration) *CoarseClock {
	if resolution <= 0 {
		panic("otter: non-positive resolution for NewCoarseClock")
	}
	c := &CoarseClock{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	c.precise.Init()
	c.now.Store(c.precise.NowNano())
	// The goroutine holds the clock through a weak pointer, so that it stops once the clock is
	// collected.
	go updateCoarseClock(weak.Make(c), resolution, c.stop, c.done)
	return c
}

func updateCoarseClock(wc weak.Pointer[CoarseClock], resolution time.Duration, stop, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(resolution)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c := wc.Value()
			if c == nil {
				return
			}
			c.now.Store(c.precise.NowNano())
		}
	}
}

// NowNano returns the number of nanoseconds elapsed since the Unix epoch, as of the last time the
// clock read the time.
func (c *CoarseClock) NowNano() int64 {
	if now := c.now.Load(); now != 0 {
		return now
	}
	c.precise.Init()
	return c.precise.NowNano()
}

// Tick returns a channel that delivers ticks at intervals of the given duration, as [time.Tick] does.
func (c *CoarseClock) Tick(duration time.Duration) <-chan time.Time {
	return time.Tick(duration)
}

// Stop stops the goroutine that reads the time. After Stop, the clock reads the precise time on
// every call, so it is safe to stop it while caches still use it. Stop is mainly useful in tests
// that check for leaked goroutines: otherwise, the goroutine stops by itself once the clock is no
// longer reachable.
func (c *CoarseClock) Stop() {
	if c.stop == nil {
		return
	}
	c.stopOnce.Do(func() {
		close(c.stop)
	})
	<-c.done
	c.now.Store(0)
}

// coarseSource adds to CoarseClock the methods the cache needs, which the public type should not
// have.
type coarseSource struct {
	*CoarseClock
}

func (c coarseSource) Init() {}

func (c coarseSource) Ticker(duration time.Duration) (tick <-chan time.Time, stopTick func()) {
	return newStoppableTicker(duration)
}

func (c coarseSource) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (c coarseSource) ProcessTick() {}

type fakeSource struct {
	mutex          sync.Mutex
	now            time.Time
	initOnce       sync.Once
	sleeps         chan time.Duration
	tickWg         sync.WaitGroup
	sleepWg        sync.WaitGroup
	firstSleep     atomic.Bool
	withTick       atomic.Bool
	ticker         chan time.Time
	enableTickOnce sync.Once
	enableTick     chan time.Duration
}

func (f *fakeSource) Init() {
	f.initOnce.Do(func() {
		f.mutex.Lock()
		now := time.Now()
		f.now = now
		f.sleeps = make(chan time.Duration)
		f.firstSleep.Store(true)
		f.enableTick = make(chan time.Duration)
		f.ticker = make(chan time.Time, 1)
		f.mutex.Unlock()

		go func() {
			var (
				dur time.Duration
				d   time.Duration
			)
			enabled := false
			last := now
			for {
				select {
				case d = <-f.enableTick:
					enabled = true
					for d <= dur {
						if f.firstSleep.Load() {
							f.tickWg.Add(1)
							f.ticker <- last
							f.tickWg.Wait()
							f.firstSleep.Store(false)
						}
						last = last.Add(d)
						f.tickWg.Add(1)
						f.ticker <- last
						dur -= d
					}
				case s := <-f.sleeps:
					if enabled && f.firstSleep.Load() {
						f.tickWg.Add(1)
						f.ticker <- last
						f.tickWg.Wait()
						f.firstSleep.Store(false)
					}
					f.mutex.Lock()
					f.now = f.now.Add(s)
					f.mutex.Unlock()
					dur += s
					if enabled {
						for d <= dur {
							last = last.Add(d)
							f.tickWg.Add(1)
							f.ticker <- last
							dur -= d
						}
					}
					f.sleepWg.Done()
				}
			}
		}()
	})
}

func (f *fakeSource) NowNano() int64 {
	return f.getNow().UnixNano()
}

func (f *fakeSource) Tick(d time.Duration) <-chan time.Time {
	f.enableTickOnce.Do(func() {
		f.enableTick <- d
	})
	return f.ticker
}

func (f *fakeSource) Ticker(duration time.Duration) (tick <-chan time.Time, stopTick func()) {
	return f.Tick(duration), nil
}

func (f *fakeSource) Sleep(d time.Duration) {
	f.sleepWg.Add(1)
	f.sleeps <- d
	f.sleepWg.Wait()
}

func (f *fakeSource) getNow() time.Time {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.now
}

func (f *fakeSource) ProcessTick() {
	f.tickWg.Done()
}
