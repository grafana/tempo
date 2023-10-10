package registry

import (
	"context"
	"fmt"
	"sort"

	"github.com/prometheus/prometheus/model/exemplar"
	prom_histogram "github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/storage"
)

type noopAppender struct{}

var (
	_ storage.Appendable = (*noopAppender)(nil)
	_ storage.Appender   = (*noopAppender)(nil)
)

func (n noopAppender) Appender(context.Context) storage.Appender { return n }

func (n noopAppender) Append(storage.SeriesRef, labels.Labels, int64, float64) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(storage.SeriesRef, labels.Labels, exemplar.Exemplar) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendHistogram(storage.SeriesRef, labels.Labels, int64, *prom_histogram.Histogram, *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) Commit() error { return nil }

func (n noopAppender) Rollback() error { return nil }

func (n noopAppender) UpdateMetadata(storage.SeriesRef, labels.Labels, metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}

type capturingAppender struct {
	samples      []sample
	exemplars    []exemplarSample
	isCommitted  bool
	isRolledback bool
}

type sample struct {
	l labels.Labels
	t int64
	v float64
}

type exemplarSample struct {
	l labels.Labels
	e exemplar.Exemplar
}

func newSample(lbls map[string]string, t int64, v float64) sample {
	l := labels.FromMap(lbls)
	sort.Slice(l, func(i, j int) bool { return l[i].Name < l[j].Name })
	return sample{
		l: l,
		t: t,
		v: v,
	}
}

func newExemplar(lbls map[string]string, e exemplar.Exemplar) exemplarSample {
	l := labels.FromMap(lbls)
	sort.Slice(l, func(i, j int) bool { return l[i].Name < l[j].Name })
	return exemplarSample{
		l: l,
		e: e,
	}
}

func (s sample) String() string {
	return fmt.Sprintf("%s %d %g", s.l, s.t, s.v)
}

var (
	_ storage.Appendable = (*capturingAppender)(nil)
	_ storage.Appender   = (*capturingAppender)(nil)
)

func (c *capturingAppender) Appender(context.Context) storage.Appender {
	return c
}

func (c *capturingAppender) Append(ref storage.SeriesRef, l labels.Labels, t int64, v float64) (storage.SeriesRef, error) {
	c.samples = append(c.samples, sample{l, t, v})
	return ref, nil
}

func (c *capturingAppender) AppendExemplar(ref storage.SeriesRef, l labels.Labels, e exemplar.Exemplar) (storage.SeriesRef, error) {
	c.exemplars = append(c.exemplars, exemplarSample{l, e})
	return ref, nil
}

func (c *capturingAppender) AppendHistogram(ref storage.SeriesRef, l labels.Labels, e int64, h *prom_histogram.Histogram, fg *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return ref, nil
}

func (c *capturingAppender) Commit() error {
	c.isCommitted = true
	return nil
}

func (c *capturingAppender) Rollback() error {
	c.isRolledback = true
	return nil
}

func (c *capturingAppender) UpdateMetadata(storage.SeriesRef, labels.Labels, metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}
