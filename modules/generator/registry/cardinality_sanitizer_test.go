package registry

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestCardinalitySanitizer_Disabled(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 0}, 15*time.Minute)

	lbls := labels.FromStrings("__name__", "foo", "method", "GET", "url", "/api/users/123")
	result := s.Sanitize(lbls)
	require.Equal(t, lbls, result)
}

func TestCardinalitySanitizer_UnderLimit(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 100}, 15*time.Minute)

	// Insert a few distinct values — well under the limit
	for i := 0; i < 5; i++ {
		lbls := labels.FromStrings("__name__", "foo", "method", fmt.Sprintf("m%d", i))
		result := s.Sanitize(lbls)
		// Before the first maintenance tick, overLimit is false (default), so everything passes through
		require.Equal(t, fmt.Sprintf("m%d", i), result.Get("method"))
	}

	// Trigger maintenance to update overLimit flags
	triggerMaintenance(s)

	// Still under limit, should pass through
	lbls := labels.FromStrings("__name__", "foo", "method", "GET")
	result := s.Sanitize(lbls)
	require.Equal(t, "GET", result.Get("method"))
}

func TestCardinalitySanitizer_SelectiveOverflow(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 5}, 15*time.Minute)

	// Push many distinct url values but few method values
	for i := 0; i < 20; i++ {
		lbls := labels.FromStrings("__name__", "http_requests", "method", "GET", "url", fmt.Sprintf("/users/%d", i))
		s.Sanitize(lbls)
	}

	// Trigger maintenance to update overLimit flags
	triggerMaintenance(s)

	// Now the url should overflow but the method should be preserved
	lbls := labels.FromStrings("__name__", "http_requests", "method", "GET", "url", "/users/999")
	result := s.Sanitize(lbls)
	require.Equal(t, "GET", result.Get("method"), "low-cardinality label should be preserved")
	require.Equal(t, overflowValue, result.Get("url"), "high-cardinality label should overflow")
	require.Equal(t, "http_requests", result.Get("__name__"), "__name__ should be preserved")
}

func TestCardinalitySanitizer_MetricNamePreserved(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 1}, 15*time.Minute)

	// Push multiple distinct metric names — __name__ should never be overflowed
	for i := 0; i < 20; i++ {
		lbls := labels.FromStrings("__name__", fmt.Sprintf("metric_%d", i))
		s.Sanitize(lbls)
	}

	triggerMaintenance(s)

	lbls := labels.FromStrings("__name__", "metric_999")
	result := s.Sanitize(lbls)
	require.Equal(t, "metric_999", result.Get("__name__"))
}

// TestCardinalitySanitizer_ConcurrentAccess verifies Sanitize() is safe to call
// from multiple goroutines. Run with -race to detect unsynchronized access.
func TestCardinalitySanitizer_ConcurrentAccess(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 10}, 15*time.Minute)

	var wg sync.WaitGroup
	wg.Add(10)
	for g := 0; g < 10; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				lbls := labels.FromStrings("__name__", "m", "label", fmt.Sprintf("g%d-v%d", id, i))
				_ = s.Sanitize(lbls)
			}
		}(g)
	}
	wg.Wait()
}

func TestCardinalitySanitizer_OverflowMetrics(t *testing.T) {
	overflowCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_overflow_counter",
	})

	s := NewCardinalitySanitizer("test-metrics", &mockOverrides{maxCardinalityPerLabel: 5}, 15*time.Minute)
	s.overflowCounter = overflowCounter

	// Push enough distinct values to exceed limit
	for i := 0; i < 20; i++ {
		lbls := labels.FromStrings("__name__", "m", "url", fmt.Sprintf("/path/%d", i))
		s.Sanitize(lbls)
	}

	// Force overLimit
	s.mtx.Lock()
	s.labelsState["url"].overLimit = true
	s.mtx.Unlock()

	// Now sanitize — should increment the counter
	lbls := labels.FromStrings("__name__", "m", "url", "/path/new")
	result := s.Sanitize(lbls)
	require.Equal(t, overflowValue, result.Get("url"))

	var m io_prometheus_client.Metric
	require.NoError(t, overflowCounter.Write(&m))
	require.Equal(t, float64(1), m.GetCounter().GetValue())
}

func TestCardinalitySanitizer_MultipleLabelsOverflow(t *testing.T) {
	s := NewCardinalitySanitizer("test", &mockOverrides{maxCardinalityPerLabel: 5}, 15*time.Minute)

	// Push many distinct values for BOTH url and user_id
	for i := 0; i < 20; i++ {
		lbls := labels.FromStrings("__name__", "m", "method", "GET", "url", fmt.Sprintf("/p/%d", i), "user_id", fmt.Sprintf("u%d", i))
		s.Sanitize(lbls)
	}

	triggerMaintenance(s)

	lbls := labels.FromStrings("__name__", "m", "method", "GET", "url", "/p/999", "user_id", "u999")
	result := s.Sanitize(lbls)
	require.Equal(t, "GET", result.Get("method"), "low-cardinality label preserved")
	require.Equal(t, overflowValue, result.Get("url"), "high-cardinality url overflows")
	require.Equal(t, overflowValue, result.Get("user_id"), "high-cardinality user_id overflows")
}

// triggerMaintenance forces overLimit flags to be re-evaluated from current sketch estimates.
func triggerMaintenance(s *CardinalitySanitizer) {
	s.mtx.Lock()
	for _, state := range s.labelsState {
		est := state.sketch.Estimate()
		state.overLimit = est > s.maxCardinality
	}
	s.mtx.Unlock()
}
