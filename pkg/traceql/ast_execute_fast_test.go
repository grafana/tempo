package traceql

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBinaryOperationFastPath(t *testing.T) {
	span := newMockSpan([]byte{1}).
		WithDuration(uint64(2*time.Second)).
		WithSpanString("str", "foo").
		WithSpanInt("int", 200).
		WithSpanFloat("float", 200.5).
		WithSpanFloat("nan", math.NaN())
	span.attributes[NewScopedAttribute(AttributeScopeSpan, false, "bool")] = NewStaticBool(true)
	span.attributes[NewIntrinsic(IntrinsicName)] = NewStaticString("GET /api")
	span.attributes[NewIntrinsic(IntrinsicStatus)] = NewStaticStatus(StatusError)
	span.attributes[NewIntrinsic(IntrinsicKind)] = NewStaticKind(KindServer)
	span.attributes[NewIntrinsic(IntrinsicDuration)] = NewStaticDuration(2 * time.Second)
	span.attributes[NewScopedAttribute(AttributeScopeResource, false, "str")] = NewStaticString("bar")
	span.attributes[NewScopedAttribute(AttributeScopeResource, false, "service.name")] = NewStaticString("svc")
	span.attributes[NewScopedAttribute(AttributeScopeSpan, false, "strs")] = NewStaticStringArray([]string{"a", "foo"})
	span.attributes[NewScopedAttribute(AttributeScopeSpan, false, "ints")] = NewStaticIntArray([]int{1, 200})

	tests := []struct {
		query string
		want  bool
	}{
		{`{ span.str = "foo" }`, true},
		{`{ span.str != "foo" }`, false},
		{`{ span.str > "fo" }`, true},
		{`{ span.str <= "fo" }`, false},
		{`{ span.str =~ "fo" }`, false}, // regexes are anchored
		{`{ span.str =~ "fo.*" }`, true},
		{`{ span.str =~ "o" }`, false},
		{`{ span.str =~ ".*o" }`, true},
		{`{ span.str !~ "f.*" }`, false},
		{`{ span.str = 200 }`, false},
		{`{ span.int = 200 }`, true},
		{`{ span.int = 200.0 }`, true},
		{`{ span.int < 200.5 }`, true},
		{`{ span.int >= 201 }`, false},
		{`{ span.int = "200" }`, false},
		{`{ span.float > 200 }`, true},
		{`{ span.float = 200.5 }`, true},
		{`{ span.nan = 0 }`, false},
		{`{ span.nan != 0 }`, true},
		{`{ span.nan > 0 }`, false},
		{`{ span.bool = true }`, true},
		{`{ span.bool != true }`, false},
		{`{ span.bool }`, true},
		{`{ span.missing = "foo" }`, false},
		{`{ span.missing != "foo" }`, false},
		{`{ span.missing != 0 }`, false},
		{`{ span.missing = nil }`, false}, // compares with the string "nil", see rewriteNotExists
		{`{ span.str != nil }`, true},
		{`{ .str = "foo" }`, true}, // span scope wins over resource scope
		{`{ .str = "bar" }`, false},
		{`{ resource.str = "bar" }`, true},
		{`{ .service.name = "svc" }`, true},
		{`{ span.service.name = "svc" }`, false},
		{`{ name = "GET /api" }`, true},
		{`{ status = error }`, true},
		{`{ status != ok }`, true},
		{`{ kind = server }`, true},
		{`{ duration > 1s }`, true},
		{`{ duration > 1000000000 }`, true},
		{`{ duration = 2s }`, true},
		{`{ span.strs = "foo" }`, true},
		{`{ span.strs != "foo" }`, false},
		{`{ span.strs =~ "f.*" }`, true},
		{`{ span.ints > 100 }`, true},
		{`{ span.str = "foo" && span.int = 200 }`, true},
		{`{ span.str = "foo" && span.int = 201 }`, false},
		{`{ span.str = "x" || span.int = 200 }`, true},
		{`{ span.str = "x" || span.missing = 200 }`, false},
		{`{ (span.str = "x" || status = error) && span.str =~ "fo.*" }`, true},
		{`{ !(span.str = "foo") || span.int = 200 }`, true},
		{`{ span.str != nil && span.missing != nil }`, false},
		{`{ span.missing != nil || span.str != nil }`, true},
		{`{ span.bool && span.int > 100 }`, true},
		{`{ span.strs = "foo" && span.ints = 1 }`, true},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			fast := parseFilterExpression(t, tt.query)
			generic := parseFilterExpression(t, tt.query)
			disableFastPath(generic)

			for i := 0; i < 2; i++ { // the second pass runs with compiled regexes
				r, err := generic.execute(span)
				require.NoError(t, err)
				require.Equal(t, NewStaticBool(tt.want), r, "generic")

				r, err = fast.execute(span)
				require.NoError(t, err)
				require.Equal(t, NewStaticBool(tt.want), r, "fast")
			}
		})
	}
}

