package registry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TODO 😶

func Test_native_histogram(t *testing.T) {
	var seriesAdded int
	onAdd := func(count uint32) bool {
		seriesAdded++
		return true
	}

	h := newNativeHistogram("my_histogram", onAdd, nil, "trace_id")

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-1"}), 1.0, "trace-1", 1.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 1.5, "trace-2", 1.0)

	assert.Equal(t, 2, seriesAdded)

	collectionTimeMs := time.Now().UnixMilli()
	expectedSamples := []sample{}
	expectedExemplars := []exemplarSample{}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 10, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 1.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 1.0)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{}
	expectedExemplars = []exemplarSample{}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)

	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-2"}), 2.5, "trace-2.2", 20.0)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 3.0, "trace-3", 13.5)
	h.ObserveWithExemplar(newLabelValueCombo([]string{"label"}, []string{"value-3"}), 1.0, "trace-3", 7.5)

	assert.Equal(t, 3, seriesAdded)

	collectionTimeMs = time.Now().UnixMilli()
	expectedSamples = []sample{}
	expectedExemplars = []exemplarSample{}
	collectMetricAndAssert(t, h, collectionTimeMs, nil, 15, expectedSamples, expectedExemplars)
}
