package registry

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTenant = "test-tenant"

// Duplicate labels should not grow the series count.
func Test_ObserveWithExemplar_duplicate(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(_ uint64, count uint32) bool {
			seriesAdded += int(count)
			return true
		},
	}

	h := newNativeHistogram("my_histogram", []float64{0.1, 0.2}, lifecycler, "trace_id", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)

	lv := buildTestLabels([]string{"label"}, []string{"value-1"})

	h.ObserveWithExemplar(lv, 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(lv, 1.1, "trace-1", 1.0)
	// In BOTH mode, a single histogram series contributes classic (sum+count+buckets including +Inf)
	// plus one native series.
	assert.Equal(t, int(h.activeSeriesPerHistogramSerie()), seriesAdded)
}

func Test_Histograms(t *testing.T) {
	// A single observations has a label value combo, a value, and a multiplier.
	type observations []struct {
		lbls       labels.Labels
		value      float64
		multiplier float64
		traceID    string
	}

	// A single collection has a few observations, and some expectations.  This
	// allows the test to track the accumulation of observations over a series of
	// collections.
	type collections []struct {
		observations      observations
		expectedSamples   []sample
		expectedExemplars []exemplarSample
	}

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)

	cases := []struct {
		name        string
		buckets     []float64
		collections collections
		// native histogram does not support all features yet
		skipNativeHistogram bool
	}{
		{
			name:    "single collection single observation",
			buckets: []float64{1, 2},
			collections: collections{
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-1"}),
							value:      1.0,
							multiplier: 1.0,
							traceID:    "trace-1",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
							Value:  1.0,
							Ts:     collectionTimeMs,
						}),
					},
				},
			},
		},
		{
			name:    "single collection double observation",
			buckets: []float64{1, 2},
			collections: collections{
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-1"}),
							value:      1.0,
							multiplier: 1.0,
							traceID:    "trace-1",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      1.5,
							multiplier: 1.0,
							traceID:    "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
							Value:  1.0,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  1.5,
							Ts:     collectionTimeMs,
						}),
					},
				},
			},
		},
		{
			name:    "double collection double observation",
			buckets: []float64{1, 2},
			collections: collections{
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-1"}),
							value:      1.0,
							multiplier: 1.0,
							traceID:    "trace-1",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      1.5,
							multiplier: 1.0,
							traceID:    "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
							Value:  1.0,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  1.5,
							Ts:     collectionTimeMs,
						}),
					},
				},
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      2.5,
							multiplier: 1.0,
							traceID:    "trace-2",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-3"}),
							value:      3.0,
							multiplier: 1.0,
							traceID:    "trace-3",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 2),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 4),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 2),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  2.5,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
							Value:  3,
							Ts:     collectionTimeMs,
						}),
					},
				},
			},
		},
		{
			name:    "integer multiplier",
			buckets: []float64{1, 2},
			collections: collections{
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-1"}),
							value:      1.5,
							multiplier: 20.0,
							traceID:    "trace-1",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      3.0,
							multiplier: 13,
							traceID:    "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 20*1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 13),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 13*3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 13),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
							Value:  1.5,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  3,
							Ts:     collectionTimeMs,
						}),
					},
				},
			},
		},
		{
			name:                "many observations with floating point multiplier",
			skipNativeHistogram: true,
			buckets:             []float64{1, 2},
			collections: collections{
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-1"}),
							value:      1.0,
							multiplier: 1.0,
							traceID:    "trace-1",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      1.5,
							multiplier: 1.0,
							traceID:    "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
							Value:  1.0,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  1.5,
							Ts:     collectionTimeMs,
						}),
					},
				},
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      2.5,
							multiplier: 1.0,
							traceID:    "trace-2",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-3"}),
							value:      3.0,
							multiplier: 1.0,
							traceID:    "trace-3",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 2),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 4),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 2),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, endOfLastMinuteMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, endOfLastMinuteMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, collectionTimeMs, 1),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
							Value:  2.5,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
							Value:  3,
							Ts:     collectionTimeMs,
						}),
					},
				},
				{
					observations: observations{
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-2"}),
							value:      2.5,
							multiplier: 20.0,
							traceID:    "trace-2.2",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-3"}),
							value:      3.0,
							multiplier: 13.5,
							traceID:    "trace-3",
						},
						{
							lbls:       buildTestLabels([]string{"label"}, []string{"value-3"}),
							value:      1.0,
							multiplier: 7.5,
							traceID:    "trace-3.3",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 22),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 54.0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 22),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeMs, 22),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-3"}, collectionTimeMs, 51),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 7.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 7.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, collectionTimeMs, 22.0),
					},
					expectedExemplars: []exemplarSample{
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-2.2"}),
							Value:  2.5,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
							Value:  3,
							Ts:     collectionTimeMs,
						}),
						newExemplar(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, exemplar.Exemplar{
							Labels: labels.FromMap(map[string]string{"trace_id": "trace-3.3"}),
							Value:  1,
							Ts:     collectionTimeMs,
						}),
					},
				},
			},
		},
	}

	// Tests a single histogram implementation.
	testHistogram := func(t *testing.T, h Histogram, collections collections) {
		for _, c := range collections {
			appender := &capturingAppender{}
			for _, obs := range c.observations {
				h.ObserveWithExemplar(obs.lbls, obs.value, obs.traceID, obs.multiplier)
			}

			expected := expectedSeriesLen(c.expectedSamples)
			// If we're testing the native histogram in BOTH or NATIVE mode, the active
			// series includes an additional native series per base labelset.
			if _, ok := h.(*nativeHistogram); ok {
				expected += expectedBaseSeriesCount(c.expectedSamples)
			}

			collectMetricsAndAssertSeries(t, h, collectionTimeMs, expected, appender)
			if len(c.expectedSamples) > 0 {
				assertAppenderSamples(t, appender, c.expectedSamples)
			}
			if len(c.expectedExemplars) > 0 {
				assertAppenderExemplars(t, appender, c.expectedExemplars)
			}
		}
	}

	// Tests both classic and native histograms.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("classic", func(t *testing.T) {
				h := newHistogram("test_histogram", tc.buckets, noopLimiter, "trace_id", nil, 15*time.Minute)
				testHistogram(t, h, tc.collections)
			})
			t.Run("native", func(t *testing.T) {
				if tc.skipNativeHistogram {
					t.SkipNow()
				}

				h := newNativeHistogram("test_histogram", tc.buckets, noopLimiter, "trace_id", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)
				testHistogram(t, h, tc.collections)
			})
		})
	}
}

