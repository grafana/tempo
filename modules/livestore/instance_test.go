package livestore

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
)

func instanceWithPushLimits(t *testing.T, maxBytes int, maxTraces int) (*instance, *LiveStore) {
	instance, ls := defaultInstance(t)
	limits, err := overrides.NewOverrides(overrides.Config{
		Defaults: overrides.Overrides{
			Global: overrides.GlobalOverrides{
				MaxBytesPerTrace: maxBytes,
			},
			Ingestion: overrides.IngestionOverrides{
				MaxLocalTracesPerUser: maxTraces,
			},
		},
	}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)
	instance.overrides = limits

	return instance, ls
}

func pushTrace(t *testing.T, instance *instance, tr *tempopb.Trace, id []byte) {
	b, err := tr.Marshal()
	require.NoError(t, err)
	req := &tempopb.PushBytesRequest{
		Traces: []tempopb.PreallocBytes{{Slice: b}},
		Ids:    [][]byte{id},
	}
	instance.pushBytes(time.Now(), req)
}

// TestInstanceLimits verifies MaxBytesPerTrace and MaxLocalTracesPerUser enforcement in livestore.
func TestInstanceLimits(t *testing.T) {
	// Measure a small trace size to derive a reasonable MaxBytesPerTrace
	smallID := test.ValidTraceID(nil)
	small := test.MakeTrace(5, smallID)
	smallBatchSize := small.ResourceSpans[0].Size()

	// Configure limits: allow up to ~1.5x small trace, and max 4 live traces
	maxBytes := smallBatchSize + smallBatchSize/2
	maxTraces := 4

	// bytes - succeeds: push two different traces under size limit
	t.Run("bytes - succeeds", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)
		// two different traces with different ids
		id1 := test.ValidTraceID(nil)
		id2 := test.ValidTraceID(nil)
		pushTrace(t, instance, test.MakeTrace(5, id1), id1)
		pushTrace(t, instance, test.MakeTrace(5, id2), id2)
		require.Equal(t, uint64(2), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// bytes - one fails: second push of the same trace exceeds MaxBytesPerTrace
	t.Run("bytes - one fails", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		id := test.ValidTraceID(nil)
		// First push fits
		pushTrace(t, instance, test.MakeTrace(5, id), id)
		// Second push with same id will exceed combined size (> maxBytes)
		pushTrace(t, instance, test.MakeTrace(5, id), id)
		// Only one live trace stored, and accumulated size should be <= maxBytes
		require.Equal(t, uint64(1), instance.liveTraces.Len())
		require.LessOrEqual(t, instance.liveTraces.Size(), uint64(maxBytes))

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})

	// max traces - too many: only first 4 unique traces are accepted
	t.Run("max traces - too many", func(t *testing.T) {
		instance, ls := instanceWithPushLimits(t, maxBytes, maxTraces)

		for range 10 {
			id := test.ValidTraceID(nil)
			pushTrace(t, instance, test.MakeTrace(1, id), id)
		}
		require.Equal(t, uint64(4), instance.liveTraces.Len())

		err := services.StopAndAwaitTerminated(t.Context(), ls)
		require.NoError(t, err)
	})
}

func TestInstanceNoLimits(t *testing.T) {
	instance, ls := instanceWithPushLimits(t, 0, 0) // no limits by default

	for range 100 {
		id := test.ValidTraceID(nil)
		pushTrace(t, instance, test.MakeTrace(1, id), id)
	}

	assert.Equal(t, uint64(100), instance.liveTraces.Len())
	assert.GreaterOrEqual(t, instance.liveTraces.Size(), uint64(1000))

	err := services.StopAndAwaitTerminated(t.Context(), ls)
	require.NoError(t, err)
}
