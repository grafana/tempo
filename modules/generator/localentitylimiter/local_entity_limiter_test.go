package localentitylimiter

import (
	"hash/maphash"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestLocalEntityLimiter_TrackEntities(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		maxFunc := func(string) uint32 {
			return 10
		}
		limiter := NewLocalEntityLimiter(maxFunc, 1*time.Hour)

		ticks := time.NewTicker(1 * time.Minute)
		t.Cleanup(ticks.Stop)

		go func() {
			for {
				select {
				case <-ticks.C:
					limiter.Prune(t.Context())
				case <-t.Context().Done():
					return
				}
			}
		}()

		// First we fill the limiter
		rejected, err := limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// Now check that it rejects new entities
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11})
		require.NoError(t, err)
		require.Equal(t, []uint64{11}, rejected)

		// After half the stale duration, we update 9/10 entities to they stick around
		time.Sleep(30 * time.Minute)
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// After the stale duration, the entity 10 should have been removed Note
		// we add 2 minutes here to account for any delay in starting the prune
		// loop. The test is still fully deterministic due to synctest.
		time.Sleep(32 * time.Minute)

		// Now we can admit entity 11
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11})
		require.NoError(t, err)
		require.Empty(t, rejected)

		// And finally, check that entity 10 is rejected
		rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{10})
		require.NoError(t, err)
		require.Equal(t, []uint64{10}, rejected)
	})
}

func TestLocalEntityLimiter_TrackEntities_NoMaxEntities(t *testing.T) {
	maxFunc := func(string) uint32 {
		return 0
	}
	limiter := NewLocalEntityLimiter(maxFunc, 1*time.Hour)
	rejected, err := limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	require.NoError(t, err)
	require.Empty(t, rejected)

	rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11})
	require.NoError(t, err)
	require.Empty(t, rejected)
}

func TestLocalEntityLimiter_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	currentLimit := uint32(10)
	limiter := NewLocalEntityLimiter(func(string) uint32 {
		return currentLimit
	}, 1*time.Hour)

	rejected, err := limiter.TrackEntities(t.Context(), "test", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	require.NoError(t, err)
	require.Empty(t, rejected)

	require.NoError(t, err)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_active_entities The number of active entities in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_active_entities gauge
		tempo_metrics_generator_registry_active_entities{tenant="test"} 10
		# HELP tempo_metrics_generator_registry_max_active_entities The maximum number of entities allowed to be active in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_max_active_entities gauge
		tempo_metrics_generator_registry_max_active_entities{tenant="test"} 10
	`), "tempo_metrics_generator_registry_active_entities", "tempo_metrics_generator_registry_max_active_entities")
	require.NoError(t, err)

	currentLimit = 0

	rejected, err = limiter.TrackEntities(t.Context(), "test", []uint64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	require.NoError(t, err)
	require.Empty(t, rejected)

	err = testutil.CollectAndCompare(reg, strings.NewReader(`
		# HELP tempo_metrics_generator_registry_active_entities The number of active entities in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_active_entities gauge
		tempo_metrics_generator_registry_active_entities{tenant="test"} 20
		# HELP tempo_metrics_generator_registry_max_active_entities The maximum number of entities allowed to be active in the metrics generator registry
		# TYPE tempo_metrics_generator_registry_max_active_entities gauge
		tempo_metrics_generator_registry_max_active_entities{tenant="test"} 0
	`), "tempo_metrics_generator_registry_active_entities", "tempo_metrics_generator_registry_max_active_entities")
	require.NoError(t, err)
}

func TestLocalEntityLimiter_Demand(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics = newMetrics(reg)
	limiter := NewLocalEntityLimiter(func(string) uint32 {
		return 10
	}, 1*time.Hour)

	// We need actual hashes here, because incrementing integers will not have
	// good distribution for the HLL sketch.
	seed := maphash.MakeSeed()

	numHashes := 10000
	hashes := make([]uint64, numHashes)
	for i := range hashes {
		hashes[i] = maphash.Comparable(seed, i)
	}

	rejected, err := limiter.TrackEntities(t.Context(), "test", hashes)
	require.NoError(t, err)
	require.NotEmpty(t, rejected)

	families, err := reg.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "tempo_metrics_generator_registry_entity_demand" {
			continue
		}

		require.Len(t, family.Metric, 1)
		metric := family.Metric[0]
		require.Greater(t, metric.GetGauge().GetValue(), float64(numHashes)*0.75, "entity demand is too low")
	}
}
