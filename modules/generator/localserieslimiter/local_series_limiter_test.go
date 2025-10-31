package localserieslimiter

import (
	"strings"
	"testing"

	"github.com/go-kit/log"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestLocalSeriesLimiter(t *testing.T) {
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return 10
	}, "test", limitLogger)

	for range 10 {
		require.True(t, limiter.OnAdd(0, 1))
	}

	require.False(t, limiter.OnAdd(0, 1))

	limiter.OnDelete(0, 1)

	require.True(t, limiter.OnAdd(0, 1))
}

func TestLocalSeriesLimiter_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return 10
	}, "test", limitLogger)

	for range 10 {
		require.True(t, limiter.OnAdd(0, 1))
	}

	require.False(t, limiter.OnAdd(0, 1))

	limiter.OnDelete(0, 1)

	require.True(t, limiter.OnAdd(0, 1))

	err := testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_series_limited_total The total amount of series not created because of limits per tenant
		# TYPE tempo_metrics_generator_registry_series_limited_total counter
		tempo_metrics_generator_registry_series_limited_total{tenant="test"} 1
	`), "tempo_metrics_generator_registry_series_limited_total")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_active_series The active series per tenant
		# TYPE tempo_metrics_generator_registry_active_series gauge
		tempo_metrics_generator_registry_active_series{tenant="test"} 10
	`), "tempo_metrics_generator_registry_active_series")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_max_active_series The maximum active series per tenant
		# TYPE tempo_metrics_generator_registry_max_active_series gauge
		tempo_metrics_generator_registry_max_active_series{tenant="test"} 10
	`), "tempo_metrics_generator_registry_max_active_series")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_series_added_total The total amount of series created per tenant
		# TYPE tempo_metrics_generator_registry_series_added_total counter
		tempo_metrics_generator_registry_series_added_total{tenant="test"} 11
	`), "tempo_metrics_generator_registry_series_added_total")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_series_removed_total The total amount of series removed after they have become stale per tenant
		# TYPE tempo_metrics_generator_registry_series_removed_total counter
		tempo_metrics_generator_registry_series_removed_total{tenant="test"} 1
	`), "tempo_metrics_generator_registry_series_removed_total")
	require.NoError(t, err)
}
