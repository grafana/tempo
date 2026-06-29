package traceql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func TestAttrPresenceObserver(t *testing.T) {
	attr := NewAttribute("aggregation.is_summary")
	const metricKey = "myMetric"

	t.Run("attribute absent", func(t *testing.T) {
		o := NewAttributePresenceObserver(attr, metricKey)

		// A span without the attribute keeps the observer interested.
		keepLooking := o.ObserveSpan(newMockSpan([]byte{1}).WithSpanString("foo", "bar"))
		assert.True(t, keepLooking, "observer should keep looking")
		assert.True(t, o.Active())
		assert.Nil(t, o.Stats(), "no stats until the attribute is seen")
	})

	t.Run("attribute present", func(t *testing.T) {
		o := NewAttributePresenceObserver(attr, metricKey)

		keepLooking := o.ObserveSpan(newMockSpan([]byte{1}).WithAttrBool("aggregation.is_summary", true))
		assert.False(t, keepLooking, "observer is done once it finds the attribute")
		assert.False(t, o.Active())
		assert.Equal(t, map[string]int64{metricKey: 1}, o.Stats(), "stats reported under the observer's metric key")
	})

	t.Run("stays inactive after finding", func(t *testing.T) {
		o := NewAttributePresenceObserver(attr, metricKey)

		require.False(t, o.ObserveSpan(newMockSpan([]byte{1}).WithAttrBool("aggregation.is_summary", true)))
		// A later span without the attribute must not flip the observer back on.
		assert.False(t, o.ObserveSpan(newMockSpan([]byte{2}).WithSpanString("foo", "bar")))
		assert.False(t, o.Active())
		assert.Equal(t, int64(1), o.Stats()[metricKey])
	})

	t.Run("conditions request the attribute", func(t *testing.T) {
		o := NewAttributePresenceObserver(attr, metricKey)
		conds := o.Conditions()
		require.Len(t, conds, 1)
		assert.Equal(t, attr, conds[0].Attribute)
		assert.Equal(t, OpNone, conds[0].Op)
		// The condition carries a CallBack so the fetch layer can stop loading the
		// attribute once the observer goes inactive.
		assert.NotNil(t, conds[0].CallBack)
	})
}

// testObserver is a controllable SpanObserver for exercising spanObservers.
type testObserver struct {
	conds        []Condition
	stats        map[string]int64
	deactivateOn int // 0 = always active; N = goes inactive on the Nth ObserveSpan call
	observed     int
	active       bool
}

func newTestObserver(deactivateOn int, stats map[string]int64, conds ...Condition) *testObserver {
	return &testObserver{conds: conds, stats: stats, deactivateOn: deactivateOn, active: true}
}

func (o *testObserver) Conditions() []Condition { return o.conds }

func (o *testObserver) ObserveSpan(Span) bool {
	o.observed++
	if o.deactivateOn > 0 && o.observed >= o.deactivateOn {
		o.active = false
	}
	return o.active
}

func (o *testObserver) Active() bool { return o.active }

func (o *testObserver) Stats() map[string]int64 { return o.stats }

// spansetOf wraps spans into a single spanset for feeding spanObservers.ObserveSpans.
func spansetOf(spans ...Span) []*Spanset {
	return []*Spanset{{Spans: spans}}
}

func TestSpanObservers_ZeroValueUsable(t *testing.T) {
	var s spanObservers
	assert.False(t, s.Active())
	assert.Empty(t, s.Conditions())
	assert.Empty(t, s.Stats())
	// Observing on an empty set is a no-op and leaves no active observers.
	s.ObserveSpans(spansetOf(newMockSpan([]byte{1})))
	assert.False(t, s.Active())
}

func TestSpanObservers_ConditionsCoverActivePrefixOnly(t *testing.T) {
	condA := Condition{Attribute: NewAttribute("a"), Op: OpNone}
	condB := Condition{Attribute: NewAttribute("b"), Op: OpNone}

	var s spanObservers
	s.Add(
		newTestObserver(1, nil, condA), // goes inactive after one span
		newTestObserver(0, nil, condB), // always active
	)

	assert.ElementsMatch(t, []Condition{condA, condB}, s.Conditions(), "both observers active up front")

	s.ObserveSpans(spansetOf(newMockSpan([]byte{1})))

	assert.Equal(t, []Condition{condB}, s.Conditions(), "inactive observer no longer contributes conditions")
}

