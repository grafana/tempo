package traceql

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStaticNil(t *testing.T) {
	s := NewStaticNil()
	assert.Equal(t, TypeNil, s.Type)
	assert.Equal(t, Static{}, s)
}

func TestStatic_Int(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: -math.MaxInt, ok: true},
		{arg: -1, ok: true},
		{arg: 0, ok: true},
		{arg: 1, ok: true},
		{arg: math.MaxInt, ok: true},
		// unsupported values
		{arg: "test"},
		{arg: 3.14},
		{arg: true},
		{arg: StatusOk},
		{arg: KindClient},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			i, ok := static.Int()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, i)
			}
		})
	}
}

func TestStatic_Float(t *testing.T) {
	tests := []struct {
		arg  any
		want float64
	}{
		{arg: -math.MaxFloat64},
		{arg: -1.0},
		{arg: 0.0},
		{arg: 10.0 + math.SmallestNonzeroFloat64},
		{arg: 1.0},
		{arg: math.MaxFloat64},
		{arg: int(3117), want: 3117.0},
		{arg: time.Duration(101), want: 101.0},
		{arg: "test", want: math.NaN()},
		{arg: true, want: math.NaN()},
		{arg: StatusError, want: math.NaN()},
		{arg: KindServer, want: math.NaN()},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			f := static.Float()

			if static.Type == TypeFloat {
				assert.Equal(t, tt.arg, f)
			} else if math.IsNaN(tt.want) {
				assert.True(t, math.IsNaN(f))
			} else {
				assert.Equal(t, tt.want, f)
			}
		})
	}
}

func TestStatic_String(t *testing.T) {
	tests := []struct {
		arg  any
		want string
	}{
		{arg: nil, want: "nil"},
		{arg: 101, want: "101"},
		{arg: -10, want: "-10"},
		{arg: -1.0, want: "-1"},
		{arg: 0.0, want: "0"},
		{arg: 10.0, want: "10"},
		{arg: "test", want: "`test`"},
		{arg: true, want: "true"},
		{arg: StatusOk, want: "ok"},
		{arg: KindClient, want: "client"},
		{arg: time.Duration(70) * time.Second, want: "1m10s"},
		{arg: []int{1, 2, 3}, want: "[1 2 3]"},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			assert.Equal(t, tt.want, static.String())
		})
	}
}

func TestStatic_Bool(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: false, ok: true},
		{arg: true, ok: true},
		// unsupported values
		{arg: 3.14},
		{arg: "test"},
		{arg: time.Duration(1)},
		{arg: StatusOk},
		{arg: KindClient},
		{arg: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			b, ok := static.Bool()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, b)
			}
		})
	}
}

func TestStatic_Duration(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: time.Duration(0), ok: true},
		{arg: time.Duration(1), ok: true},
		{arg: time.Duration(100) * time.Second, ok: true},
		// unsupported values
		{arg: 1},
		{arg: 3.14},
		{arg: "test"},
		{arg: true},
		{arg: StatusOk},
		{arg: KindClient},
		{arg: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			d, ok := static.Duration()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, d)
			}
		})
	}
}

func TestStatic_Status(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: StatusError, ok: true},
		{arg: StatusOk, ok: true},
		{arg: StatusUnset, ok: true},
		// unsupported values
		{arg: 1},
		{arg: 3.14},
		{arg: "test"},
		{arg: true},
		{arg: KindClient},
		{arg: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			s, ok := static.Status()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, s)
			}
		})
	}
}

func TestStatic_Kind(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: KindUnspecified, ok: true},
		{arg: KindInternal, ok: true},
		{arg: KindClient, ok: true},
		{arg: KindServer, ok: true},
		{arg: KindProducer, ok: true},
		{arg: KindConsumer, ok: true},
		// unsupported values
		{arg: 1},
		{arg: 3.14},
		{arg: "test"},
		{arg: true},
		{arg: StatusOk},
		{arg: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			k, ok := static.Kind()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, k)
			}
		})
	}
}

func TestStatic_IntArray(t *testing.T) {
	tests := []struct {
		arg any
		ok  bool
	}{
		// supported values
		{arg: []int(nil), ok: true},
		{arg: []int{}, ok: true},
		{arg: []int{1}, ok: true},
		{arg: []int{1, 2, 3, 4, 5}, ok: true},
		// unsupported values
		{arg: 1},
		{arg: 3.14},
		{arg: "test"},
		{arg: true},
		{arg: StatusOk},
		{arg: KindClient},
	}

	for _, tt := range tests {
		t.Run(testName(tt.arg), func(t *testing.T) {
			static := newStatic(tt.arg)
			a, ok := static.IntArray()

			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.arg, a)
			}
		})
	}
}

