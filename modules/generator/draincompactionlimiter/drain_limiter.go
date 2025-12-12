package draincompactionlimiter

import (
	"sync"
	"time"

	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/pkg/drain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
)

type drainLimiterMetrics struct {
	totalSpansCompacted   *prometheus.CounterVec
	patternsEvictedTotal  *prometheus.CounterVec
	patternsPrunedTotal   *prometheus.CounterVec
	patternsDetectedTotal *prometheus.CounterVec
	linesSkipped          *prometheus.CounterVec
	tokensPerLine         *prometheus.HistogramVec
	statePerLine          *prometheus.HistogramVec
	demand                *prometheus.GaugeVec
}

var _ registry.Limiter = (*DrainLimiter)(nil)

func newMetrics(reg prometheus.Registerer) drainLimiterMetrics {
	return drainLimiterMetrics{
		totalSpansCompacted: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_spans_compacted_total",
			Help:      "The total amount of spans compacted per tenant",
		}, []string{"tenant"}),
		patternsEvictedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_evicted_total",
			Help:      "The total amount of patterns evicted per tenant",
		}, []string{"tenant"}),
		patternsPrunedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_pruned_total",
			Help:      "The total amount of patterns pruned per tenant",
		}, []string{"tenant"}),
		patternsDetectedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_patterns_detected_total",
			Help:      "The total amount of patterns detected per tenant",
		}, []string{"tenant"}),
		linesSkipped: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_lines_skipped_total",
			Help:      "The total amount of lines skipped per tenant",
		}, []string{"tenant", "reason"}),
		tokensPerLine: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_tokens_per_line",
			Help:      "The number of tokens per line",
		}, []string{"tenant"}),
		statePerLine: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_state_per_line",
			Help:      "The number of state per line",
		}, []string{"tenant"}),
		demand: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_post_drain_demand_estimate",
			Help:      "The demand for the registry after applying DRAIN",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type DrainLimiter struct {
	mtx                       sync.Mutex
	backingLimiter            registry.Limiter
	dryRun                    bool
	drain                     *drain.Drain
	metricTotalSpansCompacted prometheus.Counter
	demand                    *registry.Cardinality
	demandGauge               prometheus.Gauge
	lastDemandUpdate          time.Time
}

func New(backingLimiter registry.Limiter, tenant string, dryRun bool) *DrainLimiter {
	l := &DrainLimiter{
		drain:                     makeDrain(tenant),
		dryRun:                    dryRun,
		backingLimiter:            backingLimiter,
		metricTotalSpansCompacted: metrics.totalSpansCompacted.WithLabelValues(tenant),
		demand:                    registry.NewCardinality(15*time.Minute, 5*time.Minute),
		demandGauge:               metrics.demand.WithLabelValues(tenant),
		lastDemandUpdate:          time.Now(),
	}

	return l
}

func makeDrain(tenant string) *drain.Drain {
	cfg := drain.DefaultConfig()

	drainMetrics := &drain.Metrics{
		PatternsEvictedTotal:  metrics.patternsEvictedTotal.WithLabelValues(tenant),
		PatternsPrunedTotal:   metrics.patternsPrunedTotal.WithLabelValues(tenant),
		PatternsDetectedTotal: metrics.patternsDetectedTotal.WithLabelValues(tenant),
		LinesSkipped:          metrics.linesSkipped.MustCurryWith(prometheus.Labels{"tenant": tenant}),
		TokensPerLine:         metrics.tokensPerLine.WithLabelValues(tenant),
		StatePerLine:          metrics.statePerLine.WithLabelValues(tenant),
	}

	return drain.New(cfg, drainMetrics, &drain.PunctuationAndSuffixAwareTokenizer{}, drain.DefaultIsDataHeuristic)
}

func (l *DrainLimiter) OnAdd(labelHash uint64, seriesCount uint32, lbls labels.Labels) (labels.Labels, uint64) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	spanName := lbls.Get("span_name")
	cluster := l.drain.Train(spanName, time.Now().UnixNano())
	if cluster != nil {
		newSpanName := cluster.String()
		if newSpanName != spanName {
			l.metricTotalSpansCompacted.Inc()
			builder := labels.NewBuilder(lbls)
			builder.Set("span_name", newSpanName)
			newLbls := builder.Labels()
			newLabelHash := newLbls.Hash()
			l.demand.Insert(newLabelHash)

			if !l.dryRun {
				lbls = newLbls
				labelHash = newLabelHash
			}
		}
	}

	return l.backingLimiter.OnAdd(labelHash, seriesCount, lbls)
}

func (l *DrainLimiter) OnUpdate(labelHash uint64, seriesCount uint32) {
	l.demand.Insert(labelHash)
	l.backingLimiter.OnUpdate(labelHash, seriesCount)
}

func (l *DrainLimiter) OnDelete(labelHash uint64, seriesCount uint32) {
	l.backingLimiter.OnDelete(labelHash, seriesCount)
}

func (l *DrainLimiter) OnPruneStaleSeries() {
	l.demandGauge.Set(float64(l.demand.Estimate()))
	l.demand.Advance()
}
