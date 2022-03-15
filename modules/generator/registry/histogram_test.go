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

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(onAdd, noopOnRemove)

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

	assert.Equal(t, 2, seriesAdded)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 1),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 10, expectedSamples)

	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)
	h.Observe(NewLabelValues([]string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 4.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-3"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-3"}, scrapeTimeMs, 3),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "2"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-3", "le": "+Inf"}, scrapeTimeMs, 1),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 15, expectedSamples)
}

func Test_histogram_invalidLabelValues(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(noopOnAdd, noopOnRemove)

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

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(onAdd, noopOnRemove)

	// allow adding new series
	canAdd = true

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 1),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 10, expectedSamples)

	// block new series - existing series can still be updated
	canAdd = false

	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)
	h.Observe(NewLabelValues([]string{"value-3"}), 3.0)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 4.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 10, expectedSamples)
}

func Test_histogram_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(5), count)
		removedSeries++
	}

	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(noopOnAdd, onRemove)

	timeMs := time.Now().UnixMilli()
	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

	h.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 1),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 10, expectedSamples)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	h.Observe(NewLabelValues([]string{"value-2"}), 2.5)

	h.removeStaleSeries(timeMs)

	assert.Equal(t, 1, removedSeries)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2"}, scrapeTimeMs, 2),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2"}, scrapeTimeMs, 4.0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 5, expectedSamples)
}

func Test_histogram_externalLabels(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(noopOnAdd, noopOnRemove)

	h.Observe(NewLabelValues([]string{"value-1"}), 1.0)
	h.Observe(NewLabelValues([]string{"value-2"}), 1.5)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-2", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-2", "external_label": "external_value"}, scrapeTimeMs, 1.5),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "1", "external_label": "external_value"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "2", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-2", "le": "+Inf", "external_label": "external_value"}, scrapeTimeMs, 1),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, map[string]string{"external_label": "external_value"}, 10, expectedSamples)
}

func Test_histogram_concurrencyDataRace(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(noopOnAdd, noopOnRemove)

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
		_, err := h.scrape(&noopAppender{}, 0, nil)
		assert.NoError(t, err)
	})

	go accessor(func() {
		h.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_histogram_concurrencyCorrectness(t *testing.T) {
	h := newHistogram("my_histogram", []string{"label"}, []float64{1.0, 2.0})
	h.setCallbacks(noopOnAdd, noopOnRemove)

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

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_histogram_count", "label": "value-1"}, scrapeTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_sum", "label": "value-1"}, scrapeTimeMs, 2*float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "1"}, scrapeTimeMs, 0),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "2"}, scrapeTimeMs, float64(totalCount.Load())),
		newSample(map[string]string{"__name__": "my_histogram_bucket", "label": "value-1", "le": "+Inf"}, scrapeTimeMs, float64(totalCount.Load())),
	}
	scrapeMetricAndAssert(t, h, scrapeTimeMs, nil, 5, expectedSamples)
}
