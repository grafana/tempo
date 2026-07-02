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
// of any Label.Name or Label.Value — in the default stringlabels build both
// alias the pooled buffer) past Release, store BorrowedLabels in a long-lived field,
// copy the value across goroutines, or copy the value at all and call Release
// on more than one copy — the underlying scratch and builder are returned to
// pools by Release, and a second Release on a copy double-Puts them and lets
// later callers receive the same instance concurrently. Metric methods that
// accept BorrowedLabels.Labels copy what they need synchronously before
// returning.
type BorrowedLabels struct {
	// noCopy makes `go vet` copylocks flag accidental copies of BorrowedLabels,
	// which are unsafe: a copy retains independent non-nil builder/scratch
	// pointers, so releasing both copies double-Puts them into the pools (see
	// the Release/copy warning above).
	noCopy noCopy

	Labels labels.Labels
	Hash   uint64

	builder *labelBuilder
	scratch *labels.ScratchBuilder
}

// noCopy may be added as a (named, non-embedded) field to structs which must not
// be copied after first use. It has no fields and zero runtime cost; its only
// purpose is the Lock/Unlock pair, which makes `go vet`'s copylocks analyzer
// report copies. See https://golang.org/issues/8005.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

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
	b.Hash = 0
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	metric

	Inc(lbls labels.Labels, value float64)
	// IncWithHashAt is like Inc but uses the supplied hash as the series key
	// and timeMs as the lastUpdated stamp. Precondition: hash MUST equal
	// lbls.Hash(). Supplying any other value silently maps to a different
	// series and corrupts metrics.
	IncWithHashAt(lbls labels.Labels, hash uint64, value float64, timeMs int64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	metric

	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64)
	// ObserveWithExemplarTraceIDBytesWithHashAt is like ObserveWithExemplar but
	// accepts the traceID as raw bytes, uses the supplied hash as the series key,
	// and uses timeMs as the lastUpdated stamp. The histogram copies the traceID
	// bytes synchronously, so the slice may be reused after the call.
	// Precondition: hash MUST equal lbls.Hash(). Supplying any other value
	// silently maps to a different series and corrupts metrics.
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
	// SetForTargetInfoWithHashAt is like SetForTargetInfo but uses the supplied
	// hash as the series key and timeMs as the lastUpdated stamp.
	// Precondition: hash MUST equal lbls.Hash(). Supplying any other value
	// silently maps to a different series and corrupts metrics.
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
