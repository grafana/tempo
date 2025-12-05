package registry

import "github.com/prometheus/prometheus/model/labels"

// newSeriesLabelsBuilder creates a labels builder with user labels and external labels pre-populated.
func newSeriesLabelsBuilder(lbls labels.Labels, externalLabels map[string]string) *labels.Builder {
	builder := labels.NewBuilder(lbls)
	for name, value := range externalLabels {
		builder.Set(name, value)
	}
	return builder
}

// Returns the labels for the metric series including external labels
func getSeriesLabels(metricName string, lbls labels.Labels, externalLabels map[string]string) labels.Labels {
	builder := newSeriesLabelsBuilder(lbls, externalLabels)
	builder.Set(labels.MetricName, metricName)
	return builder.Labels()
}
