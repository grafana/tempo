package registry

import (
	"math"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

type testEntityLifecycler struct {
	onAddEntityFunc    func(entityHash uint64, count uint32) bool
	onUpdateEntityFunc func(entityHash uint64)
	onRemoveEntityFunc func(count uint32)
}

var noopLifecycler entityLifecycler = &testEntityLifecycler{}

var _ entityLifecycler = (*testEntityLifecycler)(nil)

func (t *testEntityLifecycler) onAddEntity(entityHash uint64, count uint32) bool {
	if t.onAddEntityFunc == nil {
		return true
	}
	return t.onAddEntityFunc(entityHash, count)
}

func (t *testEntityLifecycler) onUpdateEntity(entityHash uint64) {
	if t.onUpdateEntityFunc == nil {
		return
	}
	t.onUpdateEntityFunc(entityHash)
}

func (t *testEntityLifecycler) onRemoveEntity(count uint32) {
	if t.onRemoveEntityFunc == nil {
		return
	}
	t.onRemoveEntityFunc(count)
}

func removeStaleSeries(m metric, collectionTimeMs int64) {
	m.deleteFunc(func(hash uint64, lastUpdateMilli int64) bool {
		return lastUpdateMilli < collectionTimeMs
	})
}

func TestTestRegistry_counter(t *testing.T) {
	testRegistry := NewTestRegistry()

	counter := testRegistry.NewCounter("counter")

	labelValues := newLabelValueCombo([]string{"foo", "bar"}, []string{"foo-value", "bar-value"})
	counter.Inc(labelValues, 1.0)
	counter.Inc(labelValues, 2.0)
	counter.Inc(labelValues, 1.5)

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})
	assert.Equal(t, 4.5, testRegistry.Query("counter", lbls))
}

func TestTestRegistry_histogram(t *testing.T) {
	testRegistry := NewTestRegistry()

	histogram := testRegistry.NewHistogram("histogram", []float64{1.0, 2.0}, HistogramModeClassic)

	labelValues := newLabelValueCombo([]string{"foo", "bar"}, []string{"foo-value", "bar-value"})
	histogram.ObserveWithExemplar(labelValues, 1.0, "", 1.0)
	histogram.ObserveWithExemplar(labelValues, 2.0, "", 1.0)
	histogram.ObserveWithExemplar(labelValues, 2.5, "", 1.0)

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})
	assert.Equal(t, 1.0, testRegistry.Query("histogram_bucket", withLe(lbls, 1.0)))
	assert.Equal(t, 2.0, testRegistry.Query("histogram_bucket", withLe(lbls, 2.0)))
	assert.Equal(t, 3.0, testRegistry.Query("histogram_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query("histogram_count", lbls))
	assert.Equal(t, 5.5, testRegistry.Query("histogram_sum", lbls))
}