func collectMetricsAndAssertSeries(t *testing.T, m metric, collectionTimeMs int64, expectedSeries int, appender storage.Appender) {
	err := m.collectMetrics(appender, collectionTimeMs)
	require.NoError(t, err)
}

func assertAppenderSamples(t *testing.T, appender *capturingAppender, expectedSamples []sample) {
	t.Run("Samples", func(t *testing.T) {
		fmt.Println("Expected samples:")
		for _, expectedSample := range expectedSamples {
			fmt.Println(" - ", expectedSample.l, expectedSample.v)
		}
		fmt.Println("Appender samples:")
		for _, sample := range appender.samples {
			fmt.Println(" - ", sample.l, sample.v)
		}

		require.Len(t, appender.samples, len(expectedSamples))
		// for i, expected := range expectedSamples {
		// 	assert.Equal(t, expected, appender.samples[i])
		// }

		sort.Slice(expectedSamples[:], func(i, j int) bool {
			return expectedSamples[i].l.String() < expectedSamples[j].l.String()
		})
		// t.Log("Expected samples:")
		// for _, expectedSample := range expectedSamples {
		// 	t.Log(" - ", expectedSample.l, expectedSample.v, expectedSample.t)
		// }

		sort.Slice(appender.samples[:], func(i, j int) bool {
			return appender.samples[i].l.String() < appender.samples[j].l.String()
		})

		// t.Log("Appender samples:")
		// for _, sample := range appender.samples {
		// 	t.Log(" - ", sample.l, sample.v, sample.t)
		// }

		matched := assert.ElementsMatchf(t, expectedSamples, appender.samples, "samples mismatch")
		if !matched {
			for i, expected := range expectedSamples {
				if expected.l.String() != appender.samples[i].l.String() || expected.v != appender.samples[i].v || expected.t != appender.samples[i].t {
					t.Log("Mismatch at index", i)
					t.Log("Expected: ", expected.l, expected.v, expected.t)
					t.Log("Actual:   ", appender.samples[i].l, appender.samples[i].v, appender.samples[i].t)
				}
			}
		}
	})
}