func TestStatic_Equals(t *testing.T) {
	areEqual := []struct {
		lhs, rhs Static
	}{
		{NewStaticInt(1), NewStaticInt(1)},
		{NewStaticFloat(1.5), NewStaticFloat(1.5)},
		{NewStaticInt(2), NewStaticFloat(2.0)},
		{NewStaticString("foo"), NewStaticString("foo")},
		{NewStaticBool(true), NewStaticBool(true)},
		{NewStaticDuration(1 * time.Second), NewStaticDuration(1000 * time.Millisecond)},
		{NewStaticStatus(StatusOk), NewStaticStatus(StatusOk)},
		{NewStaticKind(KindClient), NewStaticKind(KindClient)},
		{NewStaticDuration(0), NewStaticInt(0)},
		{NewStaticIntArray([]int{}), NewStaticIntArray(nil)},
		{NewStaticIntArray([]int{11, 111}), NewStaticIntArray([]int{11, 111})},
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
		{NewStaticStatus(StatusOk), NewStaticStatus(StatusUnset)},
		{NewStaticStatus(StatusOk), NewStaticKind(KindInternal)},
		{NewStaticStatus(StatusError), NewStaticFloat(0)},
		{NewStaticIntArray([]int{}), NewStaticIntArray([]int{0})},
		{NewStaticIntArray([]int{111, 11}), NewStaticIntArray([]int{11, 111})},
	}
	for _, tt := range areEqual {
		t.Run(fmt.Sprintf("%s==%s", testName(tt.rhs), testName(tt.rhs)), func(t *testing.T) {
			assert.True(t, tt.lhs.Equals(&tt.rhs))
			assert.True(t, tt.rhs.Equals(&tt.lhs))
		})
	}
	for _, tt := range areNotEqual {
		t.Run(fmt.Sprintf("%s!=%s", testName(tt.lhs), testName(tt.rhs)), func(t *testing.T) {
			assert.False(t, tt.lhs.Equals(&tt.rhs))
			assert.False(t, tt.rhs.Equals(&tt.lhs))
		})
	}
}

func TestStatic_compare(t *testing.T) {
	testCases := []struct {
		s1, s2 Static
		want   int
	}{
		{s1: NewStaticInt(10), s2: NewStaticInt(5), want: 1},
		{s1: NewStaticInt(5), s2: NewStaticInt(-10), want: 1},
		{s1: NewStaticInt(20), s2: NewStaticInt(20), want: 0},
		{s1: NewStaticFloat(10.5), s2: NewStaticFloat(5.5), want: 1},
		{s1: NewStaticFloat(100.0), s2: NewStaticInt(100), want: 0},
		{s1: NewStaticFloat(100.0), s2: NewStaticInt(50), want: 1},
		{s1: NewStaticString("world"), s2: NewStaticString("hello"), want: 1},
		{s1: NewStaticBool(true), s2: NewStaticBool(false), want: 1},
	}

	for _, tt := range testCases {
		t.Run(fmt.Sprintf("%s<>%s", tt.s1.String(), tt.s2.String()), func(t *testing.T) {
			res := tt.s1.compare(&tt.s2)
			require.Equal(t, tt.want, res, "s1.compare(s2)")
			res = tt.s2.compare(&tt.s1)
			require.Equal(t, -tt.want, res, "s2.compare(s1)")
		})
	}
}

func TestStatic_sumInto(t *testing.T) {
	tests := []struct {
		s1, s2, want Static
	}{
		{NewStaticInt(1), NewStaticInt(2), NewStaticInt(3)},
		{NewStaticInt(-3), NewStaticInt(2), NewStaticInt(-1)},
		{NewStaticInt(-3), NewStaticDuration(3), NewStaticInt(-3)},
		{NewStaticDuration(2 * time.Second), NewStaticDuration(1 * time.Second), NewStaticDuration(3 * time.Second)},
		{NewStaticDuration(2 * time.Second), NewStaticInt(3000), NewStaticDuration(2 * time.Second)},
		{NewStaticFloat(1.5), NewStaticFloat(2.5), NewStaticFloat(4.0)},
		{NewStaticFloat(-4.5), NewStaticFloat(2.0), NewStaticFloat(-2.5)},
		{NewStaticFloat(3.14), NewStaticInt(1), NewStaticFloat(3.14)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s+%s", tt.s1.String(), tt.s2.String()), func(t *testing.T) {
			tt.s1.sumInto(&tt.s2)
			assert.Equal(t, tt.want, tt.s1, "s1.sumInto(s2)")
		})
	}
}

