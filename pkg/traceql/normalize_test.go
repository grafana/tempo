package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize_OR_Order(t *testing.T) {
	q1 := `{ span.foo = "bar" || span.bar = "foo" }`
	q2 := `{ span.bar = "foo" || span.foo = "bar" }`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.Equal(t, ast1.String(), ast2.String())
}

func TestNormalize_AND_Order(t *testing.T) {
	q1 := `{ span.foo = "bar" && span.bar = "foo" }`
	q2 := `{ span.bar = "foo" && span.foo = "bar" }`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.Equal(t, ast1.String(), ast2.String())
}

func TestNormalize_NestedLogical(t *testing.T) {
	q1 := `{ (span.a = 1 || span.b = 2) && span.c = 3 }`
	q2 := `{ span.c = 3 && (span.b = 2 || span.a = 1) }`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.Equal(t, ast1.String(), ast2.String())
}

func TestNormalize_DoesNotReorderComparisons(t *testing.T) {
	q := `{ span.duration > 5s }`

	ast, err := Parse(q)
	require.NoError(t, err)

	before := ast.String()
	Normalize(ast)
	after := ast.String()

	require.Equal(t, before, after)
}

func TestNormalize_Idempotent(t *testing.T) {
	q := `{ span.foo = "bar" || span.bar = "foo" }`

	ast, err := Parse(q)
	require.NoError(t, err)

	Normalize(ast)
	first := ast.String()

	Normalize(ast)
	second := ast.String()

	require.Equal(t, first, second)
}
