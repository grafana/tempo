package traceql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatic_Equals(t *testing.T) {
	areEqual := []struct {
		lhs, rhs Static
	}{
		{NewStaticInt(1), NewStaticInt(1)},
		{NewStaticFloat(1.5), NewStaticFloat(1.5)},
		{NewStaticInt(1), NewStaticFloat(1)},
		{NewStaticString("foo"), NewStaticString("foo")},
		{NewStaticBool(true), NewStaticBool(true)},
		{NewStaticDuration(1 * time.Second), NewStaticDuration(1000 * time.Millisecond)},
		{NewStaticStatus(StatusOk), NewStaticStatus(StatusOk)},
		{NewStaticKind(KindClient), NewStaticKind(KindClient)},
		{NewStaticDuration(0), NewStaticInt(0)},
		// Status and int comparison
		{NewStaticStatus(StatusError), NewStaticInt(0)},
		{NewStaticStatus(StatusOk), NewStaticInt(1)},
		{NewStaticStatus(StatusUnset), NewStaticInt(2)},
	}
	areNotEqual := []struct {
		lhs, rhs Static
	}{
		{NewStaticInt(1), NewStaticInt(2)},
		{NewStaticBool(true), NewStaticInt(1)},
		{NewStaticString("foo"), NewStaticString("bar")},
		{NewStaticKind(KindClient), NewStaticKind(KindConsumer)},
		{NewStaticStatus(StatusError), NewStaticStatus(StatusOk)},
		{NewStaticStatus(StatusOk), NewStaticInt(0)},
		{NewStaticStatus(StatusError), NewStaticFloat(0)},
	}
	for _, tt := range areEqual {
		t.Run(fmt.Sprintf("%v == %v", tt.lhs, tt.rhs), func(t *testing.T) {
			assert.True(t, tt.lhs.Equals(tt.rhs))
			assert.True(t, tt.rhs.Equals(tt.lhs))
		})
	}
	for _, tt := range areNotEqual {
		t.Run(fmt.Sprintf("%v != %v", tt.lhs, tt.rhs), func(t *testing.T) {
			assert.False(t, tt.lhs.Equals(tt.rhs))
			assert.False(t, tt.rhs.Equals(tt.lhs))
		})
	}
}

func TestPipelineExtractConditions(t *testing.T) {
	testCases := []struct {
		query   string
		request FetchSpansRequest
	}{
		{
			"{ .foo1 = `a` } | { .foo2 = `b` }",
			FetchSpansRequest{
				Conditions: []Condition{
					newCondition(NewAttribute("foo1"), OpEqual, NewStaticString("a")),
					newCondition(NewAttribute("foo2"), OpEqual, NewStaticString("b")),
				},
				AllConditions: false,
			},
		},
		{
			"{ .foo = `a` } | by(.namespace) | count() > 3",
			FetchSpansRequest{
				Conditions: []Condition{
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
					newCondition(NewAttribute("namespace"), OpNone),
				},
				AllConditions: false,
			},
		},
		{
			"{ .foo = `a` } | avg(duration) > 20ms",
			FetchSpansRequest{
				Conditions: []Condition{
					newCondition(NewAttribute("foo"), OpEqual, NewStaticString("a")),
					newCondition(NewIntrinsic(IntrinsicDuration), OpNone),
				},
				AllConditions: false,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			actualRequest := FetchSpansRequest{
				AllConditions: true,
			}
			ast.Pipeline.extractConditions(&actualRequest)
			require.Equal(t, tc.request, actualRequest)
		})
	}
}

