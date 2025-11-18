package registry

import (
	"sync"

	"github.com/prometheus/prometheus/model/labels"
)

var builderPool = sync.Pool{
	New: func() interface{} {
		return labels.NewBuilder(labels.New())
	},
}

type labelBuilder struct {
	builder             *labels.Builder
	maxLabelNameLength  int
	maxLabelValueLength int
}

var _ LabelBuilder = (*labelBuilder)(nil)

func NewLabelBuilder(maxLabelNameLength int, maxLabelValueLength int) LabelBuilder {
	builder := builderPool.Get().(*labels.Builder)
	builder.Reset(labels.New())
	return &labelBuilder{
		builder:             builder,
		maxLabelNameLength:  maxLabelNameLength,
		maxLabelValueLength: maxLabelValueLength,
	}
}

func (b *labelBuilder) Add(name, value string) {
	if b.maxLabelNameLength > 0 && len(name) > b.maxLabelNameLength {
		name = name[:b.maxLabelNameLength]
	}
	if b.maxLabelValueLength > 0 && len(value) > b.maxLabelValueLength {
		value = value[:b.maxLabelValueLength]
	}
	b.builder.Set(name, value)
}

func (b *labelBuilder) Labels() labels.Labels {
	labels := b.builder.Labels()
	// it's no longer safe to use the builder after this point, so we drop our
	// reference to it. this may cause a nil panic if the builder is used after
	// this point, but it's better than memory corruption.
	builderPool.Put(b.builder)
	b.builder = nil

	return labels
}
