package traceql

import (
	"sync"
	"sync/atomic"

	"github.com/grafana/tempo/pkg/tempopb"
)

var aggregationIsSummaryAttribute = NewAttribute("aggregation.is_summary")

// SpanObserver inspects spans as they flow through the TraceQL engine and records something about them.
type SpanObserver interface {
	// Conditions returns the fetch conditions the observer needs so the
	// attributes it cares about are loaded onto observed spans.
	Conditions() []Condition
	// ObserveSpan inspects a single span.
	// It returns true while the observer is still interested in further spans and false once it has what it needs.
	ObserveSpan(Span) bool
	// Active reports whether the observer still wants to see spans.
	Active() bool
	// Stats returns the metrics gathered so far, keyed by metric name.
	Stats() map[string]int64
}

var _ SpanObserver = (*attrPresenceObserver)(nil)

type attrPresenceObserver struct {
	attr      Attribute
	metricKey string
	active   atomic.Bool
}

// NewAttributePresenceObserver returns an observer that records whether any
// observed span carries attr. When the attribute is seen, Stats reports a count
// of 1 under metricKey.
func NewAttributePresenceObserver(attr Attribute, metricKey string) SpanObserver {
	o := &attrPresenceObserver{attr: attr, metricKey: metricKey}
	o.active.Store(true)
	return o
}

// NewIsSummaryObserver returns the observer used to count queries that match a
// span carrying the aggregation.is_summary attribute. The engine decides when
// to install it (see the report_is_summary hint).
func NewIsSummaryObserver() SpanObserver {
	return NewAttributePresenceObserver(aggregationIsSummaryAttribute, tempopb.AdditionalMetricAggregationIsSummary)
}

func (a *attrPresenceObserver) Conditions() []Condition {
	return []Condition{{Attribute: a.attr, Op: OpNone, CallBack: a.active.Load }}
}

func (a *attrPresenceObserver) ObserveSpan(span Span) bool {
	if !a.active.Load() {
		return false // already found; no longer interested
	}
	if _, ok := span.AttributeFor(a.attr); ok {
		a.active.Store(false)
		return false // found it; done
	}
	return true // keep looking
}

func (a *attrPresenceObserver) Active() bool {
	return a.active.Load()
}

func (a *attrPresenceObserver) Stats() map[string]int64 {
	if a.active.Load() {
		return nil
	}
	return map[string]int64{a.metricKey: 1}
}

// spanObservers keeps all observers but partitions them:
// (1) obs[:active] are still active
// (2) obs[active:] have gone inactive.
// Inactive observers are never dropped, only moved past the boundary, so ObserveSpan only walks the active prefix.
type spanObservers struct {
	mtx    sync.Mutex
	obs    []SpanObserver
	active int
}

func (s *spanObservers) Add(observers ...SpanObserver) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, o := range observers {
		s.obs = append(s.obs, o)
		if o.Active() {
			s.active++
		}
	}
}

func (s *spanObservers) Conditions() []Condition {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Only the active observers need their attributes fetched.
	conds := make([]Condition, 0, s.active)
	for _, observer := range s.obs[:s.active] {
		conds = append(conds, observer.Conditions()...)
	}
	return conds
}

func (s *spanObservers) ObserveSpan(span Span) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Only iterate the active prefix. When an observer goes inactive, swap it
	// past the boundary so it's retained but skipped on future calls.
	for i := 0; i < s.active; {
		o := s.obs[i]
		if o.ObserveSpan(span) {
			i++
			continue
		}
		s.active--
		s.obs[i], s.obs[s.active] = s.obs[s.active], s.obs[i]
		// don't advance i: re-check the observer swapped into position i
	}

	return s.active > 0
}

func (s *spanObservers) Active() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.active > 0
}

func (s *spanObservers) Stats() map[string]int64 {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	stats := make(map[string]int64)
	for _, observer := range s.obs {
		for k, v := range observer.Stats() {
			stats[k] += v
		}
	}
	return stats
}
