package traceql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func TestAttrPresenceWatcher(t *testing.T) {
	attr := NewAttribute("aggregation.is_summary")
	const metricKey = "myMetric"

	t.Run("attribute absent", func(t *testing.T) {
		o := NewAttributePresenceWatcher(attr, metricKey)

		// A span without the attribute keeps the watcher interested.
		keepLooking := o.WatchSpan(newMockSpan([]byte{1}).WithSpanString("foo", "bar"))
		assert.True(t, keepLooking, "watcher should keep looking")
		assert.True(t, o.Active())
		assert.Nil(t, o.Stats(), "no stats until the attribute is seen")
	})

	t.Run("attribute present", func(t *testing.T) {
		o := NewAttributePresenceWatcher(attr, metricKey)

		keepLooking := o.WatchSpan(newMockSpan([]byte{1}).WithAttrBool("aggregation.is_summary", true))
		assert.False(t, keepLooking, "watcher is done once it finds the attribute")
		assert.False(t, o.Active())
		assert.Equal(t, map[string]int64{metricKey: 1}, o.Stats(), "stats reported under the watcher's metric key")
	})

	t.Run("stays inactive after finding", func(t *testing.T) {
		o := NewAttributePresenceWatcher(attr, metricKey)

		require.False(t, o.WatchSpan(newMockSpan([]byte{1}).WithAttrBool("aggregation.is_summary", true)))
		// A later span without the attribute must not flip the watcher back on.
		assert.False(t, o.WatchSpan(newMockSpan([]byte{2}).WithSpanString("foo", "bar")))
		assert.False(t, o.Active())
		assert.Equal(t, int64(1), o.Stats()[metricKey])
	})

	t.Run("conditions request the attribute", func(t *testing.T) {
		o := NewAttributePresenceWatcher(attr, metricKey)
		conds := o.Conditions()
		require.Len(t, conds, 1)
		assert.Equal(t, attr, conds[0].Attribute)
		assert.Equal(t, OpNone, conds[0].Op)
		// The condition carries a CallBack so the fetch layer can stop loading the
		// attribute once the watcher goes inactive.
		assert.NotNil(t, conds[0].CallBack)
	})
}

// testWatcher is a controllable SpanWatcher for exercising spanWatchers.
type testWatcher struct {
	conds        []Condition
	stats        map[string]int64
	deactivateOn int // 0 = always active; N = goes inactive on the Nth WatchSpan call
	watched      int
	active       bool
}

func newTestWatcher(deactivateOn int, stats map[string]int64, conds ...Condition) *testWatcher {
	return &testWatcher{conds: conds, stats: stats, deactivateOn: deactivateOn, active: true}
}

func (o *testWatcher) Conditions() []Condition { return o.conds }

func (o *testWatcher) WatchSpan(Span) bool {
	o.watched++
	if o.deactivateOn > 0 && o.watched >= o.deactivateOn {
		o.active = false
	}
	return o.active
}

func (o *testWatcher) Active() bool { return o.active }

func (o *testWatcher) Stats() map[string]int64 { return o.stats }

// spansetOf wraps spans into a single spanset for feeding spanWatchers.WatchSpans.
func spansetOf(spans ...Span) []*Spanset {
	return []*Spanset{{Spans: spans}}
}

func TestSpanWatchers_ZeroValueUsable(t *testing.T) {
	var s spanWatchers
	assert.False(t, s.Active())
	assert.Empty(t, s.Conditions())
	assert.Empty(t, s.Stats())
	// Observing on an empty set is a no-op and leaves no active watchers.
	s.WatchSpans(spansetOf(newMockSpan([]byte{1})))
	assert.False(t, s.Active())
}

