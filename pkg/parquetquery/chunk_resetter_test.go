package parquetquery

import (
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

// TestSubstringResetForChunk: resetForChunk clears the memoization populated by KeepValue.
func TestSubstringResetForChunk(t *testing.T) {
	sub := NewSubstringPredicate("b")
	sub.KeepValue(parquet.ValueOf("abc"))
	require.NotEmpty(t, sub.matches)

	sub.resetForChunk()
	require.Empty(t, sub.matches)
}

// TestResetForChunkCascades: a memoizing predicate wrapped in a composite must still be
// reset, matching base where the reset rode along KeepColumnChunk through the tree.
func TestResetForChunkCascades(t *testing.T) {
	cases := []struct {
		name string
		wrap func(*SubstringPredicate) chunkResetter
	}{
		{"or", func(s *SubstringPredicate) chunkResetter { return &OrPredicate{preds: []Predicate{s}} }},
		{"instrumented", func(s *SubstringPredicate) chunkResetter { return &InstrumentedPredicate{Pred: s} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := NewSubstringPredicate("b")
			sub.KeepValue(parquet.ValueOf("abc"))
			require.NotEmpty(t, sub.matches)

			tc.wrap(sub).resetForChunk()
			require.Empty(t, sub.matches, "reset must cascade to the wrapped predicate")
		})
	}
}
