package registry

import (
	"context"
	"fmt"

	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/storage"
)

type noopAppender struct{}

var _ storage.Appendable = (*noopAppender)(nil)
var _ storage.Appender = (*noopAppender)(nil)

func (n noopAppender) Appender(context.Context) storage.Appender { return n }

func (n noopAppender) Append(storage.SeriesRef, labels.Labels, int64, float64) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendExemplar(storage.SeriesRef, labels.Labels, exemplar.Exemplar) (storage.SeriesRef, error) {
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
	return sample{
		l: labels.FromMap(lbls),
		t: t,
		v: v,
	}
}

func newExemplar(lbls map[string]string, e exemplar.Exemplar) exemplarSample {
	return exemplarSample{
		l: labels.FromMap(lbls),
		e: e,
	}
}

func (s sample) String() string {
	return fmt.Sprintf("%s %d %g", s.l, s.t, s.v)
}

var _ storage.Appendable = (*capturingAppender)(nil)
var _ storage.Appender = (*capturingAppender)(nil)

func (c *capturingAppender) Appender(ctx context.Context) storage.Appender {
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