func TestBinaryOperationFastPathClassification(t *testing.T) {
	tests := []struct {
		query string
		want  fastPath
	}{
		{`{ span.foo = "bar" }`, fastPathCompare},
		{`{ span.foo =~ "bar" }`, fastPathCompare},
		{`{ duration > 1s }`, fastPathCompare},
		{`{ span.foo = "bar" && span.baz > 1 }`, fastPathLogic},
		{`{ span.foo = "bar" || span.baz != nil }`, fastPathLogic},
		{`{ !(span.foo = "bar") && span.baz = 1 }`, fastPathLogic},
		{`{ span.foo + 1 = 2 }`, fastPathNone},
		{`{ "bar" = span.foo }`, fastPathNone},
		{`{ span.foo = span.bar }`, fastPathNone},
		{`{ span.foo + 1 }`, fastPathNone},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			e := parseFilterExpression(t, tt.query)
			o, ok := e.(*BinaryOperation)
			require.True(t, ok)
			require.Equal(t, fastPathUnknown, o.fastPath)

			_, _ = e.execute(newMockSpan(nil))
			require.Equal(t, tt.want, o.fastPath)
		})
	}

	// arrays use the generic path
	o := newBinaryOperation(OpEqual, NewAttribute("foo"), NewStaticIntArray([]int{1})).(*BinaryOperation)
	o.classifyFastPath()
	require.Equal(t, fastPathNone, o.fastPath)
}

// TestBinaryOperationFastPathEquivalence evaluates randomly generated filters with and without the
// fast path over randomly generated spans and requires identical results, errors included.
func TestBinaryOperationFastPathEquivalence(t *testing.T) {
	const (
		numExpressions = 3000
		numSpans       = 200
	)

	rnd := rand.New(rand.NewSource(1))
	spans := make([]Span, numSpans)
	for i := range spans {
		spans[i] = randomFastPathSpan(rnd, i)
	}

	var classified, notClassified int
	for i := 0; i < numExpressions; i++ {
		seed := rnd.Int63()
		fast := randomFastPathExpression(rand.New(rand.NewSource(seed)), 3)
		generic := randomFastPathExpression(rand.New(rand.NewSource(seed)), 3)
		require.Equal(t, fast.String(), generic.String())

		// the branch optimizer may reorder && and || differently in each tree, which changes
		// which error is returned first
		disableBranchOptimizer(fast)
		disableBranchOptimizer(generic)
		disableFastPath(generic)

		for j, s := range spans {
			want, wantErr := generic.execute(s)
			got, gotErr := fast.execute(s)
			require.Equal(t, wantErr, gotErr, "expression %d %s span %d", i, generic, j)
			require.Equal(t, want, got, "expression %d %s span %d", i, generic, j)
		}

		// the same spans are kept when the expression is used as a filter
		input := []*Spanset{{Spans: spans}}
		want, wantErr := newSpansetFilter(generic).evaluate(input)
		got, gotErr := newSpansetFilter(fast).evaluate(input)
		require.Equal(t, wantErr, gotErr, "expression %d %s", i, generic)
		require.Equal(t, want, got, "expression %d %s", i, generic)

		walkBinaryOperations(fast, func(o *BinaryOperation) {
			if o.fastPath == fastPathLogic || o.fastPath == fastPathCompare {
				classified++
			} else {
				notClassified++
			}
		})
	}

	// sanity check that the generated expressions exercise both paths
	require.Greater(t, classified, numExpressions)
	require.Greater(t, notClassified, numExpressions/10)
}

// TestBinaryOperationFastPathBranchOptimizer checks the fast path through and after the branch
// optimizer's sampling phase, in which && and || take the generic path.
func TestBinaryOperationFastPathBranchOptimizer(t *testing.T) {
	queries := []string{
		`{ span.a = "foo" && resource.b > 1 }`,
		`{ (span.a =~ "f.*" || span.b != nil) && (resource.a = 1 || .b <= 2.5) }`,
		`{ (span.a != "bar" && status = error) || (kind = server && span.b = false) }`,
	}

	rnd := rand.New(rand.NewSource(2))
	spans := make([]Span, 2500) // more than the sampled executions
	for i := range spans {
		spans[i] = randomFastPathSpan(rnd, i)
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			fast := parseFilterExpression(t, q)
			generic := parseFilterExpression(t, q)
			disableFastPath(generic)

			for i, s := range spans {
				want, err := generic.execute(s)
				require.NoError(t, err)
				got, err := fast.execute(s)
				require.NoError(t, err)
				require.Equal(t, want, got, "span %d", i)
			}

			require.False(t, fast.(*BinaryOperation).b.Recording)
		})
	}
}

