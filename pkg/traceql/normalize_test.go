package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize_CommutativeLogical(t *testing.T) {
	tests := []struct {
		name string
		q1   string
		q2   string
	}{
		{
			name: "OR simple",
			q1:   `{ span.foo = "bar" || span.bar = "foo" }`,
			q2:   `{ span.bar = "foo" || span.foo = "bar" }`,
		},
		{
			name: "AND simple",
			q1:   `{ span.foo = "bar" && span.bar = "foo" }`,
			q2:   `{ span.bar = "foo" && span.foo = "bar" }`,
		},
		{
			name: "nested OR",
			q1:   `{ (span.a = 1 || span.b = 2) || span.c = 3 }`,
			q2:   `{ span.c = 3 || (span.b = 2 || span.a = 1) }`,
		},
		{
			name: "mixed AND OR",
			q1:   `{ (span.a = 1 || span.b = 2) && span.c = 3 }`,
			q2:   `{ span.c = 3 && (span.b = 2 || span.a = 1) }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast1, err := Parse(tt.q1)
			require.NoError(t, err)

			ast2, err := Parse(tt.q2)
			require.NoError(t, err)

			Normalize(ast1)
			Normalize(ast2)

			require.Equal(t, ast1.String(), ast2.String())
		})
	}
}

func TestNormalize_Idempotent(t *testing.T) {
	q := `{ span.a = 1 || span.b = 2 }`

	ast, err := Parse(q)
	require.NoError(t, err)

	Normalize(ast)
	first := ast.String()

	Normalize(ast)
	second := ast.String()

	require.Equal(t, first, second)
}

func TestNormalize_DoesNotChangeNonCommutative(t *testing.T) {
	q := `{ span.duration > 5s }`

	ast, err := Parse(q)
	require.NoError(t, err)

	before := ast.String()

	Normalize(ast)

	after := ast.String()

	require.Equal(t, before, after)
}

func TestNormalize_Spanset_AND_Order(t *testing.T) {
	q1 := `({ span.a }) && ({ span.b })`
	q2 := `({ span.b }) && ({ span.a })`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.Equal(t, ast1.String(), ast2.String())
}

func TestNormalize_Spanset_OR_Order(t *testing.T) {
	q1 := `({ span.a }) || ({ span.b })`
	q2 := `({ span.b }) || ({ span.a })`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.Equal(t, ast1.String(), ast2.String())
}

func TestNormalize_DifferentExpressionsStayDifferent(t *testing.T) {
	q1 := `{ span.a = 1 || span.b = 2 }`
	q2 := `{ span.a = 1 && span.b = 2 }`

	ast1, err := Parse(q1)
	require.NoError(t, err)

	ast2, err := Parse(q2)
	require.NoError(t, err)

	Normalize(ast1)
	Normalize(ast2)

	require.NotEqual(t, ast1.String(), ast2.String())
}