func TestSpanObservers_ObserveSpanPartitioning(t *testing.T) {
	a := newTestObserver(1, nil) // inactive after 1 span
	b := newTestObserver(2, nil) // inactive after 2 spans
	c := newTestObserver(0, nil) // always active

	var s spanObservers
	s.Add(a, b, c)

	// First span: a goes inactive, b and c still active.
	s.ObserveSpans(spansetOf(newMockSpan([]byte{1})))
	assert.True(t, s.Active())

	// Second span: b goes inactive (a is skipped, only the active prefix is walked).
	s.ObserveSpans(spansetOf(newMockSpan([]byte{2})))
	assert.True(t, s.Active())

	// Every active observer was visited exactly once per span it was active for,
	// and the inactive ones were not revisited.
	assert.Equal(t, 1, a.observed, "a observed only the first span")
	assert.Equal(t, 2, b.observed, "b observed the first two spans")
	assert.Equal(t, 2, c.observed, "c observed both spans")
}

func TestSpanObservers_ObserveSpanReturnsFalseWhenAllInactive(t *testing.T) {
	var s spanObservers
	s.Add(newTestObserver(1, nil))

	s.ObserveSpans(spansetOf(newMockSpan([]byte{1})))
	assert.False(t, s.Active(), "no active observers remain")
}

func TestSpanObservers_StatsAggregatesAllObservers(t *testing.T) {
	var s spanObservers
	s.Add(
		newTestObserver(0, map[string]int64{"x": 1, "shared": 2}),
		newTestObserver(0, map[string]int64{"y": 5, "shared": 3}),
		newTestObserver(0, nil), // contributes nothing
	)

	assert.Equal(t, map[string]int64{"x": 1, "y": 5, "shared": 5}, s.Stats())
}

func TestEngineExecuteSearch_IsSummaryObserver(t *testing.T) {
	now := time.Now()

	plainSpan := func(id byte) *mockSpan {
		return &mockSpan{
			id:                 []byte{id},
			startTimeUnixNanos: uint64(now.UnixNano()),
			durationNanos:      uint64(100 * time.Millisecond),
			attributes: map[Attribute]Static{
				NewAttribute("foo"): NewStaticString("value"),
			},
		}
	}
	summarySpan := func(id byte) *mockSpan {
		s := plainSpan(id)
		s.attributes[NewAttribute("aggregation.is_summary")] = NewStaticBool(true)
		return s
	}

	const wantAbsent = int64(-1)

	tests := []struct {
		name  string
		query string
		spans []*mockSpan
		want  int64 // expected metric value, or wantAbsent if the key should be missing
	}{
		{
			name:  "hint set and summary span present",
			query: `{ .foo = "value" } with(report_is_summary=true)`,
			spans: []*mockSpan{plainSpan(1), summarySpan(2)},
			want:  1,
		},
		{
			name:  "hint set but no summary span",
			query: `{ .foo = "value" } with(report_is_summary=true)`,
			spans: []*mockSpan{plainSpan(1)},
			want:  wantAbsent,
		},
		{
			name:  "hint absent",
			query: `{ .foo = "value" }`,
			spans: []*mockSpan{summarySpan(2)},
			want:  wantAbsent,
		},
		{
			name:  "hint explicitly false",
			query: `{ .foo = "value" } with(report_is_summary=false)`,
			spans: []*mockSpan{summarySpan(2)},
			want:  wantAbsent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans := make([]Span, len(tt.spans))
			for i, s := range tt.spans {
				spans[i] = s
			}
			fetcher := &MockSpanSetFetcher{
				iterator: &MockSpanSetIterator{
					results: []*Spanset{
						{
							TraceID:         []byte{1},
							RootSpanName:    "HTTP GET",
							RootServiceName: "my-service",
							Spans:           spans,
						},
					},
				},
			}

			req := &tempopb.SearchRequest{Query: tt.query, SpansPerSpanSet: 10, Limit: 10}
			resp, err := NewEngine().ExecuteSearch(context.Background(), req, fetcher)
			require.NoError(t, err)
			require.NotNil(t, resp.Metrics)

			got, ok := resp.Metrics.AdditionalMetrics[tempopb.AdditionalMetricAggregationIsSummary]
			if tt.want == wantAbsent {
				assert.False(t, ok, "expected no aggregationIsSummary metric, got %d", got)
				return
			}
			assert.True(t, ok, "expected aggregationIsSummary metric to be present")
			assert.Equal(t, tt.want, got)
		})
	}
}