func assertAppenderExemplars(t *testing.T, appender *capturingAppender, expected []exemplarSample) {
	t.Run("Exemplars", func(t *testing.T) {
		require.Len(t, appender.exemplars, len(expected))
		sort.Slice(expected[:], func(i, j int) bool {
			return expected[i].l.String() < expected[j].l.String()
		})

		sort.Slice(appender.exemplars[:], func(i, j int) bool {
			return appender.exemplars[i].l.String() < appender.exemplars[j].l.String()
		})

		matched := assert.ElementsMatchf(t, expected, appender.exemplars, "exemplars mismatch")
		if !matched {
			for i, expected := range expected {
				if expected.l.String() != appender.exemplars[i].l.String() || !expected.e.Equals(appender.exemplars[i].e) {
					t.Log("Mismatch at index", i)
					t.Log("Expected: ", expected.l, expected.e)
					t.Log("Actual:   ", appender.exemplars[i].l, appender.exemplars[i].e)
				}
			}
		}
	})
}

func expectedSeriesLen(samples []sample) int {
	series := make(map[string]struct{})
	for _, s := range samples {
		series[s.l.String()] = struct{}{}
	}
	return len(series)
}

// expectedBaseSeriesCount returns the number of distinct base labelsets in the
// expected classic samples by collapsing away metric name and bucket labels.
// This approximates the number of native histogram series emitted in addition
// to the classic series when using BOTH or NATIVE modes.
func expectedBaseSeriesCount(samples []sample) int {
	base := make(map[string]struct{})
	for _, s := range samples {
		lb := labels.NewBuilder(s.l)
		lb.Del(labels.MetricName)
		lb.Del(labels.BucketLabel)
		key := lb.Labels().String()
		base[key] = struct{}{}
	}
	return len(base)
}

// Test specifically for native-only mode to ensure exemplars work
func Test_NativeOnlyExemplars(t *testing.T) {
	buckets := []float64{1, 2}
	collectionTimeMs := time.Now().UnixMilli()

	t.Run("native_only_with_exemplars", func(t *testing.T) {
		overrides := &mockOverrides{
			nativeHistogramBucketFactor:     1.5,
			nativeHistogramMaxBucketNumber:  10,
			nativeHistogramMinResetDuration: time.Minute,
		}

		// Use HistogramModeNative to test native-only behavior
		h := newNativeHistogram("test_native_histogram", buckets, noopLimiter, "trace_id", HistogramModeNative, nil, testTenant, overrides, 15*time.Minute)

		// Add some observations with exemplars
		lbls := buildTestLabels([]string{"service"}, []string{"test-service"})
		h.ObserveWithExemplar(lbls, 1.5, "trace-123", 1.0)
		h.ObserveWithExemplar(lbls, 0.5, "trace-456", 1.0)

		// Collect metrics
		appender := &capturingAppender{}
		err := h.collectMetrics(appender, collectionTimeMs)
		require.NoError(t, err)
		t.Logf("Captured samples: %d", len(appender.samples))
		t.Logf("Captured exemplars: %d", len(appender.exemplars))

		for i, ex := range appender.exemplars {
			t.Logf("Exemplar %d: labels=%v, value=%f, trace=%v", i, ex.l, ex.e.Value, ex.e.Labels)
		}

		// We should have exemplars
		require.Greater(t, len(appender.exemplars), 0, "Native-only mode should capture exemplars")

		// Verify that exemplars have the correct format for native histograms
		for _, ex := range appender.exemplars {
			// Exemplars should be attached to the main histogram metric, not bucket metrics
			require.Equal(t, "test_native_histogram", ex.l.Get(labels.MetricName), "Exemplar should be attached to main histogram metric")
			require.Contains(t, ex.e.Labels.String(), "trace_id", "Exemplar should contain trace_id")
			require.Greater(t, ex.e.Value, 0.0, "Exemplar should have a value")
		}

		require.Len(t, appender.exemplars, 2, "Should have exactly 2 exemplars for 2 observations")

		require.Equal(t, "trace-456", appender.exemplars[0].e.Labels.Get("trace_id"))
		require.Equal(t, 0.5, appender.exemplars[0].e.Value)
		require.Equal(t, "trace-123", appender.exemplars[1].e.Labels.Get("trace_id"))
		require.Equal(t, 1.5, appender.exemplars[1].e.Value)
	})

	t.Run("native_only_histogram_exemplars", func(t *testing.T) {
		overrides := &mockOverrides{
			nativeHistogramBucketFactor:     1.5,
			nativeHistogramMaxBucketNumber:  10,
			nativeHistogramMinResetDuration: time.Minute,
		}

		// Create a native histogram with empty buckets to force native-only mode
		h := newNativeHistogram("test_native_only", []float64{}, noopLimiter, "trace_id", HistogramModeNative, nil, testTenant, overrides, 15*time.Minute)

		// Add some observations with exemplars
		lbls := buildTestLabels([]string{"service"}, []string{"native-only-xyz"})
		h.ObserveWithExemplar(lbls, 1.5, "trace-native-123", 1.0)
		h.ObserveWithExemplar(lbls, 0.5, "trace-native-456", 1.0)

		// Collect metrics
		appender := &capturingAppender{}
		err := h.collectMetrics(appender, collectionTimeMs)
		require.NoError(t, err)
		t.Logf("Native-only - Captured samples: %d", len(appender.samples))
		t.Logf("Native-only - Captured exemplars: %d", len(appender.exemplars))

		// This test might still use bucket exemplars due to Prometheus's default behavior,
		// but it helps us understand the difference

		require.Len(t, appender.exemplars, 2, "Should have exactly 2 exemplars for 2 observations")

		for i, ex := range appender.exemplars {
			t.Logf("Native-only exemplar %d: labels=%v, value=%f, trace=%v", i, ex.l, ex.e.Value, ex.e.Labels)
		}

		require.Equal(t, "trace-native-456", appender.exemplars[0].e.Labels.Get("trace_id"))
		require.Equal(t, 0.5, appender.exemplars[0].e.Value)
		require.Equal(t, "trace-native-123", appender.exemplars[1].e.Labels.Get("trace_id"))
		require.Equal(t, 1.5, appender.exemplars[1].e.Value)
	})
}