func TestPipelineEvaluate(t *testing.T) {
	testCases := []struct {
		query  string
		input  []*Spanset
		output []*Spanset
	}{
		{
			"{ true } | { true } | { true }",
			[]*Spanset{
				{Spans: []Span{&mockSpan{}}},
			},
			[]*Spanset{
				{Spans: []Span{&mockSpan{}}},
			},
		},
		{
			"{ true } | { false } | { true }",
			[]*Spanset{
				{Spans: []Span{&mockSpan{}}},
			},
			[]*Spanset{},
		},
		{
			"{ .foo1 = `a` } | { .foo2 = `b` }",
			[]*Spanset{
				{Spans: []Span{
					// First span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo1"): NewStaticString("a")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo1"): NewStaticString("a"), NewAttribute("foo2"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo1"): NewStaticString("a"), NewAttribute("foo2"): NewStaticString("b")}},
				}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			actual, err := ast.Pipeline.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}

func TestSpansetFilterEvaluate(t *testing.T) {
	testCases := []struct {
		query  string
		input  []*Spanset
		output []*Spanset
	}{
		{
			"{ true }",
			[]*Spanset{
				// Empty spanset is dropped
				{Spans: []Span{}},
				{Spans: []Span{&mockSpan{}}},
			},
			[]*Spanset{
				{Spans: []Span{&mockSpan{}}},
			},
		},
		{
			"{ .foo = `a` }",
			[]*Spanset{
				{Spans: []Span{
					// Second span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
				{Spans: []Span{
					// This entire spanset will be dropped
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("b")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}},
				}},
			},
		},
		{
			"{ .http.status > `200` }",
			[]*Spanset{
				{Spans: []Span{
					// First span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("200")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("201")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("300")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("301")}},
				}},
				{Spans: []Span{
					// This entire spanset will be dropped
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("100")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("201")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("300")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("301")}},
				}},
			},
		},
		{
			"{ .http.status <= `300` }",
			[]*Spanset{
				{Spans: []Span{
					// Last span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("200")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("201")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("300")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("301")}},
				}},
				{Spans: []Span{
					// This entire spanset is valid
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("100")}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("200")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("201")}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("300")}},
				}},
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticString("100")}},
				}},
			},
		},
		{
			"{ .http.status > `200` }",
			[]*Spanset{
				{Spans: []Span{
					// This entire spanset will be dropped because mismatch type
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticInt(200)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticInt(201)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticInt(300)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticInt(301)}},
				}},
				{Spans: []Span{
					// This entire spanset will be dropped because mismatch type
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("http.status"): NewStaticInt(100)}},
				}},
			},
			nil,
		},
		{
			"{ .foo = 1 || (.foo >= 4 && .foo < 6) }",
			[]*Spanset{
				{Spans: []Span{
					// Second span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(2)}},
				}},
				{Spans: []Span{
					// First span should be dropped here
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(3)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(4)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(5)}},
				}},
				{Spans: []Span{
					// Entire spanset should be dropped
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(6)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(7)}},
				}},
			},
			[]*Spanset{
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}},
				}},
				{Spans: []Span{
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(4)}},
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(5)}},
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			ast, err := Parse(tc.query)
			require.NoError(t, err)

			filt := ast.Pipeline.Elements[0].(*SpansetFilter)

			actual, err := filt.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}

func TestStaticCompare(t *testing.T) {
	testCases := []struct {
		name     string
		s1       Static
		s2       Static
		expected int
	}{
		{
			name:     "IntComparison_Greater",
			s1:       Static{Type: TypeInt, N: 10},
			s2:       Static{Type: TypeInt, N: 5},
			expected: 1,
		},
		{
			name:     "FloatComparison_Greater",
			s1:       Static{Type: TypeFloat, F: 10.5},
			s2:       Static{Type: TypeFloat, F: 5.5},
			expected: 1,
		},
		{
			name:     "StringComparison_Less",
			s1:       Static{Type: TypeString, S: "hello"},
			s2:       Static{Type: TypeString, S: "world"},
			expected: -1,
		},
		{
			name:     "BooleanComparison_Greater",
			s1:       Static{Type: TypeBoolean, B: true},
			s2:       Static{Type: TypeBoolean, B: false},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.s1.compare(&tc.s2)
			require.Equal(t, tc.expected, result)
		})
	}
}

var _ Span = (*mockSpan)(nil)

type mockSpan struct {
	id                 []byte
	startTimeUnixNanos uint64
	durationNanos      uint64
	attributes         map[Attribute]Static

	parentID, left, right int
}

func newMockSpan(id []byte) *mockSpan {
	return &mockSpan{
		id:         id,
		attributes: map[Attribute]Static{},
	}
}

func (m *mockSpan) WithNestedSetInfo(parentid, left, right int) *mockSpan {
	m.parentID = parentid
	m.left = left
	m.right = right
	return m
}

func (m *mockSpan) WithAttrBool(key string, value bool) *mockSpan {
	m.attributes[NewAttribute(key)] = NewStaticBool(value)
	return m
}

func (m *mockSpan) Attributes() map[Attribute]Static {
	return m.attributes
}

func (m *mockSpan) ID() []byte {
	return m.id
}

func (m *mockSpan) StartTimeUnixNanos() uint64 {
	return m.startTimeUnixNanos
}

func (m *mockSpan) DurationNanos() uint64 {
	return m.durationNanos
}

func (m *mockSpan) DescendantOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, buffer []Span) []Span { // jpe - fix
	return loop(lhs, rhs, falseForAll, invert, descendantOf)
}

func descendantOf(s1 Span, s2 Span) bool {
	return s2.(*mockSpan).left > s1.(*mockSpan).left && s2.(*mockSpan).left < s1.(*mockSpan).right
}

func (m *mockSpan) SiblingOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, buffer []Span) []Span {
	return loop(lhs, rhs, falseForAll, invert, siblingOf)
}

func siblingOf(s1 Span, s2 Span) bool {
	return s1.(*mockSpan).parentID == s2.(*mockSpan).parentID
}

func (m *mockSpan) ChildOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, buffer []Span) []Span {
	return loop(lhs, rhs, falseForAll, invert, childOf)
}

func childOf(s1 Span, s2 Span) bool {
	return s1.(*mockSpan).parentID == s2.(*mockSpan).left
}

func loop(lhs []Span, rhs []Span, falseForAll bool, invert bool, eval func(s1 Span, s2 Span) bool) []Span {
	out := []Span{}

	for _, l := range lhs {
		match := false
		for _, r := range rhs {
			if invert {
				r, l = l, r
			}

			if eval(l, r) {
				match = true
				break
			}
		}

		if (match && !falseForAll) ||
			(!match && falseForAll) {
			out = append(out, l)
		}
	}

	return out
}
