package registry

import "github.com/prometheus/prometheus/model/labels"

type LabelTransformer func(lbls labels.Labels) labels.Labels

type labelBuilder struct {
	builder     *labels.Builder
	transformer LabelTransformer

	maxLabelNameLength  int
	maxLabelValueLength int
}

var _ LabelBuilder = (*labelBuilder)(nil)

func NewLabelBuilder(maxLabelNameLength int, maxLabelValueLength int, transformer LabelTransformer) LabelBuilder {
	return labelBuilder{
		builder:             labels.NewBuilder(labels.New()),
		maxLabelNameLength:  maxLabelNameLength,
		maxLabelValueLength: maxLabelValueLength,
		transformer:         transformer,
	}
}

func (b labelBuilder) Add(name, value string) {
	if b.maxLabelNameLength > 0 && len(name) > b.maxLabelNameLength {
		name = name[:b.maxLabelNameLength]
	}
	if b.maxLabelValueLength > 0 && len(value) > b.maxLabelValueLength {
		value = value[:b.maxLabelValueLength]
	}
	b.builder.Set(name, value)
}

func (b labelBuilder) Labels() labels.Labels {
	return b.transformer(b.builder.Labels())
}
