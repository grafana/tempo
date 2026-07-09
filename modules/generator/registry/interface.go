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
	CloseAndBorrowLabels() (*BorrowedLabels, bool)
}

// BorrowedLabels holds a label set whose backing memory is owned by a pooled
// scratch builder, together with the precomputed Hash of Labels. Hash is
// computed by CloseAndBorrowLabels, so the pair cannot disagree; the metric
// *Borrowed methods trust it as the series key. The pointer returned by
// CloseAndBorrowLabels points into the pooled builder itself, so borrowing
// allocates nothing.
//
// Labels and Hash are only valid between CloseAndBorrowLabels and Release.
// Callers MUST NOT retain the pointer or Labels (or substrings of any
// Label.Name or Label.Value — in the default stringlabels build both alias
// the pooled buffer) past Release, store them in a long-lived field, or share
// them across goroutines — Release returns the underlying scratch and builder
// to pools, and the same BorrowedLabels is re-issued to future borrowers.
// Metric methods that accept *BorrowedLabels copy what they need synchronously
// before returning.
//
// Tests with owned labels may hand-construct a BorrowedLabels, setting Hash to
// Labels.Hash(); Release on such a value is a no-op.
type BorrowedLabels struct {
	// noCopy makes `go vet` copylocks flag accidental copies of the
	// dereferenced value, which are unsafe: a copy retains independent
	// non-nil builder/scratch pointers, so releasing both the original and
	// the copy double-Puts them into the pools (handing one to two future
	// callers concurrently).
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
// After Release, Labels and Hash are no longer valid. The fields are cleared
// before the builder is pooled: once it is Put, another goroutine may Get it,
// so writing to them afterwards would race with the next borrower.
//
// Treat Release as call-exactly-once per successful CloseAndBorrowLabels.
// Calling it again through the same pointer is a no-op only until the pooled
// builder is re-issued to a future borrower; after that the same
// *BorrowedLabels is live again, and a stale Release would clear the new
// borrower's fields and double-Put the builder and scratch into their pools.
// The no-op softens misuse — it does not license early or repeated Release.
//
// Release on a nil *BorrowedLabels is a no-op, so the nil returned by a
// failed CloseAndBorrowLabels is safe to Release (e.g. via a defer set up
// before checking ok).
func (b *BorrowedLabels) Release() {
	if b == nil {
		// CloseAndBorrowLabels returns nil when the label set is invalid.
		return
	}
	builder := b.builder
	if builder == nil {
		return
	}
	scratch := b.scratch
	b.builder = nil
	b.scratch = nil
	b.Labels = labels.EmptyLabels()
	b.Hash = 0
	builder.releaseBorrowedLabels(scratch)
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	metric

	Inc(lbls labels.Labels, value float64)
	// IncBorrowed is like Inc but keys the series by the precomputed
	// lbls.Hash and uses timeMs as the lastUpdated stamp. It copies what it
	// needs before returning, so the caller may Release lbls afterwards.
	IncBorrowed(lbls *BorrowedLabels, value float64, timeMs int64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	metric

	// ObserveWithExemplar observes a datapoint with the given values. traceID will be added as exemplar.
	ObserveWithExemplar(lbls labels.Labels, value float64, traceID string, multiplier float64)
	// ObserveBorrowed is like ObserveWithExemplar but keys the series by the
	// precomputed lbls.Hash, uses timeMs as the lastUpdated stamp, and takes
	// the exemplar trace ID as raw bytes (empty = no exemplar). It copies the
	// trace ID bytes and label data it needs before returning, so the caller
	// may Release lbls and reuse the slice afterwards.
	ObserveBorrowed(lbls *BorrowedLabels, value float64, traceID []byte, multiplier float64, timeMs int64)
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
	// SetForTargetInfoBorrowed is like SetForTargetInfo but keys the series by
	// the precomputed lbls.Hash and uses timeMs as the lastUpdated stamp. It
	// copies what it needs before returning, so the caller may Release lbls
	// afterwards.
	SetForTargetInfoBorrowed(lbls *BorrowedLabels, value float64, timeMs int64)
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