func TestStatic_divideBy(t *testing.T) {
	tests := []struct {
		s    Static
		f    float64
		want Static
	}{
		{NewStaticInt(10), 2, NewStaticFloat(5)},
		{NewStaticDuration(12 * time.Second), 2.1, NewStaticDuration(6 * time.Second)},
		{NewStaticFloat(12.2), 2.0, NewStaticFloat(6.1)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%.2f", tt.s.String(), tt.f), func(t *testing.T) {
			s := tt.s.divideBy(tt.f)
			assert.Equal(t, tt.want, s, "s.divideBy(f)")
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

func (m *mockSpan) WithStartTime(nanos uint64) *mockSpan {
	m.startTimeUnixNanos = nanos
	return m
}

func (m *mockSpan) WithDuration(nanos uint64) *mockSpan {
	m.durationNanos = nanos
	return m
}

func (m *mockSpan) WithNestedSetInfo(parentid, left, right int) *mockSpan {
	m.parentID = parentid
	m.left = left
	m.right = right
	return m
}

func (m *mockSpan) WithSpanString(key string, value string) *mockSpan {
	m.attributes[NewScopedAttribute(AttributeScopeSpan, false, key)] = NewStaticString(value)
	return m
}

func (m *mockSpan) WithAttrBool(key string, value bool) *mockSpan {
	m.attributes[NewAttribute(key)] = NewStaticBool(value)
	return m
}

func (m *mockSpan) AttributeFor(a Attribute) (Static, bool) {
	s, ok := m.attributes[a]
	// if not found explicitly, check if it's a span attribute
	if !ok && a.Scope == AttributeScopeNone {
		aSpan := a
		aSpan.Scope = AttributeScopeSpan
		s, ok = m.attributes[aSpan]
	}
	// if not found explicitly, check if it's a resource attribute
	if !ok && a.Scope == AttributeScopeNone {
		aRes := a
		aRes.Scope = AttributeScopeResource
		s, ok = m.attributes[aRes]
	}
	return s, ok
}

func (m *mockSpan) AllAttributes() map[Attribute]Static {
	return m.attributes
}

func (m *mockSpan) AllAttributesFunc(cb func(Attribute, Static)) {
	for k, v := range m.attributes {
		cb(k, v)
	}
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

func (m *mockSpan) DescendantOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, _ bool, _ []Span) []Span {
	return loop(lhs, rhs, falseForAll, invert, descendantOf)
}

func descendantOf(s1 Span, s2 Span) bool {
	return s2.(*mockSpan).left > s1.(*mockSpan).left && s2.(*mockSpan).left < s1.(*mockSpan).right
}

func (m *mockSpan) SiblingOf(lhs []Span, rhs []Span, falseForAll bool, _ bool, _ []Span) []Span {
	return loop(lhs, rhs, falseForAll, false, siblingOf)
}

func siblingOf(s1 Span, s2 Span) bool {
	return s1.(*mockSpan).parentID == s2.(*mockSpan).parentID
}

func (m *mockSpan) ChildOf(lhs []Span, rhs []Span, falseForAll bool, invert bool, _ bool, _ []Span) []Span {
	return loop(lhs, rhs, falseForAll, invert, childOf)
}

func childOf(s1 Span, s2 Span) bool {
	return s1.(*mockSpan).left == s2.(*mockSpan).parentID
}

func loop(lhs []Span, rhs []Span, falseForAll bool, invert bool, eval func(s1 Span, s2 Span) bool) []Span {
	out := []Span{}

	for _, r := range rhs {
		match := false
		for _, l := range lhs {
			if invert {
				match = eval(r, l)
			} else {
				match = eval(l, r)
			}

			if match {
				break
			}
		}

		if (match && !falseForAll) ||
			(!match && falseForAll) {
			out = append(out, r)
		}
	}

	return out
}

func newStatic(val any) Static {
	if val == nil {
		return NewStaticNil()
	}

	switch v := val.(type) {
	case int:
		return NewStaticInt(v)
	case float64:
		return NewStaticFloat(v)
	case string:
		return NewStaticString(v)
	case bool:
		return NewStaticBool(v)
	case time.Duration:
		return NewStaticDuration(v)
	case Status:
		return NewStaticStatus(v)
	case Kind:
		return NewStaticKind(v)
	case []int:
		return NewStaticIntArray(v)
	}
	panic(fmt.Sprintf("unsupported type %T", val))
}

func testName(val any) string {
	if val == nil {
		return "nil"
	}

	switch v := val.(type) {
	case float64:
		return fmt.Sprintf("%e", v)
	case []int:
		return fmt.Sprintf("[%d]int", len(v))
	case Static:
		return v.EncodeToString(false)
	default:
		return fmt.Sprintf("%v", v)
	}
}
