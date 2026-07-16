package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// Hash must be computed on the post-sanitize/post-limit labels — the
	// *Borrowed metric methods trust it as the series key. A regression that
	// hashed the pre-transform labels would silently split or collide series.
	assert.Equal(t, borrowed.Labels.Hash(), borrowed.Hash, "Hash must match the transformed labels")
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

func TestBorrowedLabels_DoubleReleaseBeforeReuseIsNoOp(t *testing.T) {
	// Release must not double-Put the builder/scratch into their pools when
	// called twice through the same pointer. This is best-effort misuse
	// mitigation, not a guarantee: it only holds until the pooled builder is
	// re-issued by a future CloseAndBorrowLabels — the same *BorrowedLabels is
	// then live again, and a stale Release would clear the new borrower's
	// fields and still double-Put. Release is documented as call-exactly-once.
	// Release on a dereferenced copy of BorrowedLabels is likewise forbidden:
	// the copy retains independent non-nil builder/scratch pointers, so
	// releasing both double-Puts.
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

func TestBorrowedLabels_ReleaseOnFailedBorrowIsNoOp(t *testing.T) {
	// CloseAndBorrowLabels returns (nil, false) for an invalid label set.
	// Release on that nil must be a no-op so a caller that defers Release
	// before checking ok cannot be panicked by remote-controlled input (span
	// attributes with invalid UTF-8).
	builder := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
	builder.Add("name", "svc-\xc3\x28") // invalid UTF-8

	borrowed, ok := builder.CloseAndBorrowLabels()
	require.False(t, ok)
	require.Nil(t, borrowed)
	require.NotPanics(t, func() { borrowed.Release() })
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
	// Add every name once in reverse order, then again in reverse order with
	// the final values: duplicates are interleaved across the whole set, so
	// last-write-wins only holds if the fallback sort is stable.
	for i := insertionSortThreshold + 1; i >= 0; i-- {
		builder.Add(fmt.Sprintf("label_%03d", i), "first")
	}
	for i := insertionSortThreshold + 1; i >= 0; i-- {
		name := fmt.Sprintf("label_%03d", i)
		builder.Add(name, fmt.Sprintf("value_%03d", i))
		want[name] = fmt.Sprintf("value_%03d", i)
	}

	lbls, ok := builder.CloseAndBuildLabels()
	assert.True(t, ok)
	assert.Equal(t, labels.FromMap(want), lbls)
}

// TestBorrowedLabels_CollapsedLabelsProduceSingleSeries is the behavioral
// counterpart of the Hash==Labels.Hash() invariant: when the sanitizer collapses
// two different raw inputs to the same final label set, they must map to a
// single series. This only holds because BorrowedLabels.Hash is computed on the
// post-transform labels; a regression that hashed pre-transform labels would
// split them into two series (or collide unrelated ones).
func TestBorrowedLabels_CollapsedLabelsProduceSingleSeries(t *testing.T) {
	collapse := sanitizerFunc(func(_ labels.Labels) labels.Labels {
		return labels.FromStrings("span_name", "collapsed")
	})

	c := newCounter("my_counter", noopLimiter, nil, 15*time.Minute)

	for _, raw := range []string{"GET /a", "GET /b"} {
		builder := NewLabelBuilder(0, 0, collapse, newTestLabelLimiter())
		builder.Add("span_name", raw)
		borrowed, ok := builder.CloseAndBorrowLabels()
		require.True(t, ok)
		require.Equal(t, borrowed.Labels.Hash(), borrowed.Hash)
		c.IncBorrowed(borrowed, 1, time.Now().UnixMilli())
		borrowed.Release()
	}

	require.Equal(t, 1, c.countActiveSeries(), "labels that collapse to the same set must share one series")
}

// TestLabelBuilder_DoubleClosePanics verifies both close methods reject a second
// close on the same builder. Without the guard the second close would hand out
// a second borrow backed by the same pooled builder/scratch, and releasing both
// would double-Put them into the pools, handing one instance to two future
// callers.
func TestLabelBuilder_DoubleClosePanics(t *testing.T) {
	t.Run("CloseAndBuildLabels", func(t *testing.T) {
		b := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
		b.Add("name", "value")
		_, ok := b.CloseAndBuildLabels()
		require.True(t, ok)
		require.PanicsWithValue(t, "label builder used after Close", func() {
			_, _ = b.CloseAndBuildLabels()
		})
	})

	t.Run("CloseAndBorrowLabels", func(t *testing.T) {
		b := NewLabelBuilder(0, 0, newTestDrainSanitizer(SpanNameSanitizationDisabled), newTestLabelLimiter())
		b.Add("name", "value")
		borrowed, ok := b.CloseAndBorrowLabels()
		require.True(t, ok)
		defer borrowed.Release()
		require.PanicsWithValue(t, "label builder used after Close", func() {
			_, _ = b.CloseAndBorrowLabels()
		})
	})
}
