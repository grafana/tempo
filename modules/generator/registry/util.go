package registry

import "github.com/prometheus/prometheus/model/labels"

// newSeriesLabelsBuilder creates a labels builder with user labels and external labels pre-populated.
func newSeriesLabelsBuilder(labelValueCombo *LabelValueCombo, externalLabels map[string]string) *labels.Builder {
	var builder *labels.Builder
	if labelValueCombo != nil {
		builder = labels.NewBuilder(labelValueCombo.labels)
	} else {
		builder = labels.NewBuilder(labels.New())
	}
	for name, value := range externalLabels {
		builder.Set(name, value)
	}
	return builder
}

// Returns the labels for the metric series including external labels
func getSeriesLabels(metricName string, labelValueCombo *LabelValueCombo, externalLabels map[string]string) labels.Labels {
	builder := newSeriesLabelsBuilder(labelValueCombo, externalLabels)
	builder.Set(labels.MetricName, metricName)
	return builder.Labels()
}
