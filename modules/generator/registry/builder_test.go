package registry

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestLabelBuilder(t *testing.T) {
	builder := NewLabelBuilder(0, 0)
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
	builder := NewLabelBuilder(10, 10)
	builder.Add("name", "very_long_value")
	builder.Add("very_long_name", "value")

	lbls, ok := builder.CloseAndBuildLabels()

	assert.True(t, ok)
	assert.Equal(t, labels.FromStrings("name", "very_long_", "very_long_", "value"), lbls)
}

func TestLabelBuilder_InvalidUTF8(t *testing.T) {
	builder := NewLabelBuilder(0, 0)
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
