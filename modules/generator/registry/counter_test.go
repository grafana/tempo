package registry

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func Test_counter(t *testing.T) {
	var seriesAdded int
	onAdd := func(_ uint32) bool {
		seriesAdded++
		return true
	}

	c := newCounter("my_counter", onAdd, nil, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	endOfLastMinuteMs = getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-3"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-3"}, collectionTimeMs, 3),
	}

	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 3, expectedSamples, nil)
}

func TestCounterDifferentLabels(t *testing.T) {
	var seriesAdded int
	onAdd := func(_ uint32) bool {
		seriesAdded++
		return true
	}

	c := newCounter("my_counter", onAdd, nil, nil)

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"another_label"}, []string{"another_value"}), 2.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "another_label": "another_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "another_label": "another_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_counter_cantAdd(t *testing.T) {
	canAdd := false
	onAdd := func(count uint32) bool {
		assert.Equal(t, uint32(1), count)
		return canAdd
	}

	c := newCounter("my_counter", onAdd, nil, nil)

	// allow adding new series
	canAdd = true

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	// block new series - existing series can still be updated
	canAdd = false

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_counter_removeStaleSeries(t *testing.T) {
	var removedSeries int
	onRemove := func(count uint32) {
		assert.Equal(t, uint32(1), count)
		removedSeries++
	}

	c := newCounter("my_counter", nil, onRemove, nil)

	timeMs := time.Now().UnixMilli()
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	assert.Equal(t, 0, removedSeries)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	// Expected is 4 because of when a sample is new we backdate it.
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)

	timeMs = time.Now().UnixMilli()
	time.Sleep(10 * time.Millisecond)
	// Anything after this will stay
	stalenessTimeMs := time.Now().UnixMilli()

	// update value-2 series
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)
	activeSeries, err := c.collectMetrics(newCapturingAppender(), timeMs, stalenessTimeMs)
	require.NoError(t, err)
	require.Equal(t, 1, activeSeries)
	require.Equal(t, 1, removedSeries)

	time.Sleep(1 * time.Second)
	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2"}, collectionTimeMs, 4),
	}

	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 1, expectedSamples, nil)
}

func Test_counter_externalLabels(t *testing.T) {
	c := newCounter("my_counter", nil, nil, map[string]string{"external_label": "external_value"})

	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
	c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.0)

	collectionTimeMs := time.Now().UnixMilli()
	endOfLastMinuteMs := getEndOfLastMinuteMs(collectionTimeMs)
	expectedSamples := []sample{
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-1", "external_label": "external_value"}, collectionTimeMs, 1),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "external_label": "external_value"}, endOfLastMinuteMs, 0),
		newSample(map[string]string{"__name__": "my_counter", "label": "value-2", "external_label": "external_value"}, collectionTimeMs, 2),
	}
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 2, expectedSamples, nil)
}

func Test_counter_concurrencyDataRace(t *testing.T) {
	c := newCounter("my_counter", nil, nil, nil)

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

func Test_counter_concurrencyCorrectness(t *testing.T) {
	c := newCounter("my_counter", nil, nil, nil)

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
					c.Inc(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0)
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
	collectMetricAndAssert(t, c, collectionTimeMs, dontCheckStaleness, 1, expectedSamples, nil)
}

func collectMetricAndAssert(t *testing.T, m metric, collectionTimeMs int64, stalenessTimeMs int64, expectedActiveSeries int, expectedSamples []sample, expectedExemplars []exemplarSample) {
	appender := newCapturingAppender()

	activeSeries, err := m.collectMetrics(appender, collectionTimeMs, stalenessTimeMs)
	assert.NoError(t, err)
	assert.Equal(t, expectedActiveSeries, activeSeries)

	assert.False(t, appender.isCommitted)
	assert.False(t, appender.isRolledback)
	go func() {
		_ = appender.Commit()
	}()
	samples := <-appender.onCommit
	assert.ElementsMatch(t, expectedSamples, samples)
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
