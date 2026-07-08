package registry

import (
	"slices"
	"strings"
	"sync"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type labelBuilder struct {
	sanitizer       Sanitizer
	perLabelLimiter LabelLimiter
	labels          []labels.Label
	pools           *labelBuilderPools

	maxLabelNameLength  int
	maxLabelValueLength int
	closed              bool
}

var _ LabelBuilder = (*labelBuilder)(nil)

// labelBuilderPools holds the object pools used while building a label set.
// Each ManagedRegistry owns one set (see registry.New), so pooled scratch
// memory is never shared across tenants: a retention bug then degrades from a
// cross-tenant disclosure to same-tenant corruption.
type labelBuilderPools struct {
	labelBuilderPool   sync.Pool
	scratchBuilderPool sync.Pool
}

func newLabelBuilderPools() *labelBuilderPools {
	p := &labelBuilderPools{}
	p.labelBuilderPool.New = func() interface{} {
		return &labelBuilder{
			labels: make([]labels.Label, 0, 16),
		}
	}
	p.scratchBuilderPool.New = func() interface{} {
		b := labels.NewScratchBuilder(16)
		return &b
	}
	return p
}

// defaultLabelBuilderPools backs the exported NewLabelBuilder, which is used by
// tests. Production code builds labels via (*ManagedRegistry).NewLabelBuilder /
// NewInfoMetricLabelBuilder, which use the registry's own per-tenant pools.
var defaultLabelBuilderPools = newLabelBuilderPools()

// NewLabelBuilder returns a LabelBuilder backed by the package-default pools.
// Intended for tests; production code uses the per-tenant pools owned by a
// ManagedRegistry.
func NewLabelBuilder(maxLabelNameLength int, maxLabelValueLength int, sanitizer Sanitizer, perLabelLimiter LabelLimiter) LabelBuilder {
	return defaultLabelBuilderPools.newLabelBuilder(maxLabelNameLength, maxLabelValueLength, sanitizer, perLabelLimiter)
}

func (p *labelBuilderPools) newLabelBuilder(maxLabelNameLength int, maxLabelValueLength int, sanitizer Sanitizer, perLabelLimiter LabelLimiter) LabelBuilder {
	b := p.labelBuilderPool.Get().(*labelBuilder)
	b.sanitizer = sanitizer
	b.perLabelLimiter = perLabelLimiter
	b.labels = b.labels[:0]
	b.maxLabelNameLength = maxLabelNameLength
	b.maxLabelValueLength = maxLabelValueLength
	b.closed = false
	b.pools = p
	return b
}

func (b *labelBuilder) Add(name, value string) {
	if b.closed {
		panic("label builder used after Close")
	}
	if b.maxLabelNameLength > 0 && len(name) > b.maxLabelNameLength {
		name = name[:b.maxLabelNameLength]
	}
	if b.maxLabelValueLength > 0 && len(value) > b.maxLabelValueLength {
		value = value[:b.maxLabelValueLength]
	}
	// Empty values are appended as-is and dropped by compactLabels when the
	// set is built: last-write-wins dedupe means an empty value deletes any
	// prior label with this name (mirrors the prometheus.Builder.Set("x","")
	// -> Del("x") semantics that callers depend on for sanitization-collision
	// handling).
	b.labels = append(b.labels, labels.Label{Name: name, Value: value})
}

func (b *labelBuilder) CloseAndBuildLabels() (labels.Labels, bool) {
	if b.closed {
		// This method pools the builder before returning, so a second call would
		// double-Put it (handing one builder to two future callers concurrently)
		// and operate on a builder another caller may already hold. Mirror Add's
		// use-after-close guard.
		panic("label builder used after Close")
	}
	b.closed = true
	scratch := b.pools.scratchBuilderPool.Get().(*labels.ScratchBuilder)
	b.prepareScratch(scratch)
	lbls := scratch.Labels()

	// We always run sanitizer first and then run per label limiter to ensure that
	// per label limits are always applied after sanitizer.
	// Pipeline: sanitize labels --> per-label cardinality limit --> entity/series limit
	lbls = b.sanitizer.Sanitize(lbls)
	lbls = b.perLabelLimiter.Limit(lbls)
	// scratch.Labels() copied the label data out of the scratch, so both the
	// builder and the scratch can be pooled before returning.
	b.releaseBorrowedLabels(scratch)

	return lbls, lbls.IsValid(model.UTF8Validation)
}

func (b *labelBuilder) CloseAndBorrowLabels() (BorrowedLabels, bool) {
	if b.closed {
		// Closing twice would borrow a second value backed by this same builder
		// and scratch, and releasing both double-Puts them into the pools
		// (handing one to two future callers concurrently). Mirror Add's guard.
		panic("label builder used after Close")
	}
	b.closed = true
	scratch := b.pools.scratchBuilderPool.Get().(*labels.ScratchBuilder)
	b.prepareScratch(scratch)
	var lbls labels.Labels
	scratch.Overwrite(&lbls)

	// We always run sanitizer first and then run per label limiter to ensure that
	// per label limits are always applied after sanitizer.
	// Pipeline: sanitize labels --> per-label cardinality limit --> entity/series limit
	lbls = b.sanitizer.Sanitize(lbls)
	lbls = b.perLabelLimiter.Limit(lbls)
	if !lbls.IsValid(model.UTF8Validation) {
		b.releaseBorrowedLabels(scratch)
		return BorrowedLabels{}, false
	}

	return BorrowedLabels{
		Labels:  lbls,
		Hash:    lbls.Hash(),
		builder: b,
		scratch: scratch,
	}, true
}

// releaseBorrowedLabels resets the builder and returns it and the scratch
// buffer to their pools. It is no longer safe to use the builder after this
// point, so we drop our references: a stale use nil-panics rather than
// corrupting memory. The closed flag is best-effort — it catches misuse
// against the same builder reference, but once the builder is recycled to the
// pool and re-issued, a stale reference would corrupt another caller's labels.
func (b *labelBuilder) releaseBorrowedLabels(scratch *labels.ScratchBuilder) {
	b.sanitizer = nil
	b.perLabelLimiter = nil
	b.labels = b.labels[:0]
	b.maxLabelNameLength = 0
	b.maxLabelValueLength = 0
	pools := b.pools
	pools.scratchBuilderPool.Put(scratch)
	pools.labelBuilderPool.Put(b)
}

func (b *labelBuilder) prepareScratch(scratch *labels.ScratchBuilder) {
	scratch.Reset()
	sortLabels(b.labels)
	b.labels = compactLabels(b.labels)
	for _, l := range b.labels {
		scratch.Add(l.Name, l.Value)
	}
}

// insertionSortThreshold is the label count above which sortLabels falls back
// to the standard library sort. Span metric label sets are almost always small
// and nearly sorted, where insertion sort is faster and allocation-free, but it
// is O(n²) so large label sets (e.g. target_info on resources with many
// attributes) use O(n log n) instead. Both paths are stable: compactLabels
// relies on the last-added duplicate name winning.
const insertionSortThreshold = 16

func sortLabels(lbls []labels.Label) {
	if len(lbls) > insertionSortThreshold {
		slices.SortStableFunc(lbls, func(a, b labels.Label) int {
			return strings.Compare(a.Name, b.Name)
		})
		return
	}
	for i := 1; i < len(lbls); i++ {
		for j := i; j > 0 && strings.Compare(lbls[j-1].Name, lbls[j].Name) > 0; j-- {
			lbls[j-1], lbls[j] = lbls[j], lbls[j-1]
		}
	}
}

// compactLabels owns duplicate-name and empty-value handling for sorted label
// slices: the last-added entry per name wins, and an empty value deletes the
// name entirely (Add appends empty values as-is). The common case — no
// duplicates, no empty values — returns the slice unchanged.
func compactLabels(lbls []labels.Label) []labels.Label {
	if len(lbls) == 0 {
		return lbls
	}

	for i, l := range lbls {
		if l.Value == "" || (i > 0 && l.Name == lbls[i-1].Name) {
			return compactLabelsSlow(lbls)
		}
	}
	return lbls
}

func compactLabelsSlow(lbls []labels.Label) []labels.Label {
	write := 0
	for read := 0; read < len(lbls); {
		next := read + 1
		for next < len(lbls) && lbls[next].Name == lbls[read].Name {
			next++
		}

		last := lbls[next-1]
		if last.Value != "" {
			lbls[write] = last
			write++
		}
		read = next
	}
	return lbls[:write]
}
