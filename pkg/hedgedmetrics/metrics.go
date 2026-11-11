package hedgedmetrics

import (
	"time"

	"github.com/cristalhq/hedgedhttp"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	hedgedMetricsPublishDuration = 10 * time.Second
)

// statsProvider defines the interface that wraps hedgedhttp.Stats for ease of testing
type statsProvider interface {
	ActualRoundTrips() uint64
	RequestedRoundTrips() uint64
}

// hedgedCounter publishes the number of hedged requests (actual minus requested round trips) to a
// prometheus counter.
type hedgedCounter struct {
	published uint64 // highest number of hedged requests published so far
	counter   prometheus.Counter
}

func (h *hedgedCounter) update(s statsProvider) {
	// hedgedhttp increments RequestedRoundTrips before dispatching a request but ActualRoundTrips only
	// once each sub-request starts, so in-flight requests can make actual lower than requested, or the
	// difference lower than a previous one. Loading actual first means a newer request can only lower
	// the difference, never raise it above the real number of hedged requests. Only publish increases
	// over the highest value published so far.
	actual := s.ActualRoundTrips()
	requested := s.RequestedRoundTrips()
	if actual <= requested {
		return
	}

	hedged := actual - requested
	if hedged <= h.published {
		return
	}

	h.counter.Add(float64(hedged - h.published))
	h.published = hedged
}

// Publish flushes metrics from hedged requests every tickerDur
func Publish(s *hedgedhttp.Stats, counter prometheus.Counter) {
	publishWithDuration(s, counter, hedgedMetricsPublishDuration)
}

func publishWithDuration(s statsProvider, counter prometheus.Counter, duration time.Duration) {
	ticker := time.NewTicker(duration)
	hc := &hedgedCounter{counter: counter}

	go func() {
		for range ticker.C {
			hc.update(s)
		}
	}()
}
