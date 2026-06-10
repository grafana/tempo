package registry

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/prometheus/prometheus/model/labels"
)

type labelBuilder struct {
	sanitizer       Sanitizer
	perLabelLimiter LabelLimiter
	labels          []labels.Label

	maxLabelNameLength  int
	maxLabelValueLength int
	closed              bool
}

var _ LabelBuilder = (*labelBuilder)(nil)

var labelBuilderPool = sync.Pool{
	New: func() interface{} {
		return &labelBuilder{
			labels: make([]labels.Label, 0, 16),
		}
	},
}

var scratchBuilderPool = sync.Pool{
	New: func() interface{} {
		b := labels.NewScratchBuilder(16)
		return &b
	},
}

func NewLabelBuilder(maxLabelNameLength int, maxLabelValueLength int, sanitizer Sanitizer, perLabelLimiter LabelLimiter) LabelBuilder {
	b := labelBuilderPool.Get().(*labelBuilder)
	b.sanitizer = sanitizer
	b.perLabelLimiter = perLabelLimiter
	b.labels = b.labels[:0]
	b.maxLabelNameLength = maxLabelNameLength
	b.maxLabelValueLength = maxLabelValueLength
	b.closed = false
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
	if value == "" {
		// Empty value deletes any prior label with this name (mirrors the
		// prometheus.Builder.Set("x","") -> Del("x") semantics that callers
		// depend on for sanitization-collision handling).
		out := b.labels[:0]
		for _, l := range b.labels {
			if l.Name != name {
				out = append(out, l)
			}
		}
		b.labels = out
		return
	}
	b.labels = append(b.labels, labels.Label{Name: name, Value: value})
}

func (b *labelBuilder) CloseAndBuildLabels() (labels.Labels, bool) {
	b.closed = true
	scratch := scratchBuilderPool.Get().(*labels.ScratchBuilder)
	b.prepareScratch(scratch)
	lbls := scratch.Labels()
	scratchBuilderPool.Put(scratch)

	// We always run sanitizer first and then run per label limiter to ensure that
	// per label limits are always applied after sanitizer.
	// Pipeline: sanitize labels --> per-label cardinality limit --> entity/series limit
	lbls = b.sanitizer.Sanitize(lbls)
	lbls = b.perLabelLimiter.Limit(lbls)
	// it's no longer safe to use the builder after this point, so we drop our
	// reference to it. The closed flag is best-effort: it catches misuse against
	// the same builder reference, but once the builder is recycled to the pool
	// and re-issued, a stale reference would corrupt another caller's labels.
	b.sanitizer = nil
	b.perLabelLimiter = nil
	b.labels = b.labels[:0]
	b.maxLabelNameLength = 0
	b.maxLabelValueLength = 0
	labelBuilderPool.Put(b)

	if !validUTF8Labels(lbls) {
		return lbls, false
	}

	return lbls, true
}

func (b *labelBuilder) CloseAndBorrowLabels() (BorrowedLabels, bool) {
	b.closed = true
	scratch := scratchBuilderPool.Get().(*labels.ScratchBuilder)
	b.prepareScratch(scratch)
	var lbls labels.Labels
	scratch.Overwrite(&lbls)

	// We always run sanitizer first and then run per label limiter to ensure that
	// per label limits are always applied after sanitizer.
	// Pipeline: sanitize labels --> per-label cardinality limit --> entity/series limit
	lbls = b.sanitizer.Sanitize(lbls)
	lbls = b.perLabelLimiter.Limit(lbls)
	if !validUTF8Labels(lbls) {
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

func (b *labelBuilder) releaseBorrowedLabels(scratch *labels.ScratchBuilder) {
	// it's no longer safe to use the builder after this point, so we drop our
	// reference to it. this may cause a nil panic if the builder is used after
	// this point, but it's better than memory corruption.
	b.sanitizer = nil
	b.perLabelLimiter = nil
	b.labels = b.labels[:0]
	b.maxLabelNameLength = 0
	b.maxLabelValueLength = 0
	scratchBuilderPool.Put(scratch)
	labelBuilderPool.Put(b)
}

func (b *labelBuilder) prepareScratch(scratch *labels.ScratchBuilder) {
	scratch.Reset()
	sortLabels(b.labels)
	b.labels = compactLabels(b.labels)
	for _, l := range b.labels {
		scratch.Add(l.Name, l.Value)
	}
}

func validUTF8Labels(lbls labels.Labels) bool {
	valid := true
	lbls.Range(func(l labels.Label) {
		if !valid {
			return
		}
		valid = l.Name != "" && utf8.ValidString(l.Name) && utf8.ValidString(l.Value)
	})
	return valid
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