func BenchmarkBinOpFilter(b *testing.B) {
	queries := []string{
		`{ resource.service.name = "checkout" && name = "GET /cart" && span.cache.hit = false }`,
		`{ resource.service.name = "checkout" && kind = server && span.http.status_code = 500 }`,
		`{ (span.http.status_code != 200 || status != ok) && span.db.system =~ "postgres|mysql" }`,
		`{ duration > 100ms }`,
	}

	spans := make([]Span, 1024)
	for i := range spans {
		spans[i] = &sliceSpan{
			mockSpan: newMockSpan([]byte{1}),
			resourceAttrs: []sliceSpanAttr{
				{NewScopedAttribute(AttributeScopeResource, false, "service.name"), NewStaticString("checkout")},
				{NewScopedAttribute(AttributeScopeResource, false, "deployment.environment"), NewStaticString("prod")},
			},
			spanAttrs: []sliceSpanAttr{
				{NewIntrinsic(IntrinsicName), NewStaticString("GET /cart")},
				{NewIntrinsic(IntrinsicDuration), NewStaticDuration(time.Duration(i) * time.Millisecond)},
				{NewIntrinsic(IntrinsicKind), NewStaticKind(KindServer)},
				{NewIntrinsic(IntrinsicStatus), NewStaticStatus(StatusUnset)},
				{NewScopedAttribute(AttributeScopeSpan, false, "cache.hit"), NewStaticBool(i%3 == 0)},
				{NewScopedAttribute(AttributeScopeSpan, false, "db.system"), NewStaticString("postgres")},
				{NewScopedAttribute(AttributeScopeSpan, false, "http.status_code"), NewStaticInt(200 + i%2*300)},
			},
		}
	}

	for i, q := range queries {
		expr, err := Parse(q)
		require.NoError(b, err)
		pl, ok := expr.SinglePipeline()
		require.True(b, ok)
		f := pl.Elements[0].(*SpansetFilter)

		b.Run(fmt.Sprintf("q%d", i), func(b *testing.B) {
			for b.Loop() {
				for _, s := range spans {
					if _, err := f.Expression.execute(s); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

type sliceSpanAttr struct {
	a Attribute
	s Static
}

// sliceSpan stores attributes in per scope slices, similar to the vParquet span implementations.
type sliceSpan struct {
	*mockSpan
	resourceAttrs, spanAttrs []sliceSpanAttr
}

func (s *sliceSpan) AttributeFor(a Attribute) (Static, bool) {
	find := func(attrs []sliceSpanAttr) (Static, bool) {
		for i := range attrs {
			if attrs[i].a == a {
				return attrs[i].s, true
			}
		}
		return StaticNil, false
	}

	switch a.Scope {
	case AttributeScopeResource:
		return find(s.resourceAttrs)
	case AttributeScopeSpan:
		return find(s.spanAttrs)
	}
	return find(s.spanAttrs) // intrinsics
}

func parseFilterExpression(t *testing.T, query string) FieldExpression {
	expr, err := Parse(query)
	require.NoError(t, err)
	pl, ok := expr.SinglePipeline()
	require.True(t, ok)
	f, ok := pl.Elements[0].(*SpansetFilter)
	require.True(t, ok)
	return f.Expression
}

func walkBinaryOperations(e FieldExpression, fn func(*BinaryOperation)) {
	switch e := e.(type) {
	case *BinaryOperation:
		fn(e)
		walkBinaryOperations(e.LHS, fn)
		walkBinaryOperations(e.RHS, fn)
	case UnaryOperation:
		walkBinaryOperations(e.Expression, fn)
	}
}

func disableFastPath(e FieldExpression) {
	walkBinaryOperations(e, func(o *BinaryOperation) { o.fastPath = fastPathNone })
}

func disableBranchOptimizer(e FieldExpression) {
	walkBinaryOperations(e, func(o *BinaryOperation) { o.b.Recording = false })
}

var (
	fastPathTestNames = []string{"a", "b"}

	fastPathTestValues = []Static{
		NewStaticString("foo"),
		NewStaticString("bar"),
		NewStaticString(""),
		NewStaticString("200"),
		NewStaticString("nil"),
		NewStaticInt(0),
		NewStaticInt(1),
		NewStaticInt(-5),
		NewStaticInt(200),
		NewStaticFloat(0),
		NewStaticFloat(1),
		NewStaticFloat(200.5),
		NewStaticFloat(math.Copysign(0, -1)),
		NewStaticFloat(math.NaN()),
		NewStaticFloat(math.Inf(1)),
		NewStaticBool(true),
		NewStaticBool(false),
		NewStaticDuration(0),
		NewStaticDuration(time.Second),
		NewStaticStatus(StatusError),
		NewStaticStatus(StatusOk),
		NewStaticStatus(StatusUnset),
		NewStaticKind(KindServer),
		NewStaticKind(KindClient),
		NewStaticStringArray([]string{"foo", "bar"}),
		NewStaticStringArray([]string{}),
		NewStaticIntArray([]int{1, 200}),
		NewStaticFloatArray([]float64{1, math.NaN()}),
		NewStaticBooleanArray([]bool{false, true}),
		StaticNil,
	}

	fastPathTestRegexes = []string{"foo", "fo", "f.*", "^f", "o+$", ".*", "", "ba[rz]", "2.*", "(", "nil"}

	fastPathTestOps = []Operator{
		OpEqual, OpNotEqual, OpGreater, OpGreaterEqual, OpLess, OpLessEqual, OpRegex, OpNotRegex,
	}
)

func randomFastPathSpan(rnd *rand.Rand, i int) Span {
	s := newMockSpan([]byte{byte(i), byte(i >> 8)})
	value := func() Static { return fastPathTestValues[rnd.Intn(len(fastPathTestValues))] }

	for _, name := range fastPathTestNames {
		for _, scope := range []AttributeScope{AttributeScopeSpan, AttributeScopeResource} {
			if rnd.Intn(4) > 0 {
				s.attributes[NewScopedAttribute(scope, false, name)] = value()
			}
		}
	}
	for _, intrinsic := range []Intrinsic{IntrinsicName, IntrinsicStatus, IntrinsicKind, IntrinsicDuration} {
		if rnd.Intn(4) > 0 {
			s.attributes[NewIntrinsic(intrinsic)] = value()
		}
	}
	return s
}

func randomFastPathAttribute(rnd *rand.Rand) Attribute {
	name := fastPathTestNames[rnd.Intn(len(fastPathTestNames))]
	switch rnd.Intn(8) {
	case 0:
		return NewScopedAttribute(AttributeScopeResource, false, name)
	case 1:
		return NewAttribute(name)
	case 2:
		return NewScopedAttribute(AttributeScopeSpan, false, "missing")
	case 3:
		return NewIntrinsic([]Intrinsic{IntrinsicName, IntrinsicStatus, IntrinsicKind, IntrinsicDuration}[rnd.Intn(4)])
	default:
		return NewScopedAttribute(AttributeScopeSpan, false, name)
	}
}

func randomFastPathStatic(rnd *rand.Rand, op Operator) Static {
	if (op == OpRegex || op == OpNotRegex) && rnd.Intn(4) > 0 {
		return NewStaticString(fastPathTestRegexes[rnd.Intn(len(fastPathTestRegexes))])
	}
	return fastPathTestValues[rnd.Intn(len(fastPathTestValues))]
}

// randomFastPathExpression builds a random boolean expression from comparisons and && / ||. Most
// operands are comparisons of an attribute with a static, the shape the fast path handles.
func randomFastPathExpression(rnd *rand.Rand, depth int) FieldExpression {
	if depth > 0 && rnd.Intn(3) > 0 {
		op := OpAnd
		if rnd.Intn(2) == 0 {
			op = OpOr
		}
		return newBinaryOperation(op, randomFastPathExpression(rnd, depth-1), randomFastPathExpression(rnd, depth-1))
	}

	op := fastPathTestOps[rnd.Intn(len(fastPathTestOps))]
	attr := randomFastPathAttribute(rnd)
	static := randomFastPathStatic(rnd, op)

	switch rnd.Intn(12) {
	case 0:
		return newUnaryOperation(OpExists, attr)
	case 1:
		return newUnaryOperation(OpNotExists, attr)
	case 2:
		return newUnaryOperation(OpNot, newBinaryOperation(op, attr, static))
	case 3:
		return newBinaryOperation(op, static, attr) // static on the left
	case 4:
		return newBinaryOperation(op, newBinaryOperation(OpAdd, attr, NewStaticInt(1)), static)
	case 5:
		return attr // bare attribute, an implied `= true` when used with && or ||
	case 6:
		return newBinaryOperation([]Operator{OpIn, OpNotIn, OpRegexMatchAny, OpRegexMatchNone}[rnd.Intn(4)],
			attr, NewStaticStringArray([]string{"foo", "f.*"}))
	default:
		return newBinaryOperation(op, attr, static)
	}
}
