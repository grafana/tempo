package registry

import (
	"fmt"
	"sync"
	"sync/atomic"
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

// TestPerLabelLimiter_RuntimeEnableDisable verifies that toggling max_cardinality_per_label at runtime
// via overrides takes effect without restarting the generator.
func TestPerLabelLimiter_RuntimeEnableDisable(t *testing.T) {
	tenant := "test-runtime-toggle"
	var maxCardinality atomic.Uint64
	maxCardinality.Store(0) // start disabled

	s := NewPerLabelLimiter(tenant, func(string) uint64 {
		return maxCardinality.Load()
	}, 15*time.Minute)

	// Phase 1: Disabled - all labels pass through
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "m", "url", fmt.Sprintf("/path/%d", i))
		result := s.Limit(lbls)
		require.Equal(t, fmt.Sprintf("/path/%d", i), result.Get("url"), "should pass through when disabled")
	}

	// when disabled, Limit() returns before inserting into sketches, so labelsState
	// is empty, which means no demand gauge was published as well.
	triggerDemandUpdate(s)
	s.mtx.Lock()
	labelsStateLen := len(s.labelsState)
	s.mtx.Unlock()
	require.Equal(t, 0, labelsStateLen, "no label state should exist when disabled")

	// Phase 2: Enable at runtime by changing the override
	maxCardinality.Store(5)
	triggerDemandUpdate(s)

	// Push more distinct values - should overflow now
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "m", "url", fmt.Sprintf("/path/%d", i))
		s.Limit(lbls)
	}
	triggerDemandUpdate(s)

	// demand gauge should be published when enabled
	var g io_prometheus_client.Metric
	require.NoError(t, metricLabelCardinalityDemand.WithLabelValues(tenant, "url").Write(&g))
	require.Greater(t, g.GetGauge().GetValue(), float64(5), "demand gauge should be published when enabled")

	lbls := labels.FromStrings("__name__", "m", "url", "/path/999")
	result := s.Limit(lbls)
	require.Equal(t, overflowValue, result.Get("url"), "should overflow after runtime enable")

	// capture the demand gauge value before disabling
	var before io_prometheus_client.Metric
	require.NoError(t, metricLabelCardinalityDemand.WithLabelValues(tenant, "url").Write(&before))
	demandBefore := before.GetGauge().GetValue()

	// Phase 3: Disable again at runtime
	maxCardinality.Store(0)
	triggerDemandUpdate(s)

	lbls = labels.FromStrings("__name__", "m", "url", "/path/new/10000")
	result = s.Limit(lbls)
	require.Equal(t, "/path/new/10000", result.Get("url"), "should pass through after runtime disable")

	// demand gauge should be stale (not updated) after disabling
	var after io_prometheus_client.Metric
	require.NoError(t, metricLabelCardinalityDemand.WithLabelValues(tenant, "url").Write(&after))
	require.Equal(t, demandBefore, after.GetGauge().GetValue(), "demand gauge should not be updated when disabled")
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
	triggerDemandUpdate(s)

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
	triggerDemandUpdate(s)

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

	triggerDemandUpdate(s)

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

	triggerDemandUpdate(s)

	lbls := labels.FromStrings("__name__", "metric_999", "__type__", "type_999", "__unit__", "unit_999", "url", "/path/999")
	result := s.Limit(lbls)
	require.Equal(t, "metric_999", result.Get("__name__"), "__name__ should never overflow")
	require.Equal(t, "type_999", result.Get("__type__"), "__type__ should never overflow")
	require.Equal(t, "unit_999", result.Get("__unit__"), "__unit__ should never overflow")
	require.Equal(t, overflowValue, result.Get("url"), "non-metadata label should overflow")
}

