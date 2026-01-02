package registry

import (
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/drain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
)

type drainSanitizerMetrics struct {
	totalSpansCompacted *prometheus.CounterVec
	demand              *prometheus.GaugeVec
}

func newMetrics(reg prometheus.Registerer) drainSanitizerMetrics {
	return drainSanitizerMetrics{
		totalSpansCompacted: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_drain_spans_compacted_total",
			Help:      "The total amount of spans compacted per tenant",
		}, []string{"tenant"}),
		demand: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "tempo",
			Name:      "metrics_generator_registry_post_drain_demand_estimate",
			Help:      "The demand for the registry after applying DRAIN",
		}, []string{"tenant"}),
	}
}

var metrics = newMetrics(prometheus.DefaultRegisterer)

type DrainSanitizer struct {
	mtx    sync.Mutex
	drain  *drain.Drain
	dryRun bool
	demand *Cardinality

	metricTotalSpansCompacted prometheus.Counter
	demandGauge               prometheus.Gauge

	// channels for periodic maintenance. these allow us to avoid spawning
	// additional goroutines and rely on the fact that Sanitize is called
	// frequently from the registry.
	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewDrainSanitizer(tenant string, dryRun bool) *DrainSanitizer {
	return &DrainSanitizer{
		drain:                     drain.New(tenant, drain.DefaultConfig()),
		dryRun:                    dryRun,
		metricTotalSpansCompacted: metrics.totalSpansCompacted.WithLabelValues(tenant),
		demand:                    NewCardinality(15*time.Minute, 5*time.Minute),
		demandGauge:               metrics.demand.WithLabelValues(tenant),
		demandUpdateChan:          time.Tick(15 * time.Second),
		pruneChan:                 time.Tick(5 * time.Minute),
	}
}

func (s *DrainSanitizer) Sanitize(lbls labels.Labels) labels.Labels {
	s.doPeriodicMaintenance()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	spanName := lbls.Get("span_name")
	cluster := s.drain.Train(spanName)
	// drain has various limits to prevent excessive memory usage, etc. in these
	// cases, we will just return the original labels.
	if cluster == nil {
		s.demand.Insert(lbls.Hash())
		return lbls
	}

	// before drain has found a pattern, it will return the original span name.
	// in this case, we can avoid the expensive label building and just return
	// the original labels.
	newSpanName := cluster.String()
	if newSpanName == spanName {
		s.demand.Insert(lbls.Hash())
		return lbls
	}

	s.metricTotalSpansCompacted.Inc()
	builder := labels.NewBuilder(lbls)
	builder.Set("span_name", newSpanName)
	newLbls := builder.Labels()
	newLabelHash := newLbls.Hash()
	s.demand.Insert(newLabelHash)

	if s.dryRun {
		return lbls
	}

	return newLbls
}

func (s *DrainSanitizer) doPeriodicMaintenance() {
	select {
	case <-s.demandUpdateChan:
		s.demandGauge.Set(float64(s.demand.Estimate()))
	case <-s.pruneChan:
		s.demand.Advance()
		s.drain.Prune()
	default:
		break
	}
}
