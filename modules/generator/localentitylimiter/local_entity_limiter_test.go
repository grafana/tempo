package localentitylimiter

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

func TestLocalEntityLimiter(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 10
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	l := New(maxFunc, "test", limitLogger)
	overflowLabels := labels.FromStrings("metric_overflow", "true")

	var firstHash uint64
	for i := 0; i < 10; i++ {
		testLabels := labels.FromStrings("test", string(rune('a'+i)))
		hash := testLabels.Hash()
		if i == 0 {
			firstHash = hash
		}
		returnedLabels, _ := l.OnAdd(hash, 1, testLabels)
		require.Equal(t, testLabels, returnedLabels, "entity should be accepted")
	}

	testLabels := labels.FromStrings("test", "k")
	returnedLabels, _ := l.OnAdd(testLabels.Hash(), 1, testLabels)
	require.Equal(t, overflowLabels, returnedLabels, "entity should be rejected at limit")

	// Delete one of the accepted entities
	l.OnDelete(firstHash, 1)

	testLabels2 := labels.FromStrings("test", "l")
	returnedLabels2, _ := l.OnAdd(testLabels2.Hash(), 1, testLabels2)
	require.Equal(t, testLabels2, returnedLabels2, "entity should be accepted after delete")
}

func TestLocalEntityLimiter_OnDelete(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 1
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)
	overflowLabels := labels.FromStrings("metric_overflow", "true")

	testLabels1 := labels.FromStrings("test", "value1")
	hash1 := testLabels1.Hash()
	returnedLabels, _ := limiter.OnAdd(hash1, 1, testLabels1)
	require.Equal(t, testLabels1, returnedLabels, "entity should be accepted")
	returnedLabels, _ = limiter.OnAdd(hash1, 1, testLabels1)
	require.Equal(t, testLabels1, returnedLabels, "same entity should be accepted")
	returnedLabels, _ = limiter.OnAdd(hash1, 1, testLabels1)
	require.Equal(t, testLabels1, returnedLabels, "same entity should be accepted")

	testLabels2 := labels.FromStrings("test", "value2")
	hash2 := testLabels2.Hash()
	returnedLabels, _ = limiter.OnAdd(hash2, 1, testLabels2)
	require.Equal(t, overflowLabels, returnedLabels, "new entity should be rejected at limit")

	limiter.OnDelete(hash1, 3)
	returnedLabels, _ = limiter.OnAdd(hash2, 1, testLabels2)
	require.Equal(t, testLabels2, returnedLabels, "entity should be accepted after delete")
}

func TestLocalEntityLimiter_OverflowOnDelete(t *testing.T) {
	// This test is to guard against accidental overflow. This is a programming
	// error, but we should be defensive anyway.
	maxFunc := func(string) uint32 {
		return 1
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)

	testLabels1 := labels.FromStrings("test", "value1")
	hash1 := testLabels1.Hash()
	returnedLabels, _ := limiter.OnAdd(hash1, 1, testLabels1)
	require.Equal(t, testLabels1, returnedLabels, "entity should be accepted")

	limiter.OnDelete(hash1, 3)

	testLabels2 := labels.FromStrings("test", "value2")
	hash2 := testLabels2.Hash()
	returnedLabels, _ = limiter.OnAdd(hash2, 1, testLabels2)
	require.Equal(t, testLabels2, returnedLabels, "entity should be accepted after delete")
}

func TestLocalEntityLimiter_TrackEntities_NoMaxEntities(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 0
	}
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(maxFunc, "test", limitLogger)
	for i := 0; i < 10; i++ {
		testLabels := labels.FromStrings("test", string(rune('a'+i)))
		returnedLabels, _ := limiter.OnAdd(testLabels.Hash(), 1, testLabels)
		require.Equal(t, testLabels, returnedLabels, "entity should be accepted when max is 0")
	}
	testLabels := labels.FromStrings("test", "k")
	returnedLabels, _ := limiter.OnAdd(testLabels.Hash(), 1, testLabels)
	require.Equal(t, testLabels, returnedLabels, "entity should be accepted when max is 0")
}

func TestLocalEntityLimiter_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	currentLimit := uint32(10)
	limitLogger := tempo_log.NewRateLimitedLogger(1, log.NewNopLogger())
	limiter := New(func(string) uint32 {
		return currentLimit
	}, "test", limitLogger)

	overflowLabels := labels.FromStrings("metric_overflow", "true")
	for i := 0; i < 10; i++ {
		testLabels := labels.FromStrings("test", string(rune('a'+i)))
		returnedLabels, _ := limiter.OnAdd(testLabels.Hash(), 1, testLabels)
		require.Equal(t, testLabels, returnedLabels, "entity should be accepted")
	}
	testLabels := labels.FromStrings("test", "k")
	returnedLabels, _ := limiter.OnAdd(testLabels.Hash(), 1, testLabels)
	require.Equal(t, overflowLabels, returnedLabels, "entity should be rejected at limit")

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

	var deleteHash uint64
	for i := 0; i < 10; i++ {
		testLabels := labels.FromStrings("test", string(rune('a'+i+10)))
		hash := testLabels.Hash()
		if i == 0 {
			deleteHash = hash
		}
		returnedLabels, _ := limiter.OnAdd(hash, 1, testLabels)
		require.Equal(t, testLabels, returnedLabels, "entity should be accepted when max is 0")
	}
	limiter.OnDelete(deleteHash, 1)

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
