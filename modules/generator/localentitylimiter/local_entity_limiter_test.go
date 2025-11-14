package localentitylimiter

import (
	"strings"
	"testing"

	"github.com/go-kit/log"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestLocalEntityLimiter(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 10
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	l := New(maxFunc, "test", limitLogger)

	for i := 0; i < 10; i++ {
		require.True(t, l.OnAdd(uint64(i), 1))
	}

	require.False(t, l.OnAdd(11, 1))

	l.OnDelete(1, 1)

	require.True(t, l.OnAdd(11, 1))
}

func TestLocalEntityLimiter_OnDelete(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 1
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)
	require.True(t, limiter.OnAdd(1, 1))
	require.True(t, limiter.OnAdd(1, 1))
	require.True(t, limiter.OnAdd(1, 1))
	require.False(t, limiter.OnAdd(2, 1))
	limiter.OnDelete(1, 3)
	require.True(t, limiter.OnAdd(2, 1))
}

func TestLocalEntityLimiter_OverflowOnDelete(t *testing.T) {
	// This test is to guard against accidental overflow. This is a programming
	// error, but we should be defensive anyway.
	maxFunc := func(string) uint32 {
		return 1
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)
	require.True(t, limiter.OnAdd(1, 1))
	limiter.OnDelete(1, 3)
	require.True(t, limiter.OnAdd(2, 1))
}

func TestLocalEntityLimiter_TrackEntities_NoMaxEntities(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 0
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)
	for i := 0; i < 10; i++ {
		require.True(t, limiter.OnAdd(uint64(i), 1))
	}
	require.True(t, limiter.OnAdd(11, 1))
}

func TestLocalEntityLimiter_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	currentLimit := uint32(10)
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return currentLimit
	}, "test", limitLogger)

	for i := 0; i < 10; i++ {
		require.True(t, limiter.OnAdd(uint64(i), 1))
	}
	require.False(t, limiter.OnAdd(uint64(10), 1))

	err := testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_active_entities The number of active entities in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_active_entities gauge
		tempo_metrics_generator_registry_active_entities{tenant="test"} 10
		# HELP tempo_metrics_generator_registry_max_active_entities The maximum number of entities allowed to be active in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_max_active_entities gauge
		tempo_metrics_generator_registry_max_active_entities{tenant="test"} 10
	`), "tempo_metrics_generator_registry_active_entities", "tempo_metrics_generator_registry_max_active_entities")
	require.NoError(t, err)

	currentLimit = 0

	for i := 0; i < 10; i++ {
		require.True(t, limiter.OnAdd(uint64(i+10), 1))
	}
	limiter.OnDelete(uint64(10), 1)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_active_entities The number of active entities in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_active_entities gauge
		tempo_metrics_generator_registry_active_entities{tenant="test"} 19
		# HELP tempo_metrics_generator_registry_max_active_entities The maximum number of entities allowed to be active in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_max_active_entities gauge
		tempo_metrics_generator_registry_max_active_entities{tenant="test"} 0
	`), "tempo_metrics_generator_registry_active_entities", "tempo_metrics_generator_registry_max_active_entities")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_entities_limited_total The total amount of entities not created because of limits per tenant
		# TYPE tempo_metrics_generator_registry_entities_limited_total counter
		tempo_metrics_generator_registry_entities_limited_total{tenant="test"} 1
	`), "tempo_metrics_generator_registry_entities_limited_total")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_entities_added_total The total amount of entities created per tenant
		# TYPE tempo_metrics_generator_registry_entities_added_total counter
		tempo_metrics_generator_registry_entities_added_total{tenant="test"} 20
	`), "tempo_metrics_generator_registry_entities_added_total")
	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_entities_removed_total The total amount of entities removed after they have become stale per tenant
		# TYPE tempo_metrics_generator_registry_entities_removed_total counter
		tempo_metrics_generator_registry_entities_removed_total{tenant="test"} 1
	`), "tempo_metrics_generator_registry_entities_removed_total")
	require.NoError(t, err)
}
