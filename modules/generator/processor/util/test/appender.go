package test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/stretchr/testify/assert"
)

// Appender is a storage.Appender to be used in tests. It will store appended samples and has
// test functions to verify these are correct.
type Appender struct {
	IsCommitted, IsRolledback bool

	samples []Sample
}

type Metric struct {
	Labels string
	Value  float64
}

type Sample struct {
	l labels.Labels
	t int64
	v float64
}

var _ storage.Appender = (*Appender)(nil)

func (a *Appender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	a.samples = append(a.samples, Sample{l, t, v})
	return 0, nil
}

func (a *Appender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	panic("TODO add support for AppendExemplar")
}

func (a *Appender) Commit() error {
	a.IsCommitted = true
	return nil
}

func (a *Appender) Rollback() error {
	a.IsRolledback = true
	return nil
}

// Contains asserts that Appender contains expectedSample.
func (a *Appender) Contains(t *testing.T, expectedSample Metric) {
	assert.Greater(t, len(a.samples), 0)
	for _, sample := range a.samples {
		if expectedSample.Labels != sample.l.String() {
			continue
		}
		assert.Equal(t, expectedSample.Value, sample.v)
		return
	}

	t.Fatalf("could not find sample %v in Appender", expectedSample)
}

// NotContains asserts that Appender does not contain a sample with the given labels.
func (a *Appender) NotContains(t *testing.T, labels string) {
	for _, sample := range a.samples {
		if labels == sample.l.String() {
			t.Fatalf("appender contains sample %s", labels)
			return
		}
	}
}

// ContainsAll asserts that Appender contains all of expectedSamples in the given order.
// All samples should have a timestamp equal to timestamp with 1 millisecond of error margin.
func (a *Appender) ContainsAll(t *testing.T, expectedSamples []Metric, timestamp time.Time) {
	if len(expectedSamples) > 0 {
		assert.NotEmpty(t, a.samples)
	}
	sort.Slice(expectedSamples, func(i, j int) bool {
		return expectedSamples[i].Labels < expectedSamples[j].Labels
	})
	sort.Slice(a.samples, func(i, j int) bool {
		return a.samples[i].l.String() < a.samples[j].l.String()
	})

	for i, sample := range a.samples {
		labelsEqual := assert.Equal(t, expectedSamples[i].Labels, sample.l.String())
		if !labelsEqual {
			// This will happen if a time series is missing or incorrect, instead of printing a wall
			// of failed asserts as we continue iterating through the list, just dump the contents.
			fmt.Println("Test appender contains the following metrics")
			for i := range a.samples {
				fmt.Printf("%s %g\n", a.samples[i].l.String(), a.samples[i].v)
			}
			return
		}

		assert.InDelta(t, timestamp.UnixMilli(), sample.t, 1, sample.l)
		assert.Equal(t, expectedSamples[i].Value, sample.v, sample.l)
	}
}
