package localserieslimiter

import (
	"strings"
	"testing"

	"github.com/go-kit/log"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestLocalSeriesLimiter(t *testing.T) {
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return 10
	}, "test", limitLogger)
	overflowLabels := labels.FromStrings("metric_overflow", "true")
	testLabels := labels.FromStrings("test", "value")
	hash := testLabels.Hash()

	for range 10 {
		returnedLabels, _ := limiter.OnAdd(hash, 1, testLabels)
		require.Equal(t, testLabels, returnedLabels, "series should be accepted")
	}

	returnedLabels, _ := limiter.OnAdd(hash, 1, testLabels)
	require.Equal(t, overflowLabels, returnedLabels, "series should be rejected at limit")

	limiter.OnDelete(hash, 1)

	returnedLabels, _ = limiter.OnAdd(hash, 1, testLabels)
	require.Equal(t, testLabels, returnedLabels, "series should be accepted after delete")
}

func TestLocalSeriesLimiter_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return 10
	}, "test", limitLogger)
	overflowLabels := labels.FromStrings("metric_overflow", "true")
	testLabels := labels.FromStrings("test", "value")
	hash := testLabels.Hash()

	for range 5 {
		returnedLabels, _ := limiter.OnAdd(hash, 2, testLabels)
		require.Equal(t, testLabels, returnedLabels, "series should be accepted")
	}

	returnedLabels, _ := limiter.OnAdd(hash, 1, testLabels)
	require.Equal(t, overflowLabels, returnedLabels, "series should be rejected at limit")

	limiter.OnDelete(hash, 1)

	returnedLabels, _ = limiter.OnAdd(hash, 1, testLabels)
	require.Equal(t, testLabels, returnedLabels, "series should be accepted after delete")

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
