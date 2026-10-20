package registry

import (
	"maps"
	"strings"
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
	demandUpdateInterval = 15 * time.Second
	// recentValuesSize is the number of slots in each label's cache of recently
	// inserted value hashes.
	recentValuesSize = 128
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
	sketch *Cardinality
	// recent is a direct-mapped cache of value hashes recently inserted into
	// sketch, each xored with a mix of the sketch generation so that entries
	// expire when the sketch advances. Inserting a hash again into the same
	// sketch doesn't change it, so a hit skips the insert and its lock.
	recent    [recentValuesSize]atomic.Uint64
	overLimit atomic.Bool // cached flag, updated periodically in maintenance tick
	// ownedName is the heap-owned (cloned) label name. The incoming label name
	// may alias a pooled/borrowed scratch buffer (see CloseAndBorrowLabels), so
	// we keep an owned copy to hand to metricLabelValuesLimited.WithLabelValues:
	// that retains the string, and a borrowed alias would be overwritten by a
	// later borrow, corrupting the exported metric and leaking child series.
	ownedName string
	// limitedCounter is the metricLabelValuesLimited child for ownedName,
	// created lazily on the first overflow and cached so the overflow path
	// stays allocation-free.
	limitedCounter     prometheus.Counter
	limitedCounterOnce sync.Once
}

func (st *labelCardinalityState) insert(hash uint64) {
	// +1 so that empty slots don't match in generation 0
	tag := hash ^ ((st.sketch.Generation() + 1) * 0x9E3779B97F4A7C15)
	slot := &st.recent[hash%recentValuesSize]
	if slot.Load() == tag {
		return
	}
	st.sketch.Insert(hash)
	slot.Store(tag)
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
	mtx                sync.Mutex // serializes additions to labelsState
	tenant             string
	maxCardinalityFunc maxCardinalityFunc
	maxCardinality     atomic.Uint64 // refreshed on demand update tick, read atomically in Limit() hot path

	// labelsState is copy-on-write so that Limit() reads it without locking.
	labelsState   atomic.Pointer[map[string]*labelCardinalityState]
	staleDuration time.Duration

	demandUpdateChan <-chan time.Time
	pruneChan        <-chan time.Time
}

func NewPerLabelLimiter(tenant string, maxCardinalityF maxCardinalityFunc, staleDuration time.Duration) *PerLabelLimiter {
	pll := &PerLabelLimiter{
		tenant:             tenant,
		maxCardinalityFunc: maxCardinalityF,
		staleDuration:      staleDuration,
		demandUpdateChan:   time.Tick(demandUpdateInterval),
		pruneChan:          time.Tick(removeStaleSeriesInterval),
	}
	pll.labelsState.Store(&map[string]*labelCardinalityState{})
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
		state.insert(xxhash.Sum64String(l.Value))

		// we are over the limit, replace label value and capture the metric
		if state.overLimit.Load() {
			// Lazy init: only create once, so previous Set calls are preserved
			// when multiple labels overflow in the same series
			if builder == nil {
				builder = labels.NewBuilder(lbls)
			}
			builder.Set(l.Name, overflowValue)
			// Use the owned label name, never the borrowed l.Name:
			// WithLabelValues retains the string header, so a borrowed alias
			// would be corrupted once the scratch buffer is reused (and each
			// corrupted lookup would leak a new child series). Cache the child
			// counter on first use to keep this path allocation-free.
			state.limitedCounterOnce.Do(func() {
				state.limitedCounter = metricLabelValuesLimited.WithLabelValues(s.tenant, state.ownedName)
			})
			state.limitedCounter.Inc()
		}
	})

	// No labels were limited, return the original labels as is.
	if builder == nil {
		return lbls
	}
	return builder.Labels()
}

func (s *PerLabelLimiter) getOrCreateState(labelName string) *labelCardinalityState {
	if state, ok := (*s.labelsState.Load())[labelName]; ok {
		return state
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	current := *s.labelsState.Load()
	if state, ok := current[labelName]; ok {
		return state
	}
	// labelName may alias a pooled/borrowed scratch buffer (see
	// CloseAndBorrowLabels) that the caller reuses after this call returns.
	// Clone it once and retain only the owned copy (as the map key and as
	// state.ownedName); a retained alias would be overwritten by a later
	// borrow and corrupt both. Only the insert path clones; lookups above
	// compare by value, so the hot path (existing label name) stays
	// allocation-free.
	owned := strings.Clone(labelName)
	state := &labelCardinalityState{
		sketch:    NewCardinality(s.staleDuration, removeStaleSeriesInterval),
		ownedName: owned,
	}
	// label names come from config and are mostly stable, so copying the map
	// on each new name is cheap.
	next := make(map[string]*labelCardinalityState, len(current)+1)
	maps.Copy(next, current)
	next[owned] = state
	s.labelsState.Store(&next)
	return state
}

// doPeriodicMaintenance runs at most one of the demand update (every 15s) and
// the prune (every 5m) per call, on a snapshot of labelsState.
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
		for labelName, state := range *s.labelsState.Load() {
			estimate := state.sketch.Estimate()
			state.overLimit.Store(estimate > maxCardinality)
			metricLabelCardinalityDemand.WithLabelValues(s.tenant, labelName).Set(float64(estimate))
		}
	case <-s.pruneChan:
		// label names come from config and are mostly stable, so stale entries
		// in labelsState are unlikely to grow unboundedly, so we don't clean up.
		for _, state := range *s.labelsState.Load() {
			state.sketch.Advance()
		}
	default:
	}
}
