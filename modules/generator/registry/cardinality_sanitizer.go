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
	Help:      "Total number of label values replaced with __cardinality_overflow__ by the per-label cardinality sanitizer",
}, []string{"tenant"})

type labelCardinalityState struct {
	sketch    *Cardinality
	overLimit bool // cached flag, updated periodically in maintenance tick
}

// CardinalitySanitizer replaces individual label values with __cardinality_overflow__
// when the estimated cardinality of that label exceeds maxCardinality.
// Unlike the global series limiter which collapses all labels into a single overflow
// entity, this preserves all other labels and only overflows the problematic one.
type CardinalitySanitizer struct {
	mtx            sync.Mutex
	tenant         string
	maxCardinality uint64
	labelsState    map[string]*labelCardinalityState
	staleDuration  time.Duration

	overflowCounter prometheus.Counter

	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewCardinalitySanitizer(tenant string, maxCardinality uint64, staleDuration time.Duration) *CardinalitySanitizer {
	return &CardinalitySanitizer{
		tenant:           tenant,
		maxCardinality:   maxCardinality,
		labelsState:      make(map[string]*labelCardinalityState),
		staleDuration:    staleDuration,
		overflowCounter:  metricCardinalityLimitOverflows.WithLabelValues(tenant),
		demandUpdateChan: time.Tick(15 * time.Second),
		pruneChan:        time.Tick(5 * time.Minute),
	}
}

func (s *CardinalitySanitizer) Sanitize(lbls labels.Labels) labels.Labels {
	// disabled, return as is
	if s.maxCardinality == 0 {
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
		// and cause oscillation: over→overflow→estimate drops→under→real values→over→...
		state.sketch.Insert(xxhash.Sum64String(l.Value))

		if state.overLimit {
			builder.Set(l.Name, overflowValue)
			s.overflowCounter.Inc()
		}
	})

	return builder.Labels()
}

func (s *CardinalitySanitizer) getOrCreateState(labelName string) *labelCardinalityState {
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
// demand updates (every 15s) and pruning (every 5m). This blocks Sanitize()
// callers for the duration. In practice this is fast, so it's acceptable.
// If it becomes a problem, snapshot the labelsState slice under the lock
// and do sketch operations outside it
func (s *CardinalitySanitizer) doPeriodicMaintenance() {
	select {
	case <-s.demandUpdateChan:
		s.mtx.Lock()
		for _, state := range s.labelsState {
			state.overLimit = state.sketch.Estimate() > s.maxCardinality
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
