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
	sub := NewSubstringPredicate("b")
	sub.KeepValue(parquet.ValueOf("abc"))
	require.NotEmpty(t, sub.matches)

	or := &OrPredicate{preds: []Predicate{sub}}
	or.resetForChunk()
	require.Empty(t, sub.matches, "reset must cascade to the wrapped predicate")
}
