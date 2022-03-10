package registry

import (
	"math"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestTestRegistry_counter(t *testing.T) {
	testRegistry := NewTestRegistry()

	counter := testRegistry.NewCounter("counter")

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})

	counter.Inc(lbls, 1.0)
	counter.Inc(lbls, 2.0)
	counter.Inc(lbls, 1.5)

	assert.Equal(t, 4.5, testRegistry.Query("counter", lbls))
}

func TestTestRegistry_histogram(t *testing.T) {
	testRegistry := NewTestRegistry()

	histogram := testRegistry.NewHistogram("histogram", []float64{1.0, 2.0})

	lbls := labels.FromMap(map[string]string{
		"foo": "foo-value",
		"bar": "bar-value",
	})

	histogram.Observe(lbls, 1.0)
	histogram.Observe(lbls, 2.0)
	histogram.Observe(lbls, 2.5)

	assert.Equal(t, 1.0, testRegistry.Query("histogram_bucket", withLe(lbls, 1.0)))
	assert.Equal(t, 2.0, testRegistry.Query("histogram_bucket", withLe(lbls, 2.0)))
	assert.Equal(t, 3.0, testRegistry.Query("histogram_bucket", withLe(lbls, math.Inf(1))))
	assert.Equal(t, 3.0, testRegistry.Query("histogram_count", lbls))
	assert.Equal(t, 5.5, testRegistry.Query("histogram_sum", lbls))
}
