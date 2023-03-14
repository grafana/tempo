package registry

import (
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
	onAdd := func(count uint32) bool {
		seriesAdded++
		return true
	}

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, onAdd, nil)

	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 1.5, "trace-2", 1.0)

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
			Labels: labels.FromMap(map[string]string{"traceID": "trace-1"}),
			Value:  1.0,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "trace-2"}),
			Value:  1.5,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 2.5, "trace-2.2", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-3"}), 3.0, "trace-3", 1.0)

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
			Labels: labels.FromMap(map[string]string{"traceID": "trace-2.2"}),
			Value:  2.5,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "trace-3"}),
			Value:  3.0,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 2.5, "trace-2.2", 20.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-3"}), 3.0, "trace-3", 13.5)
	h.ObserveWithExemplar(newLabelValues([]string{"value-3"}), 1.0, "trace-3", 7.5)

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
			Labels: labels.FromMap(map[string]string{"traceID": "trace-2.2"}),
			Value:  2.5,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "trace-3"}),
			Value:  1.0,
			Ts:     collectionTimeMs,
		}),
		newExemplar(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, exemplar.Exemplar{
			Labels: labels.FromMap(map[string]string{"traceID": "trace-3"}),
			Value:  3.0,
			Ts:     collectionTimeMs,
		}),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)
}

func Test_histogram_invalidLabelValues(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

	assert.Panics(t, func() {
		h.ObserveWithExemplar(nil, 1.0, "", 1.0)
	})
	assert.Panics(t, func() {
		h.ObserveWithExemplar(newLabelValues([]string{"value-1", "value-2"}), 1.0, "", 1.0)
	})
}

func Test_histogram_cantAdd(t *testing.T) {
	canAdd := false
	onAdd := func(count uint32) bool {
		assert.Equal(t, uint32(5), count)
		return canAdd
	}

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, onAdd, nil)

	// allow adding new series
	canAdd = true

	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 1.5, "", 1.0)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 2.5, "", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-3"}), 3.0, "", 1.0)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, nil)
}

func Test_histogram_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(5), count)
		removedSeries++
	}

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, onRemove)

	timeMs := time.Now().UnixMilli()
	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 1.5, "", 1.0)

	h.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, nil)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 2.5, "", 1.0)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 5, expectedSamples, nil)
}

func Test_histogram_externalLabels(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "", 1.0)
	h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 1.5, "", 1.0)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1", "external_label": "external_value"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf", "external_label": "external_value"}, collectionTimeMs, 1),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, map[string]string{"external_label": "external_value"}, 10, expectedSamples, nil)
}

func Test_histogram_concurrencyDataRace(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

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
			h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "", 1.0)
			h.ObserveWithExemplar(newLabelValues([]string{"value-2"}), 1.5, "", 1.0)
		})
	}

	// this goroutine constantly creates new series
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		h.ObserveWithExemplar(newLabelValues([]string{string(s)}), 1.0, "", 1.0)
	})

	go accessor(func() {
		_, err := h.collectMetrics(&noopAppender{}, 0, nil)
		assert.NoError(t, err)
	})

	go accessor(func() {
		h.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_histogram_concurrencyCorrectness(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

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
					h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 2.0, "", 1.0)
					totalCount.Inc()
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(end)

	wg.Wait()

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 2*float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, float64(totalCount.Load())),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 5, expectedSamples, nil)
}

func Test_histogram_span_multiplier(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)
	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 1.0, "", 1.5)
	h.ObserveWithExemplar(newLabelValues([]string{"value-1"}), 2.0, "", 5)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, collectionTimeMs, 11.5),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, collectionTimeMs, 6.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, collectionTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, collectionTimeMs, 6.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, collectionTimeMs, 6.5),
	}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 5, expectedSamples, nil)
}
