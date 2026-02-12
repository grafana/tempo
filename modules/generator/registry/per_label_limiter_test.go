package registry

import (
	"fmt"
	"sync"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

// testMaxCardinality returns a maxCardinalityFunc that always returns the given value.
func testMaxCardinality(value uint64) maxCardinalityFunc {
	return func(string) uint64 { return value }
}

func TestPerLabelLimiter_Disabled(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(0), 15*time.Minute)

	lbls := labels.FromStrings("__name__", "foo", "method", "GET", "url", "/api/users/123")
	result := s.Limit(lbls)
	require.Equal(t, lbls, result)
}

func TestPerLabelLimiter_UnderLimit(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(100), 15*time.Minute)

	// Insert a few distinct values - well under the limit
	for i := 0; i < 5; i++ {
		lbls := labels.FromStrings("__name__", "foo", "method", fmt.Sprintf("m%d", i))
		result := s.Limit(lbls)
		// Before the first maintenance tick, overLimit is false (default), so everything passes through
		require.Equal(t, fmt.Sprintf("m%d", i), result.Get("method"))
	}

	// Trigger maintenance to update overLimit flags
	triggerMaintenance(s)

	// Still under limit, should pass through
	lbls := labels.FromStrings("__name__", "foo", "method", "GET")
	result := s.Limit(lbls)
	require.Equal(t, "GET", result.Get("method"))
}

func TestPerLabelLimiter_HighCardinalityOverflows(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(5), 15*time.Minute)

	// Push distinct url values but few method values
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "http_requests", "method", "GET", "url", fmt.Sprintf("/users/%d", i))
		s.Limit(lbls)
	}

	// Trigger maintenance to update overLimit flags
	triggerMaintenance(s)

	// Now the url should overflow but the method should be preserved
	lbls := labels.FromStrings("__name__", "http_requests", "method", "GET", "url", "/users/999")
	result := s.Limit(lbls)
	require.Equal(t, "GET", result.Get("method"), "low-cardinality label should be preserved")
	require.Equal(t, overflowValue, result.Get("url"), "high-cardinality label should have overflow value")
	require.Equal(t, "http_requests", result.Get("__name__"), "__name__ should be preserved")
}

func TestPerLabelLimiter_MultipleHighCardinalityOverflows(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(5), 15*time.Minute)

	// Push many distinct values for BOTH url and user_id
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "m", "method", "GET", "url", fmt.Sprintf("/p/%d", i), "user_id", fmt.Sprintf("u%d", i))
		s.Limit(lbls)
	}

	triggerMaintenance(s)

	lbls := labels.FromStrings("__name__", "m", "method", "GET", "url", "/p/999", "user_id", "u999")
	result := s.Limit(lbls)
	require.Equal(t, "GET", result.Get("method"), "low-cardinality label preserved")
	require.Equal(t, overflowValue, result.Get("url"), "high-cardinality url overflows")
	require.Equal(t, overflowValue, result.Get("user_id"), "high-cardinality user_id overflows")
}

func TestPerLabelLimiter_MetadataLabelsNeverOverflows(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(5), 15*time.Minute)

	// Push many distinct values for all metadata labels to exceed the limit
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings(
			"__name__", fmt.Sprintf("metric_%d", i),
			"__type__", fmt.Sprintf("type_%d", i),
			"__unit__", fmt.Sprintf("unit_%d", i),
			"url", fmt.Sprintf("/path/%d", i),
		)
		s.Limit(lbls)
	}

	triggerMaintenance(s)

	lbls := labels.FromStrings("__name__", "metric_999", "__type__", "type_999", "__unit__", "unit_999", "url", "/path/999")
	result := s.Limit(lbls)
	require.Equal(t, "metric_999", result.Get("__name__"), "__name__ should never overflow")
	require.Equal(t, "type_999", result.Get("__type__"), "__type__ should never overflow")
	require.Equal(t, "unit_999", result.Get("__unit__"), "__unit__ should never overflow")
	require.Equal(t, overflowValue, result.Get("url"), "non-metadata label should overflow")
}

func TestPerLabelLimiter_OverflowMetrics(t *testing.T) {
	tenant := "test-overflow-metrics"
	s := NewPerLabelLimiter(tenant, testMaxCardinality(5), 15*time.Minute)

	// Push enough distinct values to exceed the limit
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "m", "url", fmt.Sprintf("/path/%d", i))
		s.Limit(lbls)
	}

	triggerMaintenance(s)

	// Now limit - should increment the counter
	lbls := labels.FromStrings("__name__", "m", "url", "/path/new")
	result := s.Limit(lbls)
	require.Equal(t, overflowValue, result.Get("url"))

	var m io_prometheus_client.Metric
	require.NoError(t, metricCardinalityLimitOverflows.WithLabelValues(tenant).Write(&m))
	require.Equal(t, float64(1), m.GetCounter().GetValue())
}

// TestPerLabelLimiter_ConcurrentAccess verifies Limit() is safe to call
// from multiple goroutines. Run with -race to detect unsynchronized access.
func TestPerLabelLimiter_ConcurrentAccess(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(10), 15*time.Minute)

	var wg sync.WaitGroup
	wg.Add(10)
	for g := 0; g < 10; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				lbls := labels.FromStrings("__name__", "m", "label", fmt.Sprintf("g%d-v%d", id, i))
				result := s.Limit(lbls)
				require.Equal(t, "m", result.Get("__name__"), "metric name must be preserved")
				val := result.Get("label")
				require.True(t, val == fmt.Sprintf("g%d-v%d", id, i) || val == overflowValue,
					"label value must be original or overflow, got: %s", val)
			}
		}(g)
	}
	wg.Wait()

	// After all goroutines finish, trigger maintenance and verify the state is consistent
	triggerMaintenance(s)

	s.mtx.Lock()
	state, ok := s.labelsState["label"]
	s.mtx.Unlock()
	require.True(t, ok, "label state should exist after concurrent inserts")
	require.Greater(t, state.sketch.Estimate(), uint64(0), "sketch should have recorded values")
}

// triggerMaintenance force runs doPeriodicMaintenance and evaluates overLimit from current sketch estimates.
func triggerMaintenance(s *PerLabelLimiter) {
	ch := make(chan time.Time, 1)
	s.demandUpdateChan = ch
	ch <- time.Now()
	s.doPeriodicMaintenance()
}