func Test_nativeHistogram_demandTracking(t *testing.T) {
	h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)

	// Initially, demand should be 0
	assert.Equal(t, 0, h.countSeriesDemand())

	// Add some histogram series
	for i := 0; i < 20; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		h.ObserveWithExemplar(lbls, 1.5, "", 1.0)
	}

	// In BOTH mode: sum, count, 3 buckets (classic) + 1 native = 6 series per histogram
	// 20 histograms * 6 = 120 total series (within HLL error)
	demand := h.countSeriesDemand()
	assert.Greater(t, demand, 100, "demand should be close to 120")
	assert.Less(t, demand, 140, "demand should be close to 120")

	// Active series should exactly match
	expectedActive := 20 * int(h.activeSeriesPerHistogramSerie())
	assert.Equal(t, expectedActive, h.countActiveSeries())
}

func Test_nativeHistogram_activeSeriesPerHistogramSerie(t *testing.T) {
	// Test BOTH mode with 2 buckets: sum, count, bucket1, bucket2, +Inf, native = 6
	h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)
	assert.Equal(t, uint32(6), h.activeSeriesPerHistogramSerie(), "BOTH mode should be classic + native")

	// Test NATIVE mode only: 1 native histogram series
	h2 := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", HistogramModeNative, nil, testTenant, &mockOverrides{}, 15*time.Minute)
	assert.Equal(t, uint32(1), h2.activeSeriesPerHistogramSerie(), "NATIVE mode should be 1 series")

	// Test CLASSIC mode with 2 buckets: sum, count, bucket1, bucket2, +Inf = 5
	h3 := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", HistogramModeClassic, nil, testTenant, &mockOverrides{}, 15*time.Minute)
	assert.Equal(t, uint32(5), h3.activeSeriesPerHistogramSerie(), "CLASSIC mode should be sum + count + buckets")

	// Test BOTH mode with 3 buckets: sum, count, 4 buckets, native = 7
	h4 := newNativeHistogram("my_histogram", []float64{1.0, 2.0, 3.0}, noopLimiter, "", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)
	assert.Equal(t, uint32(7), h4.activeSeriesPerHistogramSerie(), "BOTH mode with 3 buckets")
}

