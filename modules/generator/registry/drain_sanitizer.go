package registry

import (
	"sync"
	"time"

	"github.com/grafana/tempo/pkg/drain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
)

const labelSpanName = "span_name"

var (
	metricTotalSpansSanitized = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_spans_sanitized_total",
		Help:      "The total amount of spans sanitized per tenant",
	}, []string{"tenant"})
	metricPostSanitizationDemand = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_registry_post_sanitization_demand_estimate",
		Help:      "The demand for the registry after applying span name sanitization",
	}, []string{"tenant"})
)

// sanitizeModeFunc returns the current span name sanitization mode for the tenant.
type sanitizeModeFunc func(tenant string) string

type DrainSanitizer struct {
	mtx    sync.Mutex
	drain  *drain.Drain
	demand *Cardinality

	tenant        string
	sanitizeModeF sanitizeModeFunc

	metricTotalSpansSanitized prometheus.Counter
	demandGauge               prometheus.Gauge

	// channels for periodic maintenance. these allow us to avoid spawning
	// additional goroutines and rely on the fact that Sanitize is called
	// frequently from the registry.
	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewDrainSanitizer(tenant string, sanitizeModeF sanitizeModeFunc, staleDuration time.Duration) *DrainSanitizer {
	return &DrainSanitizer{
		drain:                     drain.New(tenant, drain.DefaultConfig()),
		tenant:                    tenant,
		sanitizeModeF:             sanitizeModeF,
		metricTotalSpansSanitized: metricTotalSpansSanitized.WithLabelValues(tenant),
		demand:                    NewCardinality(staleDuration, removeStaleSeriesInterval),
		demandGauge:               metricPostSanitizationDemand.WithLabelValues(tenant),
		demandUpdateChan:          time.Tick(15 * time.Second),
		pruneChan:                 time.Tick(5 * time.Minute),
	}
}

func (s *DrainSanitizer) Sanitize(lbls labels.Labels) labels.Labels {
	// Check the override at runtime so that changes to the sanitization mode override
	// take effect without restarting the generator.
	mode := s.sanitizeModeF(s.tenant)
	if mode == SpanNameSanitizationDisabled {
		return lbls
	}

	s.doPeriodicMaintenance()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	spanName := lbls.Get(labelSpanName)
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

	s.metricTotalSpansSanitized.Inc()
	builder := labels.NewBuilder(lbls)
	builder.Set(labelSpanName, newSpanName)
	newLbls := builder.Labels()
	newLabelHash := newLbls.Hash()
	s.demand.Insert(newLabelHash)

	// in dry-run mode, return the labels without modifying but capture metrics
	if mode == SpanNameSanitizationDryRun {
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
