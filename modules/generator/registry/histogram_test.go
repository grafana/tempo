package registry

import (
	"math/rand"
	"sync"
	"testing"
	"time"

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

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples)

	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)
	h.Observe(NewLabelValues([]string{"value-3"}), 3.0)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples)
}

func Test_histogram_invalidLabelValues(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

	assert.Panics(t, func() {
		h.Observe(nil, 1.0)
	})
	assert.Panics(t, func() {
		h.Observe(NewLabelValues([]string{"value-1", "value-2"}), 1.0)
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

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples)

	// block new series - existing series can still be updated
	canAdd = false

	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)
	h.Observe(NewLabelValues([]string{"value-3"}), 3.0)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples)
}

func Test_histogram_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(5), count)
		removedSeries++
	}

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, onRemove)

	timeMs := time.Now().UnixMilli()
	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 5, expectedSamples)
}

func Test_histogram_externalLabels(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0}, nil, nil)

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

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
	collectMetricAndAssert(t, h, collectionTimeMs, map[string]string{"external_label": "external_value"}, 10, expectedSamples)
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
			h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
			h.Observe(NewLabelValues([]string{"value-2"}), 1.5)
		})
	}

	// this goroutine constantly creates new series
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		h.Observe(NewLabelValues([]string{string(s)}), 1.0)
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
					h.Observe(NewLabelValues([]string{"value-1"}), 2.0)
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
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 5, expectedSamples)
}