// TestPerLabelLimiter_RecoveryAfterOverflow verifies that a label
// recovers from overflow once the user reduces cardinality and the old
// high-cardinality sketches rotate out of the sliding window.
func TestPerLabelLimiter_RecoveryAfterOverflow(t *testing.T) {
	staleDuration := 15 * time.Minute
	s := NewPerLabelLimiter("test", testMaxCardinality(5), staleDuration)

	// Phase 1: Push high-cardinality data to trigger overflow
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "http_requests", "url", fmt.Sprintf("/users/%d", i))
		s.Limit(lbls)
	}
	triggerDemandUpdate(s)

	// Verify that overflow happens
	lbls := labels.FromStrings("__name__", "http_requests", "url", "/users/999")
	result := s.Limit(lbls)
	require.Equal(t, overflowValue, result.Get("url"), "should overflow while cardinality is high")

	// Phase 2: Simulate time passing - Advance the sketch enough times to fully rotate
	// out all old high-cardinality data from the sketch ring and prune it.
	// With staleDuration=15m and sketchDuration=5m, sketchesLength=4, so prune it 4 times
	for i := 0; i < 4; i++ {
		triggerPrune(s)
	}

	// Phase 3: Push only low-cardinality data (within limit)
	for i := 0; i < 3; i++ {
		lbls := labels.FromStrings("__name__", "http_requests", "url", fmt.Sprintf("/api/v1/resource_%d", i))
		s.Limit(lbls)
	}

	// Trigger maintenance to re-evaluate overLimit from the now-low estimate
	triggerDemandUpdate(s)

	// Verify recovery - label values should pass through again
	lbls = labels.FromStrings("__name__", "http_requests", "url", "/api/v1/healthy")
	result = s.Limit(lbls)
	require.Equal(t, "/api/v1/healthy", result.Get("url"), "should recover after cardinality drops below limit")

	// Phase 4: Cardinality explodes again - verify overflow kicks back in
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "http_requests", "url", fmt.Sprintf("/new_endpoint/%d", i))
		s.Limit(lbls)
	}
	triggerDemandUpdate(s)

	lbls = labels.FromStrings("__name__", "http_requests", "url", "/new_endpoint/999")
	result = s.Limit(lbls)
	require.Equal(t, overflowValue, result.Get("url"), "should overflow again after cardinality increases")
}

func TestPerLabelLimiter_OverflowMetrics(t *testing.T) {
	tenant := "test-overflow-metrics"
	s := NewPerLabelLimiter(tenant, testMaxCardinality(5), 15*time.Minute)

	// Push enough distinct values to exceed the limit
	for i := 0; i < 10; i++ {
		lbls := labels.FromStrings("__name__", "m", "url", fmt.Sprintf("/path/%d", i))
		s.Limit(lbls)
	}

	triggerDemandUpdate(s)

	// Now limit - should increment the counter
	lbls := labels.FromStrings("__name__", "m", "url", "/path/new")
	result := s.Limit(lbls)
	require.Equal(t, overflowValue, result.Get("url"))

	var m io_prometheus_client.Metric
	require.NoError(t, metricLabelValuesLimited.WithLabelValues(tenant, "url").Write(&m))
	require.Equal(t, float64(1), m.GetCounter().GetValue())

	// Verify demand estimate gauge was updated during triggerDemandUpdate
	var g io_prometheus_client.Metric
	require.NoError(t, metricLabelCardinalityDemand.WithLabelValues(tenant, "url").Write(&g))
	require.GreaterOrEqual(t, g.GetGauge().GetValue(), float64(10), "demand estimate should reflect the distinct values pushed")
}

// TestPerLabelLimiter_ConcurrentAccess verifies Limit() is safe to call
// from multiple goroutines while doPeriodicMaintenance fires concurrently.
// Run with -race to detect unsynchronized access.
func TestPerLabelLimiter_ConcurrentAccess(t *testing.T) {
	s := NewPerLabelLimiter("test", testMaxCardinality(10), 15*time.Minute)

	// Replace tickers with channels we control, so doPeriodicMaintenance
	// actually runs its demand-update and prune paths during the test.
	demandCh := make(chan time.Time, 10)
	pruneCh := make(chan time.Time, 10)
	s.demandUpdateChan = demandCh
	s.pruneChan = pruneCh

	var wg sync.WaitGroup
	wg.Add(12) // 10 Limit() goroutines + 1 demand update ticker + 1 prune ticker

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

	// Feed demand update ticks concurrently - picked up by doPeriodicMaintenance
	// inside Limit() calls, exercising the full code path.
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			demandCh <- time.Now()
			time.Sleep(time.Millisecond)
		}
	}()

	// Feed prune ticks concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			pruneCh <- time.Now()
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	// After all goroutines finish, trigger maintenance and verify the state is consistent
	triggerDemandUpdate(s)

	s.mtx.Lock()
	state, ok := s.labelsState["label"]
	s.mtx.Unlock()
	require.True(t, ok, "label state should exist after concurrent inserts")
	// Estimate may be less than 1000 because prune ticks rotate out sketch data during the test
	require.Greater(t, state.sketch.Estimate(), uint64(0), "sketch should have recorded values")
}

