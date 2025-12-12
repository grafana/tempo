package drain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	TooFewTokens  = "too_few_tokens"
	TooManyTokens = "too_many_tokens"
)

type metrics struct {
	PatternsEvictedTotal  *prometheus.CounterVec
	PatternsExpiredTotal  *prometheus.CounterVec
	PatternsDetectedTotal *prometheus.CounterVec
	LinesSkipped          *prometheus.CounterVec
	TokensPerLine         *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	return &metrics{
		PatternsEvictedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_evicted_total",
			Help:      "The total amount of patterns evicted per tenant",
		}, []string{"tenant"}),
		PatternsExpiredTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_expired_total",
			Help:      "The total amount of patterns expired per tenant",
		}, []string{"tenant"}),
		PatternsDetectedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_detected_total",
			Help:      "The total amount of patterns detected per tenant",
		}, []string{"tenant"}),
		LinesSkipped: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_lines_skipped_total",
			Help:      "The total amount of lines skipped per tenant",
		}, []string{"reason", "tenant"}),
		TokensPerLine: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_tokens_per_line",
			Help:      "The number of tokens per line",
		}, []string{"tenant"}),
	}
}

type tenantMetrics struct {
	PatternsEvictedTotal      prometheus.Counter
	PatternsExpiredTotal      prometheus.Counter
	PatternsDetectedTotal     prometheus.Counter
	LinesSkippedTooManyTokens prometheus.Counter
	LinesSkippedTooFewTokens  prometheus.Counter
	TokensPerLine             prometheus.Observer
}

func (m *metrics) forTenant(tenant string) *tenantMetrics {
	return &tenantMetrics{
		PatternsEvictedTotal:      m.PatternsEvictedTotal.WithLabelValues(tenant),
		PatternsExpiredTotal:      m.PatternsExpiredTotal.WithLabelValues(tenant),
		PatternsDetectedTotal:     m.PatternsDetectedTotal.WithLabelValues(tenant),
		LinesSkippedTooManyTokens: m.LinesSkipped.WithLabelValues(TooManyTokens, tenant),
		LinesSkippedTooFewTokens:  m.LinesSkipped.WithLabelValues(TooFewTokens, tenant),
		TokensPerLine:             m.TokensPerLine.WithLabelValues(tenant),
	}
}

var globalMetrics = newMetrics(prometheus.DefaultRegisterer)

func metricsForTenant(tenant string) *tenantMetrics {
	return globalMetrics.forTenant(tenant)
}
