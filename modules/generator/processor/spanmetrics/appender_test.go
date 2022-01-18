package spanmetrics

import (
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
)

// testAppender is a storage.Appender to be used in tests. It will store appended samples and has
// test functions to verify these are correct.
type testAppender struct {
	isCommitted, isRolledback bool

	samples []testSample
}

type testMetric struct {
	labels string
	value  float64
}

type testSample struct {
	l labels.Labels
	t int64
	v float64
}

var _ storage.Appender = (*testAppender)(nil)

func (a *testAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.samples = append(a.samples, testSample{l, t, v})
	return 0, nil
}

func (a *testAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	panic("TODO add support for AppendExemplar")
}

func (a *testAppender) Commit() error {
	a.isCommitted = true
	return nil
}

func (a *testAppender) Rollback() error {
	a.isRolledback = true
	return nil
}

// Contains asserts that testAppender contains expectedSample.
func (a *testAppender) Contains(t *testing.T, expectedSample testMetric) {
	assert.Greater(t, len(a.samples), 0)
	for _, sample := range a.samples {
		if expectedSample.labels != sample.l.String() {
			continue
		}
		assert.Equal(t, expectedSample.value, sample.v)
		return
	}

	t.Fatalf("could not find sample %v in testAppender", expectedSample)
}

// NotContains asserts that testAppender does not contain a sample with the given labels.
func (a *testAppender) NotContains(t *testing.T, labels string) {
	for _, sample := range a.samples {
		if labels == sample.l.String() {
			t.Fatalf("appender contains sample %s", labels)
			return
		}
	}
}

// ContainsAll asserts that testAppender contains all of expectedSamples in the given order.
// All samples should have a timestamp equal to timestamp with 1 millisecond of error margin.
func (a *testAppender) ContainsAll(t *testing.T, expectedSamples []testMetric, timestamp time.Time) {
	if len(expectedSamples) > 0 {
		assert.NotEmpty(t, a.samples)
	}
	for i, sample := range a.samples {
		assert.Equal(t, expectedSamples[i].labels, sample.l.String())
		assert.InDelta(t, timestamp.UnixMilli(), sample.t, 1)
		assert.Equal(t, expectedSamples[i].value, sample.v)
	}
}
