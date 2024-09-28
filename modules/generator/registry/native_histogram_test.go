package registry

import (
	"sort"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Duplicate labels should not grow the series count.
func Test_ObserveWithExemplar_duplicate(t *testing.T) {
	var seriesAdded int
	onAdd := func(count uint32) bool {
		seriesAdded += int(count)
		return true
	}

	h := newNativeHistogram("my_histogram", []float64{0.1, 0.2}, onAdd, nil, "trace_id", HistogramModeBoth)

	lv := newLabelValueCombo([]string{"label"}, []string{"value-1"})

	h.ObserveWithExemplar(lv, 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(lv, 1.1, "trace-1", 1.0)
	assert.Equal(t, 1, seriesAdded)
}

func Test_Histograms(t *testing.T) {
	// A single observations has a label value combo, a value, and a multiplier.
	type observations []struct {
		labelValueCombo *LabelValueCombo
		value           float64
		multiplier      float64
		traceID         string
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
	collectionTimeWithOffsetMs := collectionTimeMs - 1

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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-1"}),
							value:           1.0,
							multiplier:      1.0,
							traceID:         "trace-1",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-1"}),
							value:           1.0,
							multiplier:      1.0,
							traceID:         "trace-1",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           1.5,
							multiplier:      1.0,
							traceID:         "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-1"}),
							value:           1.0,
							multiplier:      1.0,
							traceID:         "trace-1",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           1.5,
							multiplier:      1.0,
							traceID:         "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           2.5,
							multiplier:      1.0,
							traceID:         "trace-2",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-3"}),
							value:           3.0,
							multiplier:      1.0,
							traceID:         "trace-3",
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
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-1"}),
							value:           1.5,
							multiplier:      20.0,
							traceID:         "trace-1",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           3.0,
							multiplier:      13,
							traceID:         "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 20*1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 20),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 13),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 13*3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 0),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-1"}),
							value:           1.0,
							multiplier:      1.0,
							traceID:         "trace-1",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           1.5,
							multiplier:      1.0,
							traceID:         "trace-2",
						},
					},
					expectedSamples: []sample{
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           2.5,
							multiplier:      1.0,
							traceID:         "trace-2",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-3"}),
							value:           3.0,
							multiplier:      1.0,
							traceID:         "trace-3",
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
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeWithOffsetMs, 0), // zero count at the beginning
						newSample(map[string]string{"__name__": "test_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
						newSample(map[string]string{"__name__": "test_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
						newSample(map[string]string{"__name__": "test_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
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
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-2"}),
							value:           2.5,
							multiplier:      20.0,
							traceID:         "trace-2.2",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-3"}),
							value:           3.0,
							multiplier:      13.5,
							traceID:         "trace-3",
						},
						{
							labelValueCombo: newLabelValueCombo([]string{"label"}, []string{"value-3"}),
							value:           1.0,
							multiplier:      7.5,
							traceID:         "trace-3.3",
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
				h.ObserveWithExemplar(obs.labelValueCombo, obs.value, obs.traceID, obs.multiplier)
			}

			collectMetricsAndAssertSeries(t, h, collectionTimeMs, expectedSeriesLen(c.expectedSamples), appender)
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
				onAdd := func(count uint32) bool { return true }
				h := newHistogram("test_histogram", tc.buckets, onAdd, nil, "trace_id")
				testHistogram(t, h, tc.collections)
			})
			t.Run("native", func(t *testing.T) {
				if tc.skipNativeHistogram {
					t.SkipNow()
				}

				onAdd := func(count uint32) bool { return true }
				h := newNativeHistogram("test_histogram", tc.buckets, onAdd, nil, "trace_id", HistogramModeBoth)
				testHistogram(t, h, tc.collections)
			})
		})
	}
}

func collectMetricsAndAssertSeries(t *testing.T, m metric, collectionTimeMs int64, expectedSeries int, appender storage.Appender) {
	activeSeries, err := m.collectMetrics(appender, collectionTimeMs, nil)
	require.NoError(t, err)
	require.Equal(t, expectedSeries, activeSeries)
}

func assertAppenderSamples(t *testing.T, appender *capturingAppender, expectedSamples []sample) {
	t.Run("Samples", func(t *testing.T) {
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
