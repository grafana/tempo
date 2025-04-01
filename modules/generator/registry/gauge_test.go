package registry

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Test_gaugeInc(t *testing.T) {
	var seriesAdded int
	onAdd := func(_ uint32) bool {
		seriesAdded++
		return true
	}

	c := newGauge("my_gauge", onAdd, nil, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-3"}, collectionTimeMs, 3),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 3, expectedSamples, nil)
}

func TestGaugeDifferentLabels(t *testing.T) {
	var seriesAdded int
	onAdd := func(_ uint32) bool {
		seriesAdded++
		return true
	}

	c := newGauge("my_gauge", onAdd, nil, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"another_label"}, []string{"another_value"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "another_label": "another_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_gaugeSet(t *testing.T) {
	var seriesAdded int
	onAdd := func(_ uint32) bool {
		seriesAdded++
		return true
	}

	c := newGauge("my_gauge", onAdd, nil, nil)

	c.Set(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Set(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	c.Set(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	c.Set(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-3"}, collectionTimeMs, 3),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 3, expectedSamples, nil)
}

func Test_gauge_cantAdd(t *testing.T) {
	canAdd := false
	onAdd := func(count uint32) bool {
		assert.Equal(t, uint32(1), count)
		return canAdd
	}

	c := newGauge("my_gauge", onAdd, nil, nil)

	// allow adding new series
	canAdd = true

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_gauge_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(1), count)
		removedSeries++
	}

	c := newGauge("my_gauge", nil, onRemove, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
	assert.Equal(t, 0, removedSeries)

	time.Sleep(10 * time.Millisecond)
	// By setting the staleness to now after the sleep we will exclude any created after this.
	stalenessTimeMs := time.Now().UnixMilli()

	// update value-2 series
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	activeSeries, err := c.collectMetrics(newCapturingAppender(), collectionTimeMs, stalenessTimeMs)
	assert.NoError(t, err)
	assert.Equal(t, 1, activeSeries)
	assert.Equal(t, 1, removedSeries)

	collectionTimeMs = time.Now().UnixMilli()
	time.Sleep(2 * time.Second)
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, stalenessTimeMs, 1, expectedSamples, nil)
}

func Test_gauge_externalLabels(t *testing.T) {
	c := newGauge("my_gauge", nil, nil, map[string]string{"external_label": "external_value"})

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_gauge", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_gauge_concurrencyDataRace(t *testing.T) {
	c := newGauge("my_gauge", nil, nil, nil)

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
			c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
			c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 1.0)
		})
	}

	// this goroutine constantly creates new series
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	go accessor(func() {
		s := make([]rune, 6)
		for i := range s {
			s[i] = letters[rand.Intn(len(letters))]
		}
		c.Inc(newLabelValueCombo([]string{"label"}, []string{string(s)}), 1.0)
	})

	go accessor(func() {
		_, err := c.collectMetrics(&noopAppender{}, 0, 0)
		assert.NoError(t, err)
	})

	time.Sleep(200 * time.Millisecond)
	close(end)
}

func Test_gauge_concurrencyCorrectness(t *testing.T) {
	c := newGauge("my_gauge", nil, nil, nil)

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
					c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
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
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 1, expectedSamples, nil)
}

func Test_gauge_sendStaleMarkers(t *testing.T) {
	c := newGauge("my_gauge", nil, nil, nil)

	appender := newCapturingAppender()
	collectionTimeMs := time.Now().UnixMilli()

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value"}), 1.0)
	// Need to get the first collection out of the way so that that the initial item is triggered
	// since it is always created. The 0 indicates no staleness checked.
	activeSeries, err := c.collectMetrics(appender, collectionTimeMs, 0)

	// Commit runs separately.
	go appender.Commit()

	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_gauge", "label": "value"}, collectionTimeMs, staleMarker()),
	}
	require.True(t, math.IsNaN(expectedSamples[0].v))

	time.Sleep(10 * time.Millisecond)
	// Anything before this will be marked as stale.
	stalenessTimeMs := time.Now().UnixMilli()
	appender = newCapturingAppender()
	activeSeries, err = c.collectMetrics(appender, collectionTimeMs, stalenessTimeMs)
	go appender.Commit()
	samples := <-appender.onCommit
	require.NoError(t, err)
	require.Equal(t, 0, activeSeries)
	require.True(t, appender.isCommitted)
	require.False(t, appender.isRolledback)
	require.Equal(t, expectedSamples[0].String(), samples[0].String())
	require.True(t, math.IsNaN(samples[0].v))
}

func BenchmarkGauge_100ConcurrentWriters(b *testing.B) {
	numWriters := 100
	writesPerGoroutine := 1000
	
	for i := 0; i < b.N; i++ {
		g := newGauge("benchmark_gauge", nil, nil, nil)
		
		// Create a wait group to coordinate goroutines
		var wg sync.WaitGroup
		wg.Add(numWriters)
		
		// Setup a start signal
		start := make(chan struct{})
		
		// Launch workers
		for w := 0; w < numWriters; w++ {
			worker := w
			go func() {
				defer wg.Done()
				
				// Create a unique label value for this worker
				labelValueCombo := newLabelValueCombo([]string{"worker"}, []string{fmt.Sprintf("worker-%d", worker)})
				
				// Wait for start signal
				<-start
				
				// Perform writes
				for j := 0; j < writesPerGoroutine; j++ {
					g.Inc(labelValueCombo, 1.0)
				}
			}()
		}
		
		// Start all workers simultaneously
		b.ResetTimer()
		close(start)
		
		// Wait for all workers to complete
		wg.Wait()
		
		// Measure collection performance
		appender := &noopAppender{}
		timeMs := time.Now().UnixMilli()
		_, err := g.collectMetrics(appender, timeMs, 0)
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		
		// Report number of series for context
		b.ReportMetric(float64(numWriters), "num_series")
		b.ReportMetric(float64(numWriters*writesPerGoroutine), "total_writes")
	}
}

func BenchmarkGauge_ConcurrentWriteAndCollect(b *testing.B) {
	numWriters := 100
	collectionDuration := 500 * time.Millisecond
	
	for i := 0; i < b.N; i++ {
		g := newGauge("benchmark_gauge", nil, nil, nil)
		
		// Channel to signal writers to stop
		done := make(chan struct{})
		
		// Launch concurrent writers
		for w := 0; w < numWriters; w++ {
			worker := w
			go func() {
				labelValueCombo := newLabelValueCombo([]string{"worker"}, []string{fmt.Sprintf("worker-%d", worker)})
				for {
					select {
					case <-done:
						return
					default:
						g.Inc(labelValueCombo, 1.0)
					}
				}
			}()
		}
		
		// Run benchmark for collecting metrics while writes are ongoing
		b.ResetTimer()
		appender := &noopAppender{}
		startTime := time.Now()
		endTime := startTime.Add(collectionDuration)
		
		var collections int
		for time.Now().Before(endTime) {
			timeMs := time.Now().UnixMilli()
			_, err := g.collectMetrics(appender, timeMs, 0)
			if err != nil {
				b.Fatal(err)
			}
			collections++
		}
		
		// Stop writers and cleanup
		close(done)
		b.StopTimer()
		
		// Report metrics
		b.ReportMetric(float64(collections), "collections")
		b.ReportMetric(float64(collections)/collectionDuration.Seconds(), "collections_per_second")
	}
}
