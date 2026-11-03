package parquetquery

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricPredicateDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Subsystem: "parquetquery",
	Name:      "predicate_decisions_total",
	Help:      "Column chunks, pages and values inspected by iterator predicates, by tenant, level and whether they were kept or pruned.",
}, []string{"tenant", "level", "decision"})

// flushMetrics adds s to tenant's predicate decision counters and resets it.
func (s *PredicateStats) flushMetrics(tenant string) {
	addDecisions(tenant, "column_chunk", s.KeptColumnChunks, s.InspectedColumnChunks)
	addDecisions(tenant, "page", s.KeptPages, s.InspectedPages)
	addDecisions(tenant, "value", s.KeptValues, s.InspectedValues)
	*s = PredicateStats{}
}

func addDecisions(tenant, level string, nKept, nInspected int64) {
	if nKept > 0 {
		metricPredicateDecisions.WithLabelValues(tenant, level, "kept").Add(float64(nKept))
	}
	if n := nInspected - nKept; n > 0 {
		metricPredicateDecisions.WithLabelValues(tenant, level, "pruned").Add(float64(n))
	}
}
