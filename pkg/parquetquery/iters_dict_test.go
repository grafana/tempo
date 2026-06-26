package parquetquery

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// collectStringResults drains an iterator returning (rowNumber, value-as-string)
// pairs so that the dictionary fast path and the per-row slow path can be compared
// for exact equivalence.
func collectStringResults(t *testing.T, iter Iterator, selectAs string) []string {
	t.Helper()
	var out []string
	for {
		res, err := iter.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
		vals := res.ToMap()[selectAs]
		require.Len(t, vals, 1)
		out = append(out, fmt.Sprintf("%v=%s", res.RowNumber, vals[0].String()))
	}
	return out
}

func runDictEquivalence[T any](t *testing.T, rows []T, colPath string, makePred func() Predicate) {
	ctx := context.Background()
	pf := createFileWith(t, ctx, rows)
	idx, _, _ := GetColumnIndexByPath(pf, colPath)
	require.GreaterOrEqual(t, idx, 0)

	newIter := func(disable bool) *SyncIterator {
		// Fresh predicate per iterator: some predicates (e.g. substring) memoize.
		opts := []SyncIteratorOpt{
			SyncIteratorOptSelectAs("S"),
			SyncIteratorOptPredicate(makePred()),
			SyncIteratorOptMaxDefinitionLevel(MaxDefinitionLevel),
		}
		if disable {
			opts = append(opts, SyncIteratorOptDisableDictPushdown())
		}
		return NewSyncIterator(ctx, pf.RowGroups(), idx, opts...)
	}

	fast := newIter(false)
	defer fast.Close()
	slow := newIter(true)
	defer slow.Close()

	fastRes := collectStringResults(t, fast, "S")
	slowRes := collectStringResults(t, slow, "S")

	require.Equal(t, slowRes, fastRes, "dictionary fast path must match per-row results")
	require.Greater(t, fast.DictFastPathPages(), 0, "fast path should have engaged")
	require.Equal(t, 0, slow.DictFastPathPages(), "slow path should not use dict fast path")
}

func TestSyncIteratorDictPushdownRequired(t *testing.T) {
	type row struct {
		S string `parquet:",dict"`
	}
	alphabet := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	var rows []row
	for i := 0; i < 5000; i++ {
		rows = append(rows, row{S: alphabet[i%len(alphabet)]})
	}
	runDictEquivalence(t, rows, "S", func() Predicate { return NewStringInPredicate([]string{"bravo", "delta"}) })
}

func TestSyncIteratorDictPushdownOptionalWithNulls(t *testing.T) {
	type row struct {
		S *string `parquet:",dict,optional"`
	}
	alphabet := []string{"alpha", "bravo", "charlie", "delta"}
	var rows []row
	for i := 0; i < 5000; i++ {
		if i%7 == 0 {
			rows = append(rows, row{S: nil}) // null
			continue
		}
		v := alphabet[i%len(alphabet)]
		rows = append(rows, row{S: &v})
	}
	runDictEquivalence(t, rows, "S", func() Predicate { return NewStringInPredicate([]string{"alpha", "charlie"}) })
}

func TestSyncIteratorDictPushdownRepeated(t *testing.T) {
	type row struct {
		S []string `parquet:",dict,list"`
	}
	alphabet := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	var rows []row
	for i := 0; i < 3000; i++ {
		switch i % 4 {
		case 0:
			rows = append(rows, row{S: nil}) // empty list
		case 1:
			rows = append(rows, row{S: []string{alphabet[i%len(alphabet)]}})
		default:
			rows = append(rows, row{S: []string{
				alphabet[i%len(alphabet)],
				alphabet[(i+2)%len(alphabet)],
			}})
		}
	}
	runDictEquivalence(t, rows, "S.list.element", func() Predicate { return NewStringInPredicate([]string{"bravo", "echo"}) })
}

func TestSyncIteratorDictPushdownRegex(t *testing.T) {
	type row struct {
		S string `parquet:",dict"`
	}
	var rows []row
	for i := 0; i < 5000; i++ {
		rows = append(rows, row{S: fmt.Sprintf("svc-%d", i%6)})
	}
	runDictEquivalence(t, rows, "S", func() Predicate {
		p, err := NewRegexInPredicate([]string{"svc-[1-3]"})
		require.NoError(t, err)
		return p
	})
}

func TestSyncIteratorDictPushdownSubstring(t *testing.T) {
	type row struct {
		S *string `parquet:",dict,optional"`
	}
	words := []string{"alpha-one", "bravo-two", "charlie-one", "delta-three"}
	var rows []row
	for i := 0; i < 5000; i++ {
		if i%9 == 0 {
			rows = append(rows, row{S: nil})
			continue
		}
		v := words[i%len(words)]
		rows = append(rows, row{S: &v})
	}
	runDictEquivalence(t, rows, "S", func() Predicate { return NewSubstringPredicate("one") })
}

// A predicate that does not implement DictionaryPredicate must use the per-row
// path (no fast path engaged) and still return correct results.
func TestSyncIteratorDictPushdownFallsBackForNonDictPredicate(t *testing.T) {
	type row struct {
		S string `parquet:",dict"`
	}
	var rows []row
	for i := 0; i < 1000; i++ {
		rows = append(rows, row{S: fmt.Sprintf("svc-%d", i%5)})
	}
	ctx := context.Background()
	pf := createFileWith(t, ctx, rows)
	idx, _, _ := GetColumnIndexByPath(pf, "S")

	// ByteNotInPredicate keeps nulls, so it intentionally does not implement
	// DictionaryPredicate.
	iter := NewSyncIterator(ctx, pf.RowGroups(), idx,
		SyncIteratorOptSelectAs("S"),
		SyncIteratorOptPredicate(NewStringNotInPredicate([]string{"svc-1", "svc-2"})),
	)
	defer iter.Close()

	got := collectStringResults(t, iter, "S")
	require.NotEmpty(t, got)
	require.Equal(t, 0, iter.DictFastPathPages(), "non-dictionary predicate must not use dict fast path")
}
