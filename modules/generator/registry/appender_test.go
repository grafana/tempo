package registry

import (
	"context"
	"fmt"

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

func (n noopAppender) SetOptions(_ *storage.AppendOptions) {}

func (n noopAppender) UpdateMetadata(storage.SeriesRef, labels.Labels, metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

func (n noopAppender) AppendHistogramSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

type capturingAppender struct {
	samples      []sample
	histograms   []histogramSample
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

type histogramSample struct {
	l labels.Labels
	t int64
	h *prom_histogram.Histogram
}

func newSample(lbls map[string]string, t int64, v float64) sample {
	l := labels.FromMap(lbls)
	return sample{
		l: l,
		t: t,
		v: v,
	}
}

func newExemplar(lbls map[string]string, e exemplar.Exemplar) exemplarSample {
	l := labels.FromMap(lbls)
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

func (c *capturingAppender) AppendHistogram(ref storage.SeriesRef, l labels.Labels, t int64, h *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	if h != nil {
		c.histograms = append(c.histograms, histogramSample{l, t, h.Copy()})
	}
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
func (c *capturingAppender) SetOptions(_ *storage.AppendOptions) {}
func (c *capturingAppender) UpdateMetadata(storage.SeriesRef, labels.Labels, metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}

func (c *capturingAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (c *capturingAppender) AppendSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (c *capturingAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

func (c *capturingAppender) AppendHistogramSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

// refIssuingAppender is an Appender that records the input SeriesRef of every
// Append call and returns a fixed non-zero ref. Used to test that metrics
// cache the issued ref and pass it back on subsequent collect cycles, which
// capturingAppender cannot observe (it echoes whatever ref it receives).
type refIssuingAppender struct {
	nextRef   storage.SeriesRef
	inputRefs []storage.SeriesRef
}

var (
	_ storage.Appendable = (*refIssuingAppender)(nil)
	_ storage.Appender   = (*refIssuingAppender)(nil)
)

func (r *refIssuingAppender) Appender(context.Context) storage.Appender { return r }
func (r *refIssuingAppender) Append(ref storage.SeriesRef, _ labels.Labels, _ int64, _ float64) (storage.SeriesRef, error) {
	r.inputRefs = append(r.inputRefs, ref)
	return r.nextRef, nil
}

func (r *refIssuingAppender) AppendExemplar(ref storage.SeriesRef, _ labels.Labels, _ exemplar.Exemplar) (storage.SeriesRef, error) {
	return ref, nil
}

func (r *refIssuingAppender) AppendHistogram(ref storage.SeriesRef, _ labels.Labels, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return ref, nil
}
func (r *refIssuingAppender) Commit() error                       { return nil }
func (r *refIssuingAppender) Rollback() error                     { return nil }
func (r *refIssuingAppender) SetOptions(_ *storage.AppendOptions) {}
func (r *refIssuingAppender) UpdateMetadata(_ storage.SeriesRef, _ labels.Labels, _ metadata.Metadata) (storage.SeriesRef, error) {
	return 0, nil
}

func (r *refIssuingAppender) AppendCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (r *refIssuingAppender) AppendSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64) (storage.SeriesRef, error) {
	return 0, nil
}

func (r *refIssuingAppender) AppendHistogramCTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}

func (r *refIssuingAppender) AppendHistogramSTZeroSample(_ storage.SeriesRef, _ labels.Labels, _, _ int64, _ *prom_histogram.Histogram, _ *prom_histogram.FloatHistogram) (storage.SeriesRef, error) {
	return 0, nil
}
