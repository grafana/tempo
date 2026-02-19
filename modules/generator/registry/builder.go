package registry

import (
	"sync"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type safeBuilderPool struct {
	pool sync.Pool
}

func newSafeBuilderPool() *safeBuilderPool {
	return &safeBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return labels.NewBuilder(labels.New())
			},
		},
	}
}

func (p *safeBuilderPool) Get() *labels.Builder {
	return p.pool.Get().(*labels.Builder)
}

func (p *safeBuilderPool) Put(builder *labels.Builder) {
	builder.Reset(labels.New())
	p.pool.Put(builder)
}

var builderPool = newSafeBuilderPool()

type labelBuilder struct {
	builder         *labels.Builder
	sanitizer       Sanitizer
	perLabelLimiter LabelLimiter

	maxLabelNameLength  int
	maxLabelValueLength int
}

var _ LabelBuilder = (*labelBuilder)(nil)

func NewLabelBuilder(maxLabelNameLength int, maxLabelValueLength int, sanitizer Sanitizer, perLabelLimiter LabelLimiter) LabelBuilder {
	builder := builderPool.Get()
	return &labelBuilder{
		builder:             builder,
		sanitizer:           sanitizer,
		perLabelLimiter:     perLabelLimiter,
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

func (b *labelBuilder) CloseAndBuildLabels() (labels.Labels, bool) {
	// We always run sanitizer first and then run per label limiter to ensure that
	// per label limits are always applied after sanitizer.
	// Pipeline: sanitize labels --> per-label cardinality limit --> entity/series limit
	lbls := b.sanitizer.Sanitize(b.builder.Labels())
	lbls = b.perLabelLimiter.Limit(lbls)
	// it's no longer safe to use the builder after this point, so we drop our
	// reference to it. this may cause a nil panic if the builder is used after
	// this point, but it's better than memory corruption.
	builderPool.Put(b.builder)
	b.builder = nil

	if !lbls.IsValid(model.UTF8Validation) {
		return lbls, false
	}

	return lbls, true
}
