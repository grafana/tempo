package ingest

import (
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestPruneUnassignedLagMetrics(t *testing.T) {
	const group = "test-group"

	metricPartitionLag.Reset()
	metricPartitionLagSeconds.Reset()
	t.Cleanup(func() {
		metricPartitionLag.Reset()
		metricPartitionLagSeconds.Reset()
	})

	// Export lag for three partitions, as the exporter goroutine and the fetch loop would.
	for _, p := range []int32{0, 1, 2} {
		metricPartitionLag.WithLabelValues(group, strconv.Itoa(int(p))).Set(float64(p) * 100)
		SetPartitionLagSeconds(group, p, time.Duration(p)*time.Second)
	}
	require.Equal(t, 3, testutil.CollectAndCount(metricPartitionLag))
	require.Equal(t, 3, testutil.CollectAndCount(metricPartitionLagSeconds))

	tracked := map[int32]struct{}{0: {}, 1: {}, 2: {}}

	// Partition 2 is no longer assigned: its series must be pruned from both gauges,
	// and it must be dropped from the tracked set.
	pruneUnassignedLagMetrics(group, []int32{0, 1}, tracked)

	require.Equal(t, 2, testutil.CollectAndCount(metricPartitionLag))
	require.Equal(t, 2, testutil.CollectAndCount(metricPartitionLagSeconds))
	require.Equal(t, map[int32]struct{}{0: {}, 1: {}}, tracked)

	// Still-assigned series keep their values untouched.
	require.Equal(t, float64(100), testutil.ToFloat64(metricPartitionLag.WithLabelValues(group, "1")))

	// A partition newly assigned this tick is recorded as tracked so it can be pruned later.
	pruneUnassignedLagMetrics(group, []int32{0, 1, 3}, tracked)
	require.Equal(t, map[int32]struct{}{0: {}, 1: {}, 3: {}}, tracked)

	// Series for a different group are never touched by another group's reconcile.
	metricPartitionLag.WithLabelValues("other-group", "0").Set(7)
	pruneUnassignedLagMetrics(group, nil, tracked)
	require.Empty(t, tracked)
	require.Equal(t, float64(7), testutil.ToFloat64(metricPartitionLag.WithLabelValues("other-group", "0")))
}
