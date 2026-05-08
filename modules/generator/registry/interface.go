package registry

import (
	"github.com/prometheus/prometheus/model/labels"
)

// Registry is a metrics store.
type Registry interface {
	NewLabelBuilder() LabelBuilder
	// NewInfoMetricLabelBuilder returns a LabelBuilder that skips the per-label
	// cardinality limiter and drain sanitizer.
	// Use this builder for info metrics (target_info, host_info) whose labels are high cardinality by design.
	NewInfoMetricLabelBuilder() LabelBuilder
	NewCounter(name string) Counter
	NewHistogram(name string, buckets []float64, histogramOverride HistogramMode) Histogram
	NewGauge(name string) Gauge
}

type LabelBuilder interface {
	Add(name, value string)
	CloseAndBuildLabels() (labels.Labels, bool)
	CloseAndBorrowLabels() (BorrowedLabels, bool)
}

// BorrowedLabels holds a label set whose backing memory is owned by a pooled
// scratch builder. The Labels and Hash are only valid between
// CloseAndBorrowLabels and Release. Callers MUST NOT retain Labels (or substrings
// of any Label.Value) past Release, store BorrowedLabels in a long-lived field,
// copy the value across goroutines, or copy the value at all and call Release
// on more than one copy — the underlying scratch and builder are returned to
// pools by Release, and a second Release on a copy double-Puts them and lets
// later callers receive the same instance concurrently. Metric methods that
// accept BorrowedLabels.Labels copy what they need synchronously before
// returning.
type BorrowedLabels struct {
	Labels labels.Labels
	Hash   uint64

	builder *labelBuilder
	scratch *labels.ScratchBuilder
}

// Release returns the underlying builder and scratch buffer to their pools.
// It is safe to call multiple times: subsequent calls are no-ops.
// After Release, Labels and Hash are no longer valid.
func (b *BorrowedLabels) Release() {
	if b.builder == nil {
		return
	}
	b.builder.releaseBorrowedLabels(b.scratch)
	b.builder = nil
	b.scratch = nil
	b.Labels = labels.EmptyLabels()
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	metric

	Inc(lbls labels.Labels, value float64)
	// IncWithHash is like Inc but uses the supplied hash as the series key.
	// Precondition: hash MUST equal lbls.Hash(). Supplying any other value
	// silently maps to a different series and corrupts metrics.
	IncWithHash(lbls labels.Labels, hash uint64, value float64)
	// IncWithHashAt is like IncWithHash but uses timeMs as the lastUpdated stamp.
	// Same hash precondition as IncWithHash.
	IncWithHashAt(lbls labels.Labels, hash uint64, value float64, timeMs int64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	metric

	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64)
	// ObserveWithExemplarWithHash is like ObserveWithExemplar but uses the supplied hash.
	// Precondition: hash MUST equal lbls.Hash().
	ObserveWithExemplarWithHash(lbls labels.Labels, hash uint64, value float64, traceID string, multiplier float64)
	// ObserveWithExemplarWithHashAt is like ObserveWithExemplarWithHash but uses timeMs as the lastUpdated stamp.
	// Precondition: hash MUST equal lbls.Hash(). traceID MUST be an owned string —
	// the histogram stores it without copying for the lifetime of the exemplar.
	ObserveWithExemplarWithHashAt(lbls labels.Labels, hash uint64, value float64, traceID string, multiplier float64, timeMs int64)
	// ObserveWithExemplarTraceIDBytesWithHashAt accepts the traceID as raw bytes.
	// The histogram copies the bytes synchronously, so the slice may be reused after the call.
	// Precondition: hash MUST equal lbls.Hash().
	ObserveWithExemplarTraceIDBytesWithHashAt(lbls labels.Labels, hash uint64, value float64, traceID []byte, multiplier float64, timeMs int64)
}

// Gauge
// https://prometheus.io/docs/concepts/metric_types/#gauge
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Gauge
type Gauge interface {
	metric

	// Set sets the Gauge to an arbitrary value.
	Set(lbls labels.Labels, value float64)
	Inc(lbls labels.Labels, value float64)
	SetForTargetInfo(lbls labels.Labels, value float64)
	// SetForTargetInfoWithHash is like SetForTargetInfo but uses the supplied hash.
	// Precondition: hash MUST equal lbls.Hash().
	SetForTargetInfoWithHash(lbls labels.Labels, hash uint64, value float64)
	// SetForTargetInfoWithHashAt is like SetForTargetInfoWithHash but uses timeMs as the lastUpdated stamp.
	// Precondition: hash MUST equal lbls.Hash().
	SetForTargetInfoWithHashAt(lbls labels.Labels, hash uint64, value float64, timeMs int64)
}

type HistogramMode int

const (
	HistogramModeClassic HistogramMode = iota
	HistogramModeNative
	HistogramModeBoth
)

var HistogramModeToString = map[HistogramMode]string{
	HistogramModeClassic: "classic",
	HistogramModeNative:  "native",
	HistogramModeBoth:    "both",
}

var HistogramModeToValue = map[string]HistogramMode{
	"classic": HistogramModeClassic,
	"native":  HistogramModeNative,
	"both":    HistogramModeBoth,
}
