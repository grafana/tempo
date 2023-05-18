package vparquet2

import (
	"testing"

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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 2},
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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 6},
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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12},
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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12},
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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 12},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 11, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 10, ParentID: 2},
					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 6, NestedSetRight: 7, ParentID: 5},
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("dddddddd"), NestedSetLeft: 8, NestedSetRight: 9, ParentID: 5},

					{SpanID: []byte("gggggggg"), NestedSetLeft: 13, NestedSetRight: 18},
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
					{SpanID: []byte("aaaaaaaa"), NestedSetLeft: 1, NestedSetRight: 8},
					{SpanID: []byte("bbbbbbbb"), ParentSpanID: []byte("aaaaaaaa"), NestedSetLeft: 2, NestedSetRight: 7, ParentID: 1},
					{SpanID: []byte("cccccccc"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 3, NestedSetRight: 4, ParentID: 2},
					{SpanID: []byte("dddddddd"), ParentSpanID: []byte("bbbbbbbb"), NestedSetLeft: 5, NestedSetRight: 6, ParentID: 2},

					{SpanID: []byte("eeeeeeee"), ParentSpanID: []byte("xxxxxxxx")}, // <- interrupted
					{SpanID: []byte("ffffffff"), ParentSpanID: []byte("eeeeeeee")},
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

	actualSpans := map[[8]byte]*Span{}
	actualCount := 0
	for _, rs := range actual.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for i, s := range ss.Spans {
				actualSpans[util.SpanIDToArray(s.SpanID)] = &ss.Spans[i]
				actualCount++
			}
		}
	}

	expectedCount := 0
	for _, rs := range expected.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, exp := range ss.Spans {
				expectedCount++
				act, ok := actualSpans[util.SpanIDToArray(exp.SpanID)]
				require.Truef(t, ok, "span '%v' expected but was missing", string(exp.SpanID))
				assert.Equalf(t, exp.NestedSetLeft, act.NestedSetLeft, "span '%v' NestedSetLeft is expected %d but was %d", string(exp.SpanID), exp.NestedSetLeft, act.NestedSetLeft)
				assert.Equalf(t, exp.NestedSetRight, act.NestedSetRight, "span '%v' NestedSetRight is expected %d but was %d", string(exp.SpanID), exp.NestedSetRight, act.NestedSetRight)
				assert.Equalf(t, exp.ParentID, act.ParentID, "span '%v' ParentID is expected %d but was %d", string(exp.SpanID), exp.ParentID, act.ParentID)
				assert.Equalf(t, exp.ParentSpanID, act.ParentSpanID, "span '%v' ParentSpanID is expected %d but was %d", string(exp.SpanID), string(exp.ParentSpanID), string(act.ParentSpanID))
			}
		}
	}

	assert.Equalf(t, expectedCount, actualCount, "expected %d spans but found %d instead", expectedCount, actualCount)
}

func TestStack(t *testing.T) {
	type testVal struct {
		val int
	}

	var (
		testStack stack[testVal]
		val       *testVal
	)

	assert.Len(t, testStack, 0)
	assert.True(t, testStack.isEmpty(), "testStack expected to be empty")

	val = testStack.peek()
	assert.Nil(t, val, "testStack.peek() expected to be nil")
	val = testStack.pop()
	assert.Nil(t, val, "testStack.pop() expected to be nil")

	testStack.push(&testVal{1})
	val = testStack.peek()
	require.NotNil(t, val)
	assert.Equal(t, &testVal{1}, val)

	testStack.push(&testVal{2})
	val = testStack.peek()
	require.NotNil(t, val)
	assert.Equal(t, &testVal{2}, val)

	val = testStack.pop()
	require.NotNil(t, val)
	assert.Equal(t, &testVal{2}, val)

	val = testStack.pop()
	require.NotNil(t, val)
	assert.Equal(t, &testVal{1}, val)

	val = testStack.pop()
	assert.True(t, testStack.isEmpty(), "testStack expected to be empty")
	assert.Nil(t, val, "testStack.peek() expected to be nil")
	assert.Nil(t, val, "testStack.pop() expected to be nil")

	testStack.push(&testVal{1})
	testStack.reset()
	assert.True(t, testStack.isEmpty(), "testStack expected to be empty")
	assert.Len(t, testStack, 0)
	assert.Greater(t, cap(testStack), 0)
}
