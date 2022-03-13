package registry

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Test_metric(t *testing.T) {
	var seriesAdded int
	onAdd := func() bool {
		seriesAdded++
		return true
	}

	m := newMetric("my_metric", []string{"label"}, onAdd, nil)

	m.add([]string{"value-1"}, 1.0)
	m.add([]string{"value-2"}, 2.0)

	assert.Equal(t, 2, seriesAdded)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 2, expectedSamples)

	m.add([]string{"value-2"}, 2.0)
	m.add([]string{"value-3"}, 3.0)

	assert.Equal(t, 3, seriesAdded)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 4),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-3"}, scrapeTimeMs, 3),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 3, expectedSamples)
}

func Test_metric_invalidLabelValues(t *testing.T) {
	m := newMetric("my_metric", []string{"label"}, nil, nil)

	assert.Panics(t, func() {
		m.add(nil, 1.0)
	})
	assert.Panics(t, func() {
		m.add([]string{"value-1", "value-2"}, 1.0)
	})
}

func Test_metric_cantAdd(t *testing.T) {
	canAdd := false
	onAdd := func() bool {
		return canAdd
	}

	m := newMetric("my_metric", []string{"label"}, onAdd, nil)

	// allow adding new series
	canAdd = true

	m.add([]string{"value-1"}, 1.0)
	m.add([]string{"value-2"}, 2.0)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 2, expectedSamples)

	// block new series - existing series can still be updated
	canAdd = false

	m.add([]string{"value-2"}, 2.0)
	m.add([]string{"value-3"}, 3.0)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 4),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 2, expectedSamples)
}

func Test_metric_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func() {
		removedSeries++
	}

	m := newMetric("my_metric", []string{"label"}, nil, onRemove)

	timeMs := time.Now().UnixMilli()
	m.add([]string{"value-1"}, 1.0)
	m.add([]string{"value-2"}, 2.0)

	m.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 2, expectedSamples)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	m.add([]string{"value-2"}, 2.0)

	m.removeStaleSeries(timeMs)

	assert.Equal(t, 1, removedSeries)

	scrapeTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2"}, scrapeTimeMs, 4),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 1, expectedSamples)
}

func Test_metric_externalLabels(t *testing.T) {
	m := newMetric("my_metric", []string{"label"}, nil, nil)

	m.add([]string{"value-1"}, 1.0)
	m.add([]string{"value-2"}, 2.0)

	scrapeTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1", "external_label": "external_value"}, scrapeTimeMs, 1),
		newSample(map[string]string{"__name__": "my_metric", "label": "value-2", "external_label": "external_value"}, scrapeTimeMs, 2),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, map[string]string{"external_label": "external_value"}, 2, expectedSamples)
}

func Test_metric_concurrencyDataRace(t *testing.T) {
	m := newMetric("my_metric", []string{"label"}, nil, nil)

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
			m.add([]string{"value-1"}, 1.0)
			m.add([]string{"value-2"}, 1.0)
		})
	}

	// this goroutine constantly creates new series
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		m.add([]string{string(s)}, 0)
	})

	go accessor(func() {
		_, err := m.scrape(&noopAppender{}, 0, nil)
		assert.NoError(t, err)
	})

	go accessor(func() {
		m.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_metric_concurrencyCorrectness(t *testing.T) {
	m := newMetric("my_metric", []string{"label"}, nil, nil)

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
					m.add([]string{"value-1"}, 1.0)
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
		newSample(map[string]string{"__name__": "my_metric", "label": "value-1"}, scrapeTimeMs, float64(totalCount.Load())),
	}
	scrapeMetricAndAssert(t, m, scrapeTimeMs, nil, 1, expectedSamples)
}

func scrapeMetricAndAssert(t *testing.T, m *metric, scrapeTimeMs int64, externalLabels map[string]string, expectedActiveSeries int, expectedSamples []sample) {
	appender := &capturingAppender{}

	activeSeries, err := m.scrape(appender, scrapeTimeMs, externalLabels)
	assert.NoError(t, err)
	assert.Equal(t, expectedActiveSeries, activeSeries)

	assert.False(t, appender.isCommitted)
	assert.False(t, appender.isRolledback)
	assert.ElementsMatch(t, expectedSamples, appender.samples)
}

func Test_hashLabelValues(t *testing.T) {
	testCases := []struct {
		v1, v2 []string
	}{
		{[]string{"foo"}, []string{"bar"}},
		{[]string{"foo", "bar"}, []string{"foob", "ar"}},
		{[]string{"foo", "bar"}, []string{"bar", "foo"}},
		{[]string{"foo_", "bar"}, []string{"foo", "_bar"}},
		{[]string{"123", "456"}, []string{"1234", "56"}},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s_%s", strings.Join(testCase.v1, ","), strings.Join(testCase.v2, ",")), func(t *testing.T) {
			assert.NotEqual(t, hashLabelValues(testCase.v1), hashLabelValues(testCase.v2))
		})
	}
}
