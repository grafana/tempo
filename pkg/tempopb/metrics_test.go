package tempopb_test

import (
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
)

func TestMergeMetricsWithNilSourceDoesNotAllocateDestination(t *testing.T) {
	assert.Nil(t, tempopb.MergeSearchMetrics(nil, nil))
	assert.Nil(t, tempopb.MergeMetadataMetrics(nil, nil))
	assert.Nil(t, tempopb.MergeTraceByIDMetrics(nil, nil))
}

func TestMergeSearchMetrics(t *testing.T) {
	dst := &tempopb.SearchMetrics{
		InspectedTraces:   1,
		InspectedBytes:    2,
		TotalBlocks:       3,
		CompletedJobs:     4,
		TotalJobs:         5,
		TotalBlockBytes:   6,
		InspectedSpans:    7,
		BackendReads:      8,
		BackendBytes:      9,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 10},
	}

	got := tempopb.MergeSearchMetrics(dst, &tempopb.SearchMetrics{
		InspectedTraces:   11,
		InspectedBytes:    12,
		TotalBlocks:       13,
		CompletedJobs:     14,
		TotalJobs:         15,
		TotalBlockBytes:   16,
		InspectedSpans:    17,
		BackendReads:      18,
		BackendBytes:      19,
		AdditionalMetrics: map[string]int64{tempopb.AdditionalMetricCacheHits: 20, tempopb.AdditionalMetricCacheMisses: 21},
	})

	assert.Same(t, dst, got)
	assert.Equal(t, &tempopb.SearchMetrics{
		InspectedTraces: 1 + 11,
		InspectedBytes:  2 + 12,
		TotalBlocks:     3 + 13,
		CompletedJobs:   4 + 14,
		TotalJobs:       5 + 15,
		TotalBlockBytes: 6 + 16,
		InspectedSpans:  7 + 17,
		BackendReads:    8 + 18,
		BackendBytes:    9 + 19,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricCacheHits:   10 + 20,
			tempopb.AdditionalMetricCacheMisses: 21,
		},
	}, got)
}

func TestMergeMetadataMetrics(t *testing.T) {
	got := tempopb.MergeMetadataMetrics(nil, &tempopb.MetadataMetrics{
		InspectedBytes:  1,
		TotalJobs:       2,
		CompletedJobs:   3,
		TotalBlocks:     4,
		TotalBlockBytes: 5,
		BackendReads:    6,
		BackendBytes:    7,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricPagesInspected: 8,
		},
	})

	assert.Equal(t, &tempopb.MetadataMetrics{
		InspectedBytes:  1,
		TotalJobs:       2,
		CompletedJobs:   3,
		TotalBlocks:     4,
		TotalBlockBytes: 5,
		BackendReads:    6,
		BackendBytes:    7,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricPagesInspected: 8,
		},
	}, got)
}

func TestMergeTraceByIDMetrics(t *testing.T) {
	got := tempopb.MergeTraceByIDMetrics(&tempopb.TraceByIDMetrics{
		InspectedBytes: 1,
		BackendReads:   2,
		BackendBytes:   3,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricRowGroupsSkipped: 4,
		},
	}, &tempopb.TraceByIDMetrics{
		InspectedBytes: 5,
		BackendReads:   6,
		BackendBytes:   7,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricRowGroupsSkipped:   8,
			tempopb.AdditionalMetricRowGroupsInspected: 9,
		},
	})

	assert.Equal(t, &tempopb.TraceByIDMetrics{
		InspectedBytes: 1 + 5,
		BackendReads:   2 + 6,
		BackendBytes:   3 + 7,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricRowGroupsSkipped:   4 + 8,
			tempopb.AdditionalMetricRowGroupsInspected: 9,
		},
	}, got)
}
