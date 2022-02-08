package remotewrite

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	samplesSent       *prometheus.CounterVec
	exemplarsSent     *prometheus.CounterVec
	remoteWriteErrors *prometheus.CounterVec
	remoteWriteTotal  *prometheus.CounterVec
}

// NewMetrics creates a Metrics and registers all counters with the given prometheus.Registerer. To
// avoid registering metrics twice, this method should only be called once per Registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		samplesSent: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_samples_sent_total",
			Help:      "Number of samples sent",
		}, []string{"tenant"}),
		exemplarsSent: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_exemplars_sent_total",
			Help:      "Number of exemplars sent",
		}, []string{"tenant"}),
		remoteWriteErrors: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_errors",
			Help:      "Number of remote-write requests that failed due to error.",
		}, []string{"tenant"}),
		remoteWriteTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_remote_write_total",
			Help:      "Number of remote-write requests.",
		}, []string{"tenant"}),
	}
}
