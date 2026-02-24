package registry

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/schema"
)

const (
	overflowValue = "__cardinality_overflow__"
	// demandUpdateInterval controls how often the cardinality estimate from HLL
	// is refreshed and the overLimit flag and demand gauge are updated.
	// kept at 15s to limit lock contention with Limit() (hot path) which shares the same mutex
	demandUpdateInterval = 15 * time.Second
)

var metricLabelValuesLimited = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_registry_label_values_limited_total",
	Help:      "Total number of times a label value was limited due to exceeding the per-label cardinality limit",
}, []string{"tenant", "label_name"})

var metricLabelCardinalityDemand = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "tempo",
	Name:      "metrics_generator_registry_label_cardinality_demand_estimate",
	Help:      "Estimated cardinality demand (distinct value count) for each label, each tenant",
}, []string{"tenant", "label_name"})

// maxCardinalityFunc returns the MaxCardinalityPerLabel config value for the tenant.
type maxCardinalityFunc func(tenant string) uint64

type labelCardinalityState struct {
	sketch    *Cardinality
	overLimit bool // cached flag, updated periodically in maintenance tick
}

// PerLabelLimiter caps the number of distinct values any single label can have.
// When a label's estimated cardinality exceeds maxCardinality, its value is replaced
// with '__cardinality_overflow__' while all other labels are preserved.
//
// This is conceptually a limiter, not a sanitizer - it enforces a cardinality ceiling
// rather than normalizing label values (like DrainSanitizer does for span names).
// It runs in the label-building pipeline after sanitization but before the global
// entity limiter, making the processing order: sanitize -> per-label limit -> entity limit.
type PerLabelLimiter struct {
	mtx                sync.Mutex
	tenant             string
	maxCardinalityFunc maxCardinalityFunc
	maxCardinality     atomic.Uint64 // refreshed on demand update tick, read atomically in Limit() hot path

	labelsState   map[string]*labelCardinalityState
	staleDuration time.Duration

	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewPerLabelLimiter(tenant string, maxCardinalityF maxCardinalityFunc, staleDuration time.Duration) *PerLabelLimiter {
	pll := &PerLabelLimiter{
		tenant:             tenant,
		maxCardinalityFunc: maxCardinalityF,
		labelsState:        make(map[string]*labelCardinalityState),
		staleDuration:      staleDuration,
		demandUpdateChan:   time.Tick(demandUpdateInterval),
		pruneChan:          time.Tick(removeStaleSeriesInterval),
	}
	// init on New, config is refreshed on demand update tick
	pll.maxCardinality.Store(maxCardinalityF(tenant))
	return pll
}

// Limit applies the per-label cardinality limit to the given labels.
// Labels whose estimated cardinality exceeds the configured max have their
// value replaced with __cardinality_overflow__.
func (s *PerLabelLimiter) Limit(lbls labels.Labels) labels.Labels {
	// do maintenance check as the first thing to ensure maxCardinality
	// is refreshed from runtime overrides. without this,
	// a limiter that starts disabled would never be enabled without restart.
	s.doPeriodicMaintenance()

	// maxCardinality is zero, so limiter is disabled, return labels as is
	if s.maxCardinality.Load() == 0 {
		return lbls
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Defer builder creation until we actually need to modify a label.
	// In the common case (no overflow), we avoid the allocations entirely.
	var builder *labels.Builder
	lbls.Range(func(l labels.Label) {
		// skip over the metadata labels
		if schema.IsMetadataLabel(l.Name) {
			return
		}

		state := s.getOrCreateState(l.Name)

		// we always insert the ORIGINAL value to hash even while overflowing,
		// which prevents the estimate from artificially dropping.
		// It will make sure that recovery (label going back under limit) only happens when the
		// actual incoming data has lower cardinality AND the old sketches have been rotated out.
		//
		// If we inserted the overflowValue, then the estimate would drop and cause oscillation:
		// over limit -> Add overflowValue -> estimate drops -> under limit -> real values -> over limit ->...
		// Insert acquires its own internal mu lock on the sketch.
		state.sketch.Insert(xxhash.Sum64String(l.Value))

		// we are over the limit, replace label value and capture the metric
		if state.overLimit {
			// Lazy init: only create once, so previous Set calls are preserved
			// when multiple labels overflow in the same series
			if builder == nil {
				builder = labels.NewBuilder(lbls)
			}
			builder.Set(l.Name, overflowValue)
			metricLabelValuesLimited.WithLabelValues(s.tenant, l.Name).Inc()
		}
	})

	// No labels were limited, return the original labels as is.
	if builder == nil {
		return lbls
	}
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
// and do sketch operations outside it, and then update the labelsState under lock.
func (s *PerLabelLimiter) doPeriodicMaintenance() {
	select {
	case <-s.demandUpdateChan:
		// step 1: refresh maxCardinality config from override
		// fetch once per tick and cache atomically, the limit is the same for all labels in a tenant
		maxCardinality := s.maxCardinalityFunc(s.tenant)
		s.maxCardinality.Store(maxCardinality)

		// if the check is disabled, skip the demand update and, exit early.
		// no data is being inserted into the sketch, so nothing to estimate or publish
		if maxCardinality == 0 {
			return
		}

		// step 2: update estimate and publish demand estimate metric
		s.mtx.Lock()
		for labelName, state := range s.labelsState {
			estimate := state.sketch.Estimate()
			state.overLimit = estimate > maxCardinality
			metricLabelCardinalityDemand.WithLabelValues(s.tenant, labelName).Set(float64(estimate))
		}
		s.mtx.Unlock()
	case <-s.pruneChan:
		s.mtx.Lock()
		// label names come from config and are mostly stable, so stale entries
		// in labelsState are unlikely to grow unboundedly, so we don't clean up.
		for _, state := range s.labelsState {
			state.sketch.Advance()
		}
		s.mtx.Unlock()
	default:
	}
}
