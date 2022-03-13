package registry

// Registry is a metrics store.
type Registry interface {
	NewCounter(name string, labels []string) Counter
	NewHistogram(name string, labels []string, buckets []float64) Histogram
}

// Counter
// https://prometheus.io/docs/concepts/metric_types/#counter
type Counter interface {
	Inc(values []string, value float64)
}

// Histogram
// https://prometheus.io/docs/concepts/metric_types/#histogram
type Histogram interface {
	Observe(values []string, value float64)
	// TODO support exemplars
	//ObserveWithExemplar(lbls labels.Labels, value float64, exemplarLbls labels.Labels)
}