func TestSpanWatchers_ConditionsCoverActivePrefixOnly(t *testing.T) {
	condA := Condition{Attribute: NewAttribute("a"), Op: OpNone}
	condB := Condition{Attribute: NewAttribute("b"), Op: OpNone}

	var s spanWatchers
	s.Add(
		newTestWatcher(1, nil, condA), // goes inactive after one span
		newTestWatcher(0, nil, condB), // always active
	)

	assert.ElementsMatch(t, []Condition{condA, condB}, s.Conditions(), "both watchers active up front")

	s.WatchSpans(spansetOf(newMockSpan([]byte{1})))

	assert.Equal(t, []Condition{condB}, s.Conditions(), "inactive watcher no longer contributes conditions")
}

func TestSpanWatchers_WatchSpanPartitioning(t *testing.T) {
	a := newTestWatcher(1, nil) // inactive after 1 span
	b := newTestWatcher(2, nil) // inactive after 2 spans
	c := newTestWatcher(0, nil) // always active

	var s spanWatchers
	s.Add(a, b, c)

	// First span: a goes inactive, b and c still active.
	s.WatchSpans(spansetOf(newMockSpan([]byte{1})))
	assert.True(t, s.Active())

	// Second span: b goes inactive (a is skipped, only the active prefix is walked).
	s.WatchSpans(spansetOf(newMockSpan([]byte{2})))
	assert.True(t, s.Active())

	// Every active watcher was visited exactly once per span it was active for,
	// and the inactive ones were not revisited.
	assert.Equal(t, 1, a.watched, "a watched only the first span")
	assert.Equal(t, 2, b.watched, "b watched the first two spans")
	assert.Equal(t, 2, c.watched, "c watched both spans")
}

func TestSpanWatchers_WatchSpanReturnsFalseWhenAllInactive(t *testing.T) {
	var s spanWatchers
	s.Add(newTestWatcher(1, nil))

	s.WatchSpans(spansetOf(newMockSpan([]byte{1})))
	assert.False(t, s.Active(), "no active watchers remain")
}

func TestSpanWatchers_StatsAggregatesAllWatchers(t *testing.T) {
	var s spanWatchers
	s.Add(
		newTestWatcher(0, map[string]int64{"x": 1, "shared": 2}),
		newTestWatcher(0, map[string]int64{"y": 5, "shared": 3}),
		newTestWatcher(0, nil), // contributes nothing
	)

	assert.Equal(t, map[string]int64{"x": 1, "y": 5, "shared": 5}, s.Stats())
}

func TestEngineExecuteSearch_IsSummaryWatcher(t *testing.T) {
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

	const (
		summaryKey = "aggregation.is_summary"
		wantAbsent = int64(-1)
	)

	tests := []struct {
		name           string
		installWatcher bool
		spans          []*mockSpan
		want           int64 // expected metric value, or wantAbsent if the key should be missing
	}{
		{
			name:           "watcher installed and summary span present",
			installWatcher: true,
			spans:          []*mockSpan{plainSpan(1), summarySpan(2)},
			want:           1,
		},
		{
			name:           "watcher installed but no summary span",
			installWatcher: true,
			spans:          []*mockSpan{plainSpan(1)},
			want:           wantAbsent,
		},
		{
			name:           "no watcher installed",
			installWatcher: false,
			spans:          []*mockSpan{summarySpan(2)},
			want:           wantAbsent,
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

			var opts []CompileOption
			if tt.installWatcher {
				opts = append(opts, WithWatchers(NewSpanPruningWatcher()))
			}

			req := &tempopb.SearchRequest{Query: `{ .foo = "value" }`, SpansPerSpanSet: 10, Limit: 10}
			resp, err := NewEngine().ExecuteSearch(context.Background(), req, fetcher, opts...)
			require.NoError(t, err)
			require.NotNil(t, resp.Metrics)

			got, ok := resp.Metrics.AdditionalMetrics[summaryKey]
			if tt.want == wantAbsent {
				assert.False(t, ok, "expected no summary metric, got %d", got)
				return
			}
			assert.True(t, ok, "expected summary metric to be present")
			assert.Equal(t, tt.want, got)
		})
	}
}
