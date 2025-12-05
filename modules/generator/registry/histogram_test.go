package registry

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Test_histogram(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	h := newHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "trace_id", nil, 15*time.Minute)

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "trace-2", 1.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0), // Zero entry for value-1 series
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0), // Zero entry for bucket
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0), // Zero entry for value-2 series
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
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
	collectMetricAndAssert(t, h, collectionTimeMs, 10, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 1.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	endOfLastMinuteMs = getEndOfLastMinuteMs(collectionTimeMs)
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
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-3"}, endOfLastMinuteMs, 0), // Zero entry for value-3 series
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-3"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-3"}, collectionTimeMs, 3),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "2"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, endOfLastMinuteMs, 0),
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
	collectMetricAndAssert(t, h, collectionTimeMs, 15, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 20.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 13.5)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-3"}), 1.0, "trace-3", 7.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, 15, expectedSamples, expectedExemplars)
}

func Test_histogram_cantAdd(t *testing.T) {
	canAdd := false
	lifecycler := &mockLimiter{
		onAddFunc: func(_ uint64, count uint32) bool {
			assert.Equal(t, uint32(5), count)
			return canAdd
		},
	}

	h := newHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", nil, 15*time.Minute)

	// allow adding new series
	canAdd = true

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "", 1.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 10, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.5, "", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0, "", 1.0)

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
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 10, expectedSamples, nil)
}

func Test_histogram_removeStaleSeries(t *testing.T) {
	var removedSeries int
	lifecycler := &mockLimiter{
		onDeleteFunc: func(_ uint64, count uint32) {
			assert.Equal(t, uint32(5), count)
			removedSeries++
		},
	}

	h := newHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", nil, 15*time.Minute)

	timeMs := time.Now().UnixMilli()
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "", 1.0)

	h.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 1),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 10, expectedSamples, nil)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.5, "", 1.0)

	h.removeStaleSeries(timeMs)

	assert.Equal(t, 1, removedSeries)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, collectionTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, collectionTimeMs, 4.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 5, expectedSamples, nil)
}

func Test_histogram_externalLabels(t *testing.T) {
	extLabels := map[string]string{"external_label": "external_value"}

	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", extLabels, 15*time.Minute)

	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "", 1.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1", "external_label": "external_value"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf", "external_label": "external_value"}, collectionTimeMs, 1),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 10, expectedSamples, nil)
}

func Test_histogram_concurrencyDataRace(t *testing.T) {
	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", nil, 15*time.Minute)

	end := make(chan struct{})

	accessor := func(f func()) {
		for {
			select {
			case <-end:
				return
			default:
				f()
			}
		}
	}

	for i := 0; i < 4; i++ {
		go accessor(func() {
			h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.0)
			h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.5, "", 1.0)
		})
	}

	// this goroutine constantly creates new series
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{string(s)}), 1.0, "", 1.0)
	})

	go accessor(func() {
		err := h.collectMetrics(&noopAppender{}, 0)
		assert.NoError(t, err)
	})

	go accessor(func() {
		h.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_histogram_concurrencyCorrectness(t *testing.T) {
	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", nil, 15*time.Minute)

	var wg sync.WaitGroup
	end := make(chan struct{})

	totalCount := atomic.NewUint64(0)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-end:
					return
				default:
					h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0, "", 1.0)
					totalCount.Inc()
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(end)

	wg.Wait()

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 2*float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, float64(totalCount.Load())),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 5, expectedSamples, nil)
}

func Test_histogram_span_multiplier(t *testing.T) {
	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", nil, 15*time.Minute)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0, "", 1.5)
	h.ObserveWithExemplar(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0, "", 5)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 6.5),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 11.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 6.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 6.5),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, 5, expectedSamples, nil)
}

func Test_histogram_demandTracking(t *testing.T) {
	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", nil, 15*time.Minute)

	// Initially, demand should be 0
	assert.Equal(t, 0, h.countSeriesDemand())

	// Add some histogram series (each histogram creates multiple series)
	for i := 0; i < 20; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		h.ObserveWithExemplar(lbls, 1.5, "", 1.0)
	}

	// Demand should be approximately 20 histogram series * 5 (sum, count, 3 buckets) = 100 total series
	// Within HLL error
	demand := h.countSeriesDemand()
	assert.Greater(t, demand, 80, "demand should be close to 100")
	assert.Less(t, demand, 120, "demand should be close to 100")

	// Active series should exactly match 20 * 5
	expectedActive := 20 * int(h.activeSeriesPerHistogramSerie())
	assert.Equal(t, expectedActive, h.countActiveSeries())
}

func Test_histogram_activeSeriesPerHistogramSerie(t *testing.T) {
	// Test with 2 buckets (creates: sum, count, bucket1, bucket2, +Inf)
	h := newHistogram("my_histogram", []float64{1.0, 2.0}, noopLimiter, "", nil, 15*time.Minute)
	assert.Equal(t, uint32(5), h.activeSeriesPerHistogramSerie(), "should be sum + count + 3 buckets")

	// Test with 3 buckets
	h2 := newHistogram("my_histogram", []float64{1.0, 2.0, 3.0}, noopLimiter, "", nil, 15*time.Minute)
	assert.Equal(t, uint32(6), h2.activeSeriesPerHistogramSerie(), "should be sum + count + 4 buckets")

	// Test with no buckets (still has +Inf)
	h3 := newHistogram("my_histogram", []float64{}, noopLimiter, "", nil, 15*time.Minute)
	assert.Equal(t, uint32(3), h3.activeSeriesPerHistogramSerie(), "should be sum + count + +Inf bucket")
}

func Test_histogram_demandVsActiveSeries(t *testing.T) {
	limitReached := false
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !limitReached
		},
	}

	h := newHistogram("my_histogram", []float64{1.0, 2.0}, lifecycler, "", nil, 15*time.Minute)

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
	assert.Greater(t, demand, expectedDemand-20, "demand should track all attempted series")
	assert.Greater(t, demand, h.countActiveSeries(), "demand should exceed active series")
}
