package vparquet2

import (
	"testing"

	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"

	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignNestedSetModelBounds(t *testing.T) {
	tests := []struct {
		name     string
		trace    [][]Span
		expected [][]Span
	}{
		{
			name: "single span",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 2, ParentID: -1},
				},
			},
		},
		{
			name: "linear trace",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 6, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 5, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
				},
			},
		},
		{
			name: "branched trace",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd")},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 11, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 10, ParentID: 2},
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 6, NestedSetRight: 7, ParentID: 5},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 8, NestedSetRight: 9, ParentID: 5},
				},
			},
		},
		{
			name: "multiple scope spans",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb")},
				},
				{
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd")},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 11, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 10, ParentID: 2},
				},
				{
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 6, NestedSetRight: 7, ParentID: 5},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 8, NestedSetRight: 9, ParentID: 5},
				},
			},
		},
		{
			name: "multiple roots",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd")},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd")},

					{SpanID: []byte("gggggggg")},
					{SpanID: []byte("iiiiiiii"), ParentSpanID: []byte("hhhhhhhh")},
					{SpanID: []byte("hhhhhhhh"), ParentSpanID: []byte("gggggggg")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 11, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 10, ParentID: 2},
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 6, NestedSetRight: 7, ParentID: 5},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 8, NestedSetRight: 9, ParentID: 5},

					{SpanID: []byte("gggggggg"), NestedSetLeft: 13, NestedSetRight: 18, ParentID: -1},
					{SpanID: []byte("hhhhhhhh"), ParentSpanID: []byte("gggggggg"), NestedSetLeft: 14, NestedSetRight: 17, ParentID: 13},
					{SpanID: []byte("iiiiiiii"), ParentSpanID: []byte("hhhhhhhh"), NestedSetLeft: 15, NestedSetRight: 16, ParentID: 14},
				},
			},
		},
		{
			name: "interrupted",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb")},

					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("xxxxxxxx")}, // <- interrupted
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("eeeeeeee")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 8, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 7, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 6, ParentID: 2},

					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("xxxxxxxx")}, // <- interrupted
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("eeeeeeee")},
				},
			},
		},
		{
			name: "partially assigned",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 4},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 0, ParentID: 1},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 4, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 3, ParentID: 1},
				},
			},
		},
		{
			name: "non unique IDs",
			trace: [][]Span{
				{
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), Kind: int(v1.Span_SPAN_KIND_CLIENT)},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_SERVER)},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb")},
					{SpanID: []byte("aaaaaaaa")},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 10, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_SERVER), NestedSetLeft: 3, NestedSetRight: 8, ParentID: 2},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), Kind: int(v1.Span_SPAN_KIND_CLIENT), NestedSetLeft: 2, NestedSetRight: 9, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 4, NestedSetRight: 5, ParentID: 3},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 6, NestedSetRight: 7, ParentID: 3},
				},
			},
		},
		{
			name: "non unique IDs 2x",
			trace: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa")},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), Kind: int(v1.Span_SPAN_KIND_CLIENT)},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_SERVER)},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("cccccccc"), Kind: int(v1.Span_SPAN_KIND_SERVER)},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_CLIENT)},
				},
			},
			expected: [][]Span{
				{
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 10, ParentID: -1},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), Kind: int(v1.Span_SPAN_KIND_CLIENT), NestedSetLeft: 2, NestedSetRight: 9, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_CLIENT), NestedSetLeft: 4, NestedSetRight: 7, ParentID: 3},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("bbbbbbbb"), Kind: int(v1.Span_SPAN_KIND_SERVER), NestedSetLeft: 3, NestedSetRight: 8, ParentID: 2},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("cccccccc"), Kind: int(v1.Span_SPAN_KIND_SERVER), NestedSetLeft: 5, NestedSetRight: 6, ParentID: 4},
				},
			},
		},
	}

	makeTrace := func(traceSpans [][]Span) *Trace {
		var resourceSpans ResourceSpans
		for _, spans := range traceSpans {
			scopeSpans := ScopeSpans{Spans: append([]Span{}, spans...)}
			resourceSpans.ScopeSpans = append(resourceSpans.ScopeSpans, scopeSpans)
		}
		return &Trace{ResourceSpans: []ResourceSpans{resourceSpans}}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := makeTrace(tt.trace)
			expected := makeTrace(tt.expected)
			assignNestedSetModelBounds(trace)
			assertEqualNestedSetModelBounds(t, trace, expected)
		})
	}
}

func assertEqualNestedSetModelBounds(t testing.TB, actual, expected *Trace) {
	t.Helper()

	actualSpans := map[uint64]*Span{}
	actualCount := 0
	for _, rs := range actual.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for i, s := range ss.Spans {
				actualSpans[util.SpanIDAndKindToToken(s.SpanID, s.Kind)] = &ss.Spans[i]
				actualCount++
			}
		}
	}

	expectedCount := 0
	for _, rs := range expected.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, exp := range ss.Spans {
				expectedCount++
				act, ok := actualSpans[util.SpanIDAndKindToToken(exp.SpanID, exp.Kind)]
				require.Truef(t, ok, "span '%v' expected but was missing", string(exp.SpanID))
				assert.Equalf(t, exp.NestedSetLeft, act.NestedSetLeft, "span '%v' NestedSetLeft is expected %d but was %d", string(exp.SpanID), exp.NestedSetLeft, act.NestedSetLeft)
				assert.Equalf(t, exp.NestedSetRight, act.NestedSetRight, "span '%v' NestedSetRight is expected %d but was %d", string(exp.SpanID), exp.NestedSetRight, act.NestedSetRight)
				assert.Equalf(t, exp.ParentID, act.ParentID, "span '%v' ParentID is expected %d but was %d", string(exp.SpanID), exp.ParentID, act.ParentID)
				assert.Equalf(t, exp.ParentSpanID, act.ParentSpanID, "span '%v' ParentSpanID is expected %d but was %d", string(exp.SpanID), string(exp.ParentSpanID), string(act.ParentSpanID))
				assert.Equalf(t, exp.Kind, act.Kind, "span '%v' Kind is expected %d but was %d", string(exp.SpanID), exp.Kind, act.Kind)
			}
		}
	}

	assert.Equalf(t, expectedCount, actualCount, "expected %d spans but found %d instead", expectedCount, actualCount)
}
