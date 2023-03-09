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
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo1"): NewStaticString("a"), NewAttribute("foo2"): NewStaticString("b")}}}},
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
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticString("a")}}}},
			},
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
					&mockSpan{attributes: map[Attribute]Static{NewAttribute("foo"): NewStaticInt(1)}}}},
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

			filt := ast.Pipeline.Elements[0].(SpansetFilter)

			actual, err := filt.evaluate(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.output, actual)
		})
	}
}

var _ Span = (*mockSpan)(nil)

type mockSpan struct {
	id                 []byte
	startTimeUnixNanos uint64
	endTimeUnixNanos   uint64
	attributes         map[Attribute]Static

	wasReleased bool
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
func (m *mockSpan) EndtimeUnixNanos() uint64 {
	return m.endTimeUnixNanos
}
func (m *mockSpan) Release() {
	m.wasReleased = true
}
