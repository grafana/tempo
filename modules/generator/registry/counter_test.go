package registry

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Test_counter(t *testing.T) {
	var seriesAdded int
	onAdd := func(count uint32) bool {
		seriesAdded++
		return true
	}

	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(onAdd, noopOnRemove)

	c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 2, expectedSamples)

	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)
	c.Inc(NewLabelValues([]string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 4),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-3"}, scrapeTimeMs, 3),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 3, expectedSamples)
}

func Test_counter_invalidLabelValues(t *testing.T) {
	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(noopOnAdd, noopOnRemove)

	assert.Panics(t, func() {
		c.Inc(nil, 1.0)
	})
	assert.Panics(t, func() {
		c.Inc(NewLabelValues([]string{"value-1", "value-2"}), 1.0)
	})
}

func Test_counter_cantAdd(t *testing.T) {
	canAdd := false
	onAdd := func(count uint32) bool {
		assert.Equal(t, uint32(1), count)
		return canAdd
	}

	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(onAdd, noopOnRemove)

	// allow adding new series
	canAdd = true

	c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 2, expectedSamples)

	// block new series - existing series can still be updated
	canAdd = false

	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)
	c.Inc(NewLabelValues([]string{"value-3"}), 3.0)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 4),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 2, expectedSamples)
}

func Test_counter_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(1), count)
		removedSeries++
	}

	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(noopOnAdd, onRemove)

	timeMs := time.Now().UnixMilli()
	c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)

	c.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 2, expectedSamples)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)

	c.removeStaleSeries(timeMs)

	assert.Equal(t, 1, removedSeries)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, scrapeTimeMs, 4),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 1, expectedSamples)
}

func Test_counter_externalLabels(t *testing.T) {
	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(noopOnAdd, noopOnRemove)

	c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
	c.Inc(NewLabelValues([]string{"value-2"}), 2.0)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "external_label": "external_value"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, map[string]string{"external_label": "external_value"}, 2, expectedSamples)
}

func Test_counter_concurrencyDataRace(t *testing.T) {
	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(noopOnAdd, noopOnRemove)

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
			c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
			c.Inc(NewLabelValues([]string{"value-2"}), 1.0)
		})
	}

	// this goroutine constantly creates new series
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		c.Inc(NewLabelValues([]string{string(s)}), 1.0)
	})

	go accessor(func() {
		_, err := c.scrape(&noopAppender{}, 0, nil)
		assert.NoError(t, err)
	})

	go accessor(func() {
		c.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_counter_concurrencyCorrectness(t *testing.T) {
	c := newCounter("my_counter", []string{"label"})
	c.setCallbacks(noopOnAdd, noopOnRemove)

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
					c.Inc(NewLabelValues([]string{"value-1"}), 1.0)
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
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, scrapeTimeMs, float64(totalCount.Load())),
	}
	scrapeMetricAndAssert(t, c, scrapeTimeMs, nil, 1, expectedSamples)
}

func scrapeMetricAndAssert(t *testing.T, m metric, scrapeTimeMs int64, externalLabels map[string]string, expectedActiveSeries int, expectedSamples []sample) {
	appender := &capturingAppender{}

	activeSeries, err := m.scrape(appender, scrapeTimeMs, externalLabels)
	assert.NoError(t, err)
	assert.Equal(t, expectedActiveSeries, activeSeries)

	assert.False(t, appender.isCommitted)
	assert.False(t, appender.isRolledback)
	assert.ElementsMatch(t, expectedSamples, appender.samples)
}

func noopOnAdd(count uint32) bool {
	return true
}

func noopOnRemove(count uint32) {
}