func BenchmarkPerLabelLimiter_Limit(b *testing.B) {
	b.Run("disabled", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(0), 15*time.Minute)
		lbls := labels.FromStrings("__name__", "http_requests", "method", "GET", "url", "/api/v1/users")
		b.ReportAllocs()
		// Reset timer so setup (limiter creation, label generation, warmup) isn't measured
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Limit(lbls)
		}
	})

	b.Run("under_limit", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(1000), 15*time.Minute)
		// Pre-generate distinct labels to simulate real traffic with unique values
		n := 500
		allLbls := make([]labels.Labels, n)
		for i := 0; i < n; i++ {
			allLbls[i] = labels.FromStrings("__name__", "http_requests", "method", "GET", "url", fmt.Sprintf("/api/v1/users/%d", i))
		}
		s.Limit(allLbls[0])
		triggerDemandUpdate(s)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Limit(allLbls[i%n])
		}
	})

	b.Run("over_limit", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(5), 15*time.Minute)
		n := 500
		allLbls := make([]labels.Labels, n)
		for i := 0; i < n; i++ {
			allLbls[i] = labels.FromStrings("__name__", "http_requests", "method", "GET", "url", fmt.Sprintf("/api/v1/users/%d", i))
		}
		for i := 0; i < 20; i++ {
			s.Limit(allLbls[i])
		}
		triggerDemandUpdate(s)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Limit(allLbls[i%n])
		}
	})

	b.Run("many_labels_under_limit", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(1000), 15*time.Minute)
		n := 500
		allLbls := make([]labels.Labels, n)
		for i := 0; i < n; i++ {
			allLbls[i] = labels.FromStrings(
				"__name__", "http_requests",
				"method", "GET",
				"url", fmt.Sprintf("/api/v1/users/%d", i),
				"status_code", "200",
				"service", "frontend",
				"region", "us-east-1",
				"instance", fmt.Sprintf("pod-%d", i),
			)
		}
		s.Limit(allLbls[0])
		triggerDemandUpdate(s)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Limit(allLbls[i%n])
		}
	})

	b.Run("many_labels_over_limit", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(5), 15*time.Minute)
		n := 500
		allLbls := make([]labels.Labels, n)
		for i := 0; i < n; i++ {
			allLbls[i] = labels.FromStrings(
				"__name__", "http_requests",
				"method", fmt.Sprintf("m%d", i),
				"url", fmt.Sprintf("/path/%d", i),
				"status_code", fmt.Sprintf("%d", i),
				"service", fmt.Sprintf("svc%d", i),
				"region", fmt.Sprintf("r%d", i),
				"instance", fmt.Sprintf("pod%d", i),
			)
		}
		// Warm up to trigger overflow on all labels
		for i := 0; i < 20; i++ {
			s.Limit(allLbls[i])
		}
		triggerDemandUpdate(s)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Limit(allLbls[i%n])
		}
	})

	b.Run("parallel", func(b *testing.B) {
		s := NewPerLabelLimiter("bench", testMaxCardinality(1000), 15*time.Minute)
		n := 500
		allLbls := make([]labels.Labels, n)
		for i := 0; i < n; i++ {
			allLbls[i] = labels.FromStrings("__name__", "http_requests", "method", "GET", "url", fmt.Sprintf("/api/v1/users/%d", i))
		}
		s.Limit(allLbls[0])
		triggerDemandUpdate(s)
		b.ReportAllocs()
		b.ResetTimer()
		var counter atomic.Int64
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				i := int(counter.Add(1))
				s.Limit(allLbls[i%n])
			}
		})
	})
}

// triggerDemandUpdate force runs the demand-update path of doPeriodicMaintenance,
// re-evaluating overLimit from current sketch estimates.
func triggerDemandUpdate(s *PerLabelLimiter) {
	ch := make(chan time.Time, 1)
	s.demandUpdateChan = ch
	ch <- time.Now()
	s.doPeriodicMaintenance()
}

// triggerPrune force runs the prune path of doPeriodicMaintenance, advancing the sketch ring by one step.
func triggerPrune(s *PerLabelLimiter) {
	ch := make(chan time.Time, 1)
	s.pruneChan = ch
	ch <- time.Now()
	s.doPeriodicMaintenance()
}
