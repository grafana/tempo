package parquetquery

import (
	"context"
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

// TestResetBeforeDictionaryScan: the reset must precede the chunk's dictionary scan,
// or it wipes the matches that scan memoized before the rows can use them.
func TestResetBeforeDictionaryScan(t *testing.T) {
	r := buildFile(t, func(w *parquet.Writer) { //nolint:all
		require.NoError(t, w.Write(&testDictString{"abc"}))
		require.NoError(t, w.Write(&testDictString{"bcd"}))
	})

	p := &recordingPredicate{}
	it := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptPredicate(p))
	defer it.Close()
	drain(t, it)

	require.NotEmpty(t, p.events)
	require.Equal(t, "reset", p.events[0])
	require.Equal(t, 1, countOf(p.events, "reset"), "one chunk, one reset")
}

type recordingPredicate struct{ events []string }

func (p *recordingPredicate) String() string { return "recordingPredicate{}" }

func (p *recordingPredicate) KeepValue(parquet.Value) bool {
	p.events = append(p.events, "value")
	return true
}
func (p *recordingPredicate) KeepRange(parquet.Value, parquet.Value) bool { return true }
func (p *recordingPredicate) SupportsColumnIndexPruning() bool            { return false }
func (p *recordingPredicate) resetForChunk()                              { p.events = append(p.events, "reset") }

func countOf(ss []string, s string) int {
	n := 0
	for _, v := range ss {
		if v == s {
			n++
		}
	}
	return n
}
