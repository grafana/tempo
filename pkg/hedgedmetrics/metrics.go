package hedgedmetrics

import (
	"sync/atomic"
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	PublishDuration = 10 * time.Second
)

// CounterWithValue wraps prometheus.Counter and keeps track of the current value.
type CounterWithValue struct {
	counter prometheus.Counter
	value   atomic.Int64
}

func NewCounterWithValue(counter prometheus.Counter) *CounterWithValue {
	return &CounterWithValue{counter: counter}
}

func (c *CounterWithValue) Add(v int64) {
	c.value.Add(v)
	c.counter.Add(float64(v))
}

func (c *CounterWithValue) Value() int64 {
	return c.value.Load()
}

// StatsProvider defines the interface that wraps hedgedhttp.Stats for ease of testing
type StatsProvider interface {
	Snapshot() hedgedhttp.StatsSnapshot
}

// Publish flushes metrics from hedged requests every tickerDur
func Publish(s StatsProvider, counter *CounterWithValue, tickerDur time.Duration) {
	ticker := time.NewTicker(tickerDur)
	go func() {
		for range ticker.C {
			snap := s.Snapshot()
			hedgedRequests := int64(snap.ActualRoundTrips) - int64(snap.RequestedRoundTrips)
			if hedgedRequests < 0 {
				hedgedRequests = 0
			}

			// *hedgedhttp.Stats has counter but we need the delta for prometheus.Counter
			delta := hedgedRequests - counter.Value()
			if delta < 0 {
				delta = 0
			}
			counter.Add(delta)
		}
	}()
}
