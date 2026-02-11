package registry

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/schema"
)

const overflowValue = "__cardinality_overflow__"

var metricCardinalityLimitOverflows = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_registry_cardinality_limit_overflows_total",
	Help:      "Total number of label values replaced with __cardinality_overflow__ by the per-label cardinality limiter",
}, []string{"tenant"})

// maxCardinalityFunc returns the MaxCardinalityPerLabel config value for the tenant.
type maxCardinalityFunc func(tenant string) uint64

type labelCardinalityState struct {
	sketch    *Cardinality
	overLimit bool // cached flag, updated periodically in maintenance tick
}

// PerLabelLimiter caps the number of distinct values any single label can have.
// When a label's estimated cardinality exceeds maxCardinality, its value is replaced
// with __cardinality_overflow__ while all other labels are preserved.
//
// This is conceptually a limiter, not a sanitizer - it enforces a cardinality ceiling
// rather than normalizing label values (like DrainSanitizer does for span names).
// It runs in the label-building pipeline after sanitization but before the global
// entity limiter, making the processing order: sanitize -> per-label limit -> entity limit.
type PerLabelLimiter struct {
	mtx                sync.Mutex
	tenant             string
	maxCardinalityFunc maxCardinalityFunc

	labelsState   map[string]*labelCardinalityState
	staleDuration time.Duration

	overflowCounter prometheus.Counter

	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewPerLabelLimiter(tenant string, maxCardinalityF maxCardinalityFunc, staleDuration time.Duration) *PerLabelLimiter {
	return &PerLabelLimiter{
		tenant:             tenant,
		maxCardinalityFunc: maxCardinalityF,
		labelsState:        make(map[string]*labelCardinalityState),
		staleDuration:      staleDuration,
		overflowCounter:    metricCardinalityLimitOverflows.WithLabelValues(tenant),
		demandUpdateChan:   time.Tick(15 * time.Second),
		pruneChan:          time.Tick(5 * time.Minute),
	}
}

// Limit applies the per-label cardinality limit to the given labels.
// Labels whose estimated cardinality exceeds the configured max have their
// value replaced with __cardinality_overflow__.
func (s *PerLabelLimiter) Limit(lbls labels.Labels) labels.Labels {
	// MaxCardinalityPerLabel is zero, so limiter is disabled, return labels as is
	if s.maxCardinalityFunc(s.tenant) == 0 {
		return lbls
	}

	s.doPeriodicMaintenance()

	s.mtx.Lock()
	defer s.mtx.Unlock()

	builder := labels.NewBuilder(lbls)
	lbls.Range(func(l labels.Label) {
		// skip over the metadata labels
		if schema.IsMetadataLabel(l.Name) {
			return
		}

		state := s.getOrCreateState(l.Name)

		// Always insert the ORIGINAL value hash to track true demand.
		// If we inserted the overflow value, the estimate would drop to 1
		// and cause oscillation: over->overflow->estimate drops->under->real values->over->...
		state.sketch.Insert(xxhash.Sum64String(l.Value))

		if state.overLimit {
			builder.Set(l.Name, overflowValue)
			s.overflowCounter.Inc()
		}
	})

	return builder.Labels()
}

func (s *PerLabelLimiter) getOrCreateState(labelName string) *labelCardinalityState {
	state, ok := s.labelsState[labelName]
	if !ok {
		state = &labelCardinalityState{
			sketch: NewCardinality(s.staleDuration, removeStaleSeriesInterval),
		}
		s.labelsState[labelName] = state
	}
	return state
}

// doPeriodicMaintenance holds s.mtx while iterating labelsState for both
// demand updates (every 15s) and pruning (every 5m). This blocks Limit()
// callers for the duration. In practice this is fast, so it's acceptable.
// If it becomes a problem, snapshot the labelsState slice under the lock
// and do sketch operations outside it.
func (s *PerLabelLimiter) doPeriodicMaintenance() {
	select {
	case <-s.demandUpdateChan:
		s.mtx.Lock()
		for _, state := range s.labelsState {
			state.overLimit = state.sketch.Estimate() > s.maxCardinalityFunc(s.tenant)
		}
		s.mtx.Unlock()
	case <-s.pruneChan:
		s.mtx.Lock()
		for _, state := range s.labelsState {
			state.sketch.Advance()
		}
		s.mtx.Unlock()
	default:
	}
}
