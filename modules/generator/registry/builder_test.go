package registry

import (
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

func TestSafeBuilderPool(t *testing.T) {
	pool := newSafeBuilderPool()
	builder := pool.Get()
	builder.Set("name", "value")
	lbls := builder.Labels()

	assert.Equal(t, labels.FromStrings("name", "value"), lbls)

	// Putting the builder back into the pool should reset it.
	pool.Put(builder)

	reusedBuilder := pool.Get()
	assert.Equal(t, builder, reusedBuilder)
	assert.Equal(t, labels.EmptyLabels(), reusedBuilder.Labels())
}

type sanitizerFunc func(lbls labels.Labels) labels.Labels

var _ Sanitizer = (*sanitizerFunc)(nil)

func (s sanitizerFunc) Sanitize(lbls labels.Labels) labels.Labels {
	return s(lbls)
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

func TestBorrowedLabels_ReleaseIsIdempotent(t *testing.T) {
	// Release must not double-Put the builder/scratch into their pools.
	// Calling it twice on the same value is a defensive guard against future
	// callers that accidentally release through both a pointer and a copy.
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
