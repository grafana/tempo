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

func Test_gaugeInc(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	c := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-3"}, collectionTimeMs, 3),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 3, expectedSamples, nil)
}

func TestGaugeDifferentLabels(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	c := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"another_label"}, []string{"another_value"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "another_label": "another_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_gaugeSet(t *testing.T) {
	var seriesAdded int
	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			seriesAdded++
			return true
		},
	}

	c := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	c.Set(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Set(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	c.Set(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)
	c.Set(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-3"}, collectionTimeMs, 3),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 3, expectedSamples, nil)
}

func Test_gauge_cantAdd(t *testing.T) {
	canAdd := false
	lifecycler := &mockLimiter{
		onAddFunc: func(_ uint64, count uint32) bool {
			assert.Equal(t, uint32(1), count)
			return canAdd
		},
	}

	c := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	// allow adding new series
	canAdd = true

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-3"}), 3.0)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_gauge_removeStaleSeries(t *testing.T) {
	var removedSeries int
	lifecycler := &mockLimiter{
		onDeleteFunc: func(_ uint64, count uint32) {
			assert.Equal(t, uint32(1), count)
			removedSeries++
		},
	}

	c := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	timeMs := time.Now().UnixMilli()
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	c.removeStaleSeries(timeMs)

	assert.Equal(t, 0, removedSeries)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
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
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 1, expectedSamples, nil)
}

func Test_gauge_externalLabels(t *testing.T) {
	c := newGauge("my_gauge", noopLimiter, map[string]string{"external_label": "external_value"}, 15*time.Minute)

	c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(buildTestLabels([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 2, expectedSamples, nil)
}

func Test_gauge_concurrencyDataRace(t *testing.T) {
	c := newGauge("my_gauge", noopLimiter, map[string]string{}, 15*time.Minute)

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

func Test_gauge_concurrencyCorrectness(t *testing.T) {
	c := newGauge("my_gauge", noopLimiter, map[string]string{}, 15*time.Minute)

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
					c.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
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
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, float64(totalCount.Load())),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, 1, expectedSamples, nil)
}

func Test_gauge_demandTracking(t *testing.T) {
	g := newGauge("my_gauge", noopLimiter, map[string]string{}, 15*time.Minute)

	// Initially, demand should be 0
	assert.Equal(t, 0, g.countSeriesDemand())

	// Add some series
	for i := 0; i < 50; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		g.Inc(lbls, 1.0)
	}

	// Demand should now be approximately 50 (within HLL error)
	demand := g.countSeriesDemand()
	assert.Greater(t, demand, 40, "demand should be close to 50")
	assert.Less(t, demand, 60, "demand should be close to 50")

	// Active series should exactly match
	assert.Equal(t, 50, g.countActiveSeries())
}

func Test_gauge_demandVsActiveSeries(t *testing.T) {
	limitReached := false

	lifecycler := &mockLimiter{
		onAddFunc: func(uint64, uint32) bool {
			return !limitReached
		},
	}
	g := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	// Add series up to a point
	for i := 0; i < 30; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		g.Set(lbls, float64(i))
	}

	assert.Equal(t, 30, g.countActiveSeries())

	// Hit the limit
	limitReached = true

	// Try to add more series (they should be rejected)
	for i := 30; i < 60; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		g.Set(lbls, float64(i))
	}

	// Active series should still be 30
	assert.Equal(t, 30, g.countActiveSeries())

	// But demand should show all attempted series
	demand := g.countSeriesDemand()
	assert.Greater(t, demand, 50, "demand should track all attempted series")
	assert.Greater(t, demand, g.countActiveSeries(), "demand should exceed active series")
}

func Test_gauge_demandDecay(t *testing.T) {
	lifecycler := &mockLimiter{}
	g := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	// Add series
	for i := 0; i < 40; i++ {
		lbls := buildTestLabels([]string{"label"}, []string{fmt.Sprintf("value-%d", i)})
		g.Inc(lbls, 1.0)
	}

	initialDemand := g.countSeriesDemand()
	assert.Greater(t, initialDemand, 0)

	// Advance the cardinality tracker enough times to clear the window
	for i := 0; i < 5; i++ {
		g.removeStaleSeries(time.Now().Add(time.Hour).UnixMilli())
	}

	// Demand should have decreased or be zero
	finalDemand := g.countSeriesDemand()
	assert.LessOrEqual(t, finalDemand, initialDemand/2, "demand should significantly decay")
}

func Test_gauge_onUpdate(t *testing.T) {
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

	g := newGauge("my_gauge", lifecycler, map[string]string{}, 15*time.Minute)

	// Add initial series
	g.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	g.Set(buildTestLabels([]string{"label"}, []string{"value-2"}), 5.0)

	// No updates yet (new series don't trigger OnUpdate)
	assert.Equal(t, 0, seriesUpdated)

	// Update existing series with Inc
	g.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 2.0)
	assert.Equal(t, 1, seriesUpdated)

	// Update existing series with Set
	g.Set(buildTestLabels([]string{"label"}, []string{"value-2"}), 10.0)
	assert.Equal(t, 2, seriesUpdated)

	// Update both series
	g.Inc(buildTestLabels([]string{"label"}, []string{"value-1"}), 1.0)
	g.Set(buildTestLabels([]string{"label"}, []string{"value-2"}), 15.0)
	assert.Equal(t, 4, seriesUpdated)
}
