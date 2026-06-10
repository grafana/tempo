package registry

import (
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestLabelBuilder(t *testing.T) {
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("name", "value")
	lbls, ok := builder.CloseAndBuildLabels()

	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("name", "value"), lbls)

	// Using the builder after calling Labels() will panic to prevent memory
	// corruption.
	assert.Panics(t, func() {
		builder.Add("test", "test")
	})
}

func TestLabelBuilder_MaxLabelNameLength(t *testing.T) {
	builder := NewLabelBuilder(10, 10, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("name", "very_long_value")
	builder.Add("very_long_name", "value")

	lbls, ok := builder.CloseAndBuildLabels()

	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("name", "very_long_", "very_long_", "value"), lbls)
}

func TestLabelBuilder_InvalidUTF8(t *testing.T) {
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("name", "svc-\xc3\x28") // Invalid UTF-8

	_, ok := builder.CloseAndBuildLabels()

	assert.False(t, ok)
}

type sanitizerFunc func(lbls labels.Labels) labels.Labels

var _ Sanitizer = (*sanitizerFunc)(nil)

func (s sanitizerFunc) Sanitize(lbls labels.Labels) labels.Labels {
	return s(lbls)
}

func TestLabelBuilder_DuplicateNamesLastWriteWins(t *testing.T) {
	// Two non-empty Adds with the same name must yield the last value, matching
	// prometheus.Builder.Set semantics. compactLabels' fast path detects the
	// duplicate and routes to compactLabelsSlow, which keeps the last entry of
	// each contiguous group of equal names after stable sorting.
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("dup", "first")
	builder.Add("other", "v")
	builder.Add("dup", "second")

	lbls, ok := builder.CloseAndBuildLabels()
	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("dup", "second", "other", "v"), lbls)
}

func TestLabelBuilder_AddEmptyValueRemovesAllMatches(t *testing.T) {
	// Add(name, "") must mirror prometheus.Builder.Set("x","")->Del("x"), which
	// removes every prior label with that name. Sanitization can collapse
	// distinct input keys (foo.bar, foo_bar, foo-bar) onto the same name, and
	// removing only the first match would leave a stale earlier value.
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("dup", "first")
	builder.Add("other", "v")
	builder.Add("dup", "second")
	builder.Add("dup", "")

	lbls, ok := builder.CloseAndBuildLabels()
	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("other", "v"), lbls)
}

func TestBorrowedLabels_AppliesSanitizerAndLimiter(t *testing.T) {
	// CloseAndBorrowLabels runs sanitizer then per-label limiter, like
	// CloseAndBuildLabels. Verify the same transformations apply through the
	// borrow path so callers can use either entry point interchangeably.
	builder := NewLabelBuilder(0, 0, sanitizerFunc(func(_ labels.Labels) labels.Labels {
		return labels.FromStrings("name", "sanitized")
	}), newTestLabelLimiter())
	builder.Add("name", "raw")

	borrowed, ok := builder.CloseAndBorrowLabels()
	assert.True(t, ok)
	assert.Equal(t, "sanitized", borrowed.Labels.Get("name"))
	borrowed.Release()
}

func TestBorrowedLabels_SequentialReuseDoesNotLeak(t *testing.T) {
	// The scratch pool reissues the same buffer to the next caller after
	// Release. Verify that a second borrow does not see labels from the first.
	for range 5 {
		builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
		builder.Add("first", "a")
		first, ok := builder.CloseAndBorrowLabels()
		assert.True(t, ok)
		assert.Equal(t, "a", first.Labels.Get("first"))
		first.Release()

		builder = NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
		builder.Add("second", "b")
		second, ok := builder.CloseAndBorrowLabels()
		assert.True(t, ok)
		assert.Equal(t, "", second.Labels.Get("first"))
		assert.Equal(t, "b", second.Labels.Get("second"))
		second.Release()
	}
}

func TestBorrowedLabels_ReleaseIsIdempotent(t *testing.T) {
	// Release must not double-Put the builder/scratch into their pools when
	// called more than once on the same struct value. This only protects
	// repeated Release on the same value — Release on a copy of BorrowedLabels
	// is documented as forbidden because each copy retains independent
	// non-nil builder/scratch pointers and the second Release on the copy
	// would still double-Put.
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("name", "value")

	borrowed, ok := builder.CloseAndBorrowLabels()
	assert.True(t, ok)
	assert.Equal(t, "value", borrowed.Labels.Get("name"))

	borrowed.Release()
	// second release is a no-op; would otherwise corrupt the sync.Pool.
	borrowed.Release()
	assert.Equal(t, labels.EmptyLabels(), borrowed.Labels)
}

func TestLabelBuilder_Sanitizer(t *testing.T) {
	builder := NewLabelBuilder(0, 0, sanitizerFunc(func(_ labels.Labels) labels.Labels {
		return labels.FromStrings("name", "sanitized_value")
	}), newTestLabelLimiter())
	builder.Add("name", "value")
	lbls, ok := builder.CloseAndBuildLabels()

	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("name", "sanitized_value"), lbls)
}

func TestLabelBuilder_LargeLabelSetSortsAndDeduplicates(t *testing.T) {
	// More than insertionSortThreshold labels routes sortLabels to the stdlib
	// stable sort. Last-write-wins for duplicates must hold on that path too.
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())

	want := make(map[string]string, insertionSortThreshold+2)
	// Add in reverse order to force real sorting work.
	for i := insertionSortThreshold + 1; i >= 0; i-- {
		name := fmt.Sprintf("label_%03d", i)
		builder.Add(name, "first")
		builder.Add(name, fmt.Sprintf("value_%03d", i))
		want[name] = fmt.Sprintf("value_%03d", i)
	}

	lbls, ok := builder.CloseAndBuildLabels()
	assert.True(t, ok)
	assert.Equal(t, labels.FromMap(want), lbls)
}
