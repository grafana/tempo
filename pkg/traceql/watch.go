package traceql

import (
	"sync/atomic"
)

const SpanPruningAttribute = "aggregation.is_summary"

// NewSpanPruningWatcher returns a watcher that reports whether any matched span is a span-pruning summary span.
func NewSpanPruningWatcher() SpanWatcher {
	return NewAttributePresenceWatcher(NewAttribute(SpanPruningAttribute), SpanPruningAttribute)
}

// SpanWatcher inspects spans as they flow through the TraceQL engine and records something about them.
type SpanWatcher interface {
	// Conditions returns the fetch conditions the watcher needs so the attributes it cares about are loaded onto watched spans.
	Conditions() []Condition
	// WatchSpan inspects a single span.
	// It returns true while the watcher is still interested in further spans.
	WatchSpan(Span) bool
	// Active reports whether the watcher still wants to see spans.
	Active() bool
	// Stats returns the metrics gathered so far, keyed by metric name.
	Stats() map[string]int64
}

var _ SpanWatcher = (*attrPresenceWatcher)(nil)

type attrPresenceWatcher struct {
	attr      Attribute
	metricKey string
	active    atomic.Bool
}

// NewAttributePresenceWatcher returns an watcher that records whether any watched span carries attr.
// When the attribute is seen, Stats reports a count of 1 under metricKey.
func NewAttributePresenceWatcher(attr Attribute, metricKey string) SpanWatcher {
	o := &attrPresenceWatcher{attr: attr, metricKey: metricKey}
	o.active.Store(true)
	return o
}

func (a *attrPresenceWatcher) Conditions() []Condition {
	return []Condition{{Attribute: a.attr, Op: OpNone, CallBack: a.active.Load}}
}

func (a *attrPresenceWatcher) WatchSpan(span Span) bool {
	if !a.active.Load() {
		return false // already found; no longer interested
	}
	if _, ok := span.AttributeFor(a.attr); ok {
		a.active.Store(false)
		return false // found it; done
	}
	return true // keep looking
}

func (a *attrPresenceWatcher) Active() bool {
	return a.active.Load()
}

func (a *attrPresenceWatcher) Stats() map[string]int64 {
	if a.active.Load() {
		return nil
	}
	return map[string]int64{a.metricKey: 1}
}

// spanWatchers keeps all watchers but partitions them:
// (1) obs[:active] are still active
// (2) obs[active:] have gone inactive.
// Inactive watchers are never dropped, only moved past the boundary, so WatchSpans only walks the active prefix.
//
// spanWatchers does no locking of its own. Callers that share one spanWatchers across multiple
// goroutines (e.g. a metricsEvaluator with WithLock set, used across concurrently-evaluated
// blocks) are responsible for synchronizing every call, including Active().
type spanWatchers struct {
	obs    []SpanWatcher
	active int
}

func (s *spanWatchers) Add(watchers ...SpanWatcher) {
	for _, o := range watchers {
		s.obs = append(s.obs, o)
		if o.Active() {
			// Swap the newly-added active watcher into the active boundary so the
			// obs[:active] partition holds regardless of add order or whether any
			// already-added watcher has gone inactive.
			last := len(s.obs) - 1
			s.obs[s.active], s.obs[last] = s.obs[last], s.obs[s.active]
			s.active++
		}
	}
}

func (s *spanWatchers) Conditions() []Condition {
	// Only the active watchers need their attributes fetched.
	conds := make([]Condition, 0, s.active)
	for _, watcher := range s.obs[:s.active] {
		conds = append(conds, watcher.Conditions()...)
	}
	return conds
}

func (s *spanWatchers) WatchSpans(spans []*Spanset) {
	if s.active == 0 {
		return // done, exit early
	}

outer:
	for _, ss := range spans {
		for _, span := range ss.Spans {
			s.watch(span)
			if s.active == 0 {
				break outer
			}
		}
	}
}

// WatchSpan feeds a single span to the active watchers.
// It's the per-span equivalent of WatchSpans,
// used by hot paths that already iterate spans individually (e.g. the span-only metrics fetch) to avoid allocating a Spanset.
// It returns whether any watchers remain active, so callers can stop calling once all watchers are done.
func (s *spanWatchers) WatchSpan(span Span) bool {
	if s.active == 0 {
		return false // done, exit early
	}

	// Fast path: skip the swap-and-recheck loop in watch() for the common single-watcher case.
	if s.active == 1 {
		if !s.obs[0].WatchSpan(span) {
			// Don't truncate obs: Stats() below the active boundary still needs the watcher.
			s.active = 0
		}
		return s.active > 0
	}

	s.watch(span)
	return s.active > 0
}

// watch walks the active prefix for a single span.
// When a watcher goes inactive, swap it past the boundary so it's retained but skipped on future calls.
func (s *spanWatchers) watch(span Span) {
	for i := 0; i < s.active; {
		if s.obs[i].WatchSpan(span) {
			i++
			continue
		}
		s.active--
		s.obs[i], s.obs[s.active] = s.obs[s.active], s.obs[i]
		// don't advance i: re-check the watcher swapped into position i
	}
}

func (s *spanWatchers) Active() bool {
	return s.active > 0
}

func (s *spanWatchers) Stats() map[string]int64 {
	stats := make(map[string]int64)
	for _, watcher := range s.obs {
		for k, v := range watcher.Stats() {
			stats[k] += v
		}
	}
	return stats
}