func Test_nativeHistogram_demandVsActiveSeries(t *testing.T) {
	limitReached := false
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !limitReached
		},
	}

	h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", HistogramModeNative, nil, testTenant, &mockOverrides{}, 15*time.Minute)

	// Add some histogram series
	for i := 0; i < 10; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		h.ObserveWithExemplar(lbls, 1.5, "", 1.0)
	}

	expectedActive := 10 * int(h.activeSeriesPerHistogramSerie())
	assert.Equal(t, expectedActive, h.countActiveSeries())

	// Hit the limit
	limitReached = true

	// Try to add more series (they should be rejected)
	for i := 10; i < 20; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		h.ObserveWithExemplar(lbls, 1.5, "", 1.0)
	}

	// Active series should not have increased
	assert.Equal(t, expectedActive, h.countActiveSeries())

	// But demand should show all attempted series
	demand := h.countSeriesDemand()
	expectedDemand := 20 * int(h.activeSeriesPerHistogramSerie())
	assert.Greater(t, demand, expectedDemand-10, "demand should track all attempted series")
	assert.Greater(t, demand, h.countActiveSeries(), "demand should exceed active series")
}

func Test_nativeHistogram_onUpdate(t *testing.T) {
	// Test BOTH mode (classic + native)
	t.Run("both_mode", func(t *testing.T) {
		var seriesUpdated int
		lifecycler := &mockLimiter{
			onAddFunc: func(_ uint64, count uint32) bool {
				// BOTH mode with 2 buckets: sum, count, bucket1, bucket2, +Inf, native = 6
				assert.Equal(t, uint32(6), count)
				return true
			},
			onUpdateFunc: func(_ uint64, count uint32) {
				assert.Equal(t, uint32(6), count)
				seriesUpdated++
			},
		}

		h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", HistogramModeBoth, nil, testTenant, &mockOverrides{}, 15*time.Minute)

		// Add initial series (first observation triggers both OnAdd and OnUpdate)
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "", 1.0)

		initialUpdates := seriesUpdated
		assert.Equal(t, 2, initialUpdates, "First observations should trigger OnUpdate")

		// Update existing series
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.5, "", 1.0)
		assert.Equal(t, initialUpdates+1, seriesUpdated)

		// Update both series
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0, "", 1.0)
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.5, "", 1.0)
		assert.Equal(t, initialUpdates+3, seriesUpdated)
	})

	// Test NATIVE mode only
	t.Run("native_mode", func(t *testing.T) {
		var seriesUpdated int
		lifecycler := &mockLimiter{
			onAddFunc: func(_ uint64, count uint32) bool {
				// NATIVE mode: only 1 native histogram series
				assert.Equal(t, uint32(1), count)
				return true
			},
			onUpdateFunc: func(_ uint64, count uint32) {
				assert.Equal(t, uint32(1), count)
				seriesUpdated++
			},
		}

		h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", HistogramModeNative, nil, testTenant, &mockOverrides{}, 15*time.Minute)

		// Add initial series (first observation triggers both OnAdd and OnUpdate)
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

		initialUpdates := seriesUpdated
		assert.Equal(t, 1, initialUpdates, "First observation should trigger OnUpdate")

		// Update existing series multiple times
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.5, "", 1.0)
		assert.Equal(t, initialUpdates+1, seriesUpdated)

		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0, "", 1.0)
		assert.Equal(t, initialUpdates+2, seriesUpdated)

		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.5, "", 1.0)
		assert.Equal(t, initialUpdates+3, seriesUpdated)
	})

	// Test CLASSIC mode
	t.Run("classic_mode", func(t *testing.T) {
		var seriesUpdated int
		lifecycler := &mockLimiter{
			onAddFunc: func(_ uint64, count uint32) bool {
				// CLASSIC mode with 2 buckets: sum, count, bucket1, bucket2, +Inf = 5
				assert.Equal(t, uint32(5), count)
				return true
			},
			onUpdateFunc: func(_ uint64, count uint32) {
				assert.Equal(t, uint32(5), count)
				seriesUpdated++
			},
		}

		h := newNativeHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", HistogramModeClassic, nil, testTenant, &mockOverrides{}, 15*time.Minute)

		// Add initial series (first observation triggers both OnAdd and OnUpdate)
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)

		initialUpdates := seriesUpdated
		assert.Equal(t, 1, initialUpdates, "First observation should trigger OnUpdate")

		// Update existing series
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.5, "", 1.0)
		assert.Equal(t, initialUpdates+1, seriesUpdated)

		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0, "", 1.0)
		assert.Equal(t, initialUpdates+2, seriesUpdated)
	})
}
