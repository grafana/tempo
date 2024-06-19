package registry

import (
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func Test_native_histogram(t *testing.T) {
	// Note this test does not verify the correctness of the native histograms

	var seriesAdded int
	onAdd := func(count uint32) bool {
		seriesAdded += int(count)
		return true
	}

	h := newNativeHistogram("my_histogram", []float64{1, 2}, onAdd, nil, "trace_id")

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 1.5, "trace-2", 1.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
	}
	expectedExemplars := []exemplarSample{
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-1"}),
			Value:  1.0,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-2"}),
			Value:  1.5,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 1.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 1.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 4.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, collectionTimeMs, 1),
	}
	expectedExemplars = []exemplarSample{
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-2.2"}),
			Value:  2.5,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
			Value:  3.0,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 20.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 13.5)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 1.0, "trace-3", 7.5)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 22),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 54.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 22),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-3"}, collectionTimeMs, 22.0),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-3"}, collectionTimeMs, 51.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 7.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 7.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, collectionTimeMs, 22.0),
	}
	expectedExemplars = []exemplarSample{
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-2.2"}),
			Value:  2.5,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
			Value:  1.0,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"trace_id": "trace-3"}),
			Value:  3.0,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)
}

// Duplicate labels should not grow the series count.
func Test_ObserveWithExemplar_duplicate(t *testing.T) {
	var seriesAdded int
	onAdd := func(count uint32) bool {
		seriesAdded += int(count)
		return true
	}
	h := newNativeHistogram("my_histogram", []float64{0.1, 0.2}, onAdd, nil, "trace_id")

	lv := newLabelValueCombo([]string{"label"}, []string{"value-1"})

	h.ObserveWithExemplar(lv, 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(lv, 1.1, "trace-1", 1.0)
	assert.Equal(t, 1, seriesAdded)
}
