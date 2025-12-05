package registry

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Test_counter(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	endOfLastMinuteMs = getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-3"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-3"}, collectionTimeMs, 3),
	}

	collectMetricAndAssert(t, c, collectionTimeMs, 3, expectedSamples, nil)
}

func TestCounterDifferentLabels(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"another_label"}, []string{"another_value"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "another_label": "another_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "another_label": "another_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_counter_cantAdd(t *testing.T) {
	canAdd := false
	lifecycler := &mockLimiter{
		onAddFunc: func(_ uint64, count uint32) bool {
			assert.Equal(t, uint32(1), count)
			return canAdd
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	// allow adding new series
	canAdd = true

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_counter_removeStaleSeries(t *testing.T) {
	var removedSeries int
	lifecycler := &mockLimiter{
		onDeleteFunc: func(_ uint64, count uint32) {
			assert.Equal(t, uint32(1), count)
			removedSeries++
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	timeMs := time.Now().UnixMilli()
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	c.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	time.Sleep(10 * time.Millisecond)
	timeMs = time.Now().UnixMilli()

	// update value-2 series
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	c.removeStaleSeries(timeMs)

	assert.Equal(t, 1, removedSeries)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 1, expectedSamples, nil)
}

func Test_counter_externalLabels(t *testing.T) {
	c := newCounter("my_counter", noopLimiter, map[string]string{"external_label": "external_value"}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_counter_concurrencyDataRace(t *testing.T) {
	c := newCounter("my_counter", noopLimiter, map[string]string{}, 15*time.Minute)

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
			c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
			c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 1.0)
		})
	}

	// this goroutine constantly creates new series
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		c.Inc(buildTestLabels([]string{"label"}, []string{string(s)}), 1.0)
	})

	go accessor(func() {
		err := c.collectMetrics(&noopAppender{}, 0)
		assert.NoError(t, err)
	})

	go accessor(func() {
		c.removeStaleSeries(time.Now().UnixMilli())
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_counter_concurrencyCorrectness(t *testing.T) {
	c := newCounter("my_counter", noopLimiter, map[string]string{}, 15*time.Minute)

	var wg sync.WaitGroup
	end := make(chan struct{})

	totalCount := atomic.NewFloat64(0)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-end:
					return
				default:
					c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
					totalCount.Add(1)
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
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, totalCount.Load()),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 1, expectedSamples, nil)
}

func collectMetricAndAssert(t *testing.T, m metric, collectionTimeMs int64, expectedActiveSeries int, expectedSamples []sample, expectedExemplars []exemplarSample) {
	appender := &capturingAppender{}

	err := m.collectMetrics(appender, collectionTimeMs)
	assert.NoError(t, err)

	assert.False(t, appender.isCommitted)
	assert.False(t, appender.isRolledback)
	assert.ElementsMatch(t, expectedSamples, appender.samples)
	fmt.Println("Expected samples:")
	for _, expectedSample := range expectedSamples {
		fmt.Println(" - ", expectedSample.l, expectedSample.v)
	}
	fmt.Println("Appender samples:")
	for _, sample := range appender.samples {
		fmt.Println(" - ", sample.l, sample.v)
	}
	assert.ElementsMatch(t, expectedExemplars, appender.exemplars)
}

func Test_counter_demandTracking(t *testing.T) {
	c := newCounter("my_counter", noopLimiter, map[string]string{}, 15*time.Minute)

	// Initially, demand should be 0
	assert.Equal(t, 0, c.countSeriesDemand())

	// Add some series
	for i := 0; i < 50; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		c.Inc(lbls, 1.0)
	}

	// Demand should now be approximately 50 (within HLL error)
	demand := c.countSeriesDemand()
	assert.Greater(t, demand, 40, "demand should be close to 50")
	assert.Less(t, demand, 60, "demand should be close to 50")

	// Active series should exactly match
	assert.Equal(t, 50, c.countActiveSeries())
}

func Test_counter_demandVsActiveSeries(t *testing.T) {
	limitReached := false
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !limitReached
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	// Add series up to a point
	for i := 0; i < 30; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		c.Inc(lbls, 1.0)
	}

	assert.Equal(t, 30, c.countActiveSeries())

	// Hit the limit
	limitReached = true

	// Try to add more series (they should be rejected)
	for i := 30; i < 60; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		c.Inc(lbls, 1.0)
	}

	// Active series should still be 30
	assert.Equal(t, 30, c.countActiveSeries())

	// But demand should show all attempted series
	demand := c.countSeriesDemand()
	assert.Greater(t, demand, 50, "demand should track all attempted series")
	assert.Greater(t, demand, c.countActiveSeries(), "demand should exceed active series")
}

func Test_counter_demandDecay(t *testing.T) {
	c := newCounter("my_counter", noopLimiter, map[string]string{}, 15*time.Minute)

	// Add series
	for i := 0; i < 40; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		c.Inc(lbls, 1.0)
	}

	initialDemand := c.countSeriesDemand()
	assert.Greater(t, initialDemand, 0)

	// Advance the cardinality tracker enough times to clear the window
	for i := 0; i < 5; i++ {
		c.removeStaleSeries(time.Now().UnixMilli())
	}

	// Demand should have decreased or be zero
	finalDemand := c.countSeriesDemand()
	assert.LessOrEqual(t, finalDemand, initialDemand/2, "demand should significantly decay")
}

func Test_counter_onUpdate(t *testing.T) {
	var seriesUpdated int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return true
		},
		onUpdateFunc: func(_ uint64, count uint32) {
			assert.Equal(t, uint32(1), count)
			seriesUpdated++
		},
	}

	c := newCounter("my_counter", lifecycler, map[string]string{}, 15*time.Minute)

	// Add initial series
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	// No updates yet (new series don't trigger OnUpdate)
	assert.Equal(t, 0, seriesUpdated)

	// Update existing series
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	assert.Equal(t, 1, seriesUpdated)

	// Update both series
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 3.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 4.0)
	assert.Equal(t, 3, seriesUpdated)
}
