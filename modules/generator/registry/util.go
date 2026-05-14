package registry

import (
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

// newSeriesLabelsBuilder creates a labels builder with user labels and external labels pre-populated.
func newSeriesLabelsBuilder(lbls labels.Labels, externalLabels map[string]string) *labels.Builder {
	builder := &labels.Builder{}
	builder.Reset(lbls)
	for name, value := range externalLabels {
		builder.Set(name, value)
	}
	return builder
}

// Returns the labels for the metric series including external labels
func getSeriesLabels(metricName string, lbls labels.Labels, externalLabels map[string]string) labels.Labels {
	if len(externalLabels) == 0 && canPrependMetricName(lbls) {
		scratch := scratchBuilderPool.Get().(*labels.ScratchBuilder)
		scratch.Reset()
		scratch.Add(model.MetricNameLabel, metricName)
		lbls.Range(func(l labels.Label) {
			scratch.Add(l.Name, l.Value)
		})
		seriesLabels := scratch.Labels()
		scratchBuilderPool.Put(scratch)
		return seriesLabels
	}

	var builder labels.Builder
	builder.Reset(lbls)
	for name, value := range externalLabels {
		builder.Set(name, value)
	}
	builder.Set(model.MetricNameLabel, metricName)
	return builder.Labels()
}

func canPrependMetricName(lbls labels.Labels) bool {
	canPrepend := true
	lbls.Range(func(l labels.Label) {
		if l.Name <= model.MetricNameLabel {
			canPrepend = false
		}
	})
	return canPrepend
}
