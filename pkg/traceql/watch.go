package traceql

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type WatcherSpec struct {
	// Attribute is the TraceQL attribute identifier to watch
	Attribute string
	// Type selects the watcher behavior.
	Type string
}

// NewWatcher builds a SpanWatcher from a spec, parsing the attribute
// identifier and selecting the watcher type.
func NewWatcher(spec WatcherSpec) (SpanWatcher, error) {
	if spec.Attribute == "" {
		return nil, fmt.Errorf("watch attribute must not be empty")
	}
	attr, err := ParseIdentifier(spec.Attribute)
	if err != nil {
		// Not a scoped/intrinsic identifier (e.g. a synthetic attribute name such
		// as "aggregation.is_summary"); treat it as an unscoped attribute name.
		attr = NewAttribute(spec.Attribute)
	}
	switch spec.Type {
	case "", "presence":
		return NewAttributePresenceWatcher(attr, spec.Attribute), nil
	default:
		return nil, fmt.Errorf("unknown watcher type %q for attribute %q", spec.Type, spec.Attribute)
	}
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
type spanWatchers struct {
	mtx    sync.Mutex
	obs    []SpanWatcher
	active int
}

func (s *spanWatchers) Add(watchers ...SpanWatcher) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Only the active watchers need their attributes fetched.
	conds := make([]Condition, 0, s.active)
	for _, watcher := range s.obs[:s.active] {
		conds = append(conds, watcher.Conditions()...)
	}
	return conds
}

func (s *spanWatchers) WatchSpans(spans []*Spanset) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

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
func (s *spanWatchers) WatchSpan(span Span) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.active == 0 {
		return // done, exit early
	}

	s.watch(span)
}

// watch walks the active prefix for a single span.
// When a watcher goes inactive, swap it past the boundary so it's retained but skipped on future calls.
// Caller must hold s.mtx.
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
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.active > 0
}

func (s *spanWatchers) Stats() map[string]int64 {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	stats := make(map[string]int64)
	for _, watcher := range s.obs {
		for k, v := range watcher.Stats() {
			stats[k] += v
		}
	}
	return stats
}
