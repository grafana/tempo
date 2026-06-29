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
		it := NewSyncIterator(ctx, pf.RowGroups(), idx, opts...)
		// indexReaderDisabled is test-only state; the fast path is always on in prod.
		// It is read lazily on the first page, so setting it post-construction is fine.
		it.indexReaderDisabled = disable
		return it
	}

	fast := newIter(false)
	defer fast.Close()
	slow := newIter(true)
	defer slow.Close()

	fastRes := collectStringResults(t, fast, "S")
	slowRes := collectStringResults(t, slow, "S")

	require.Equal(t, slowRes, fastRes, "dictionary fast path must match per-row results")
	require.Greater(t, fast.dictFastPathPages(), 0, "fast path should have engaged")
	require.Equal(t, 0, slow.dictFastPathPages(), "slow path should not use dict fast path")
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
	require.Equal(t, 0, iter.dictFastPathPages(), "non-dictionary predicate must not use dict fast path")
}

// TestIndexValueReaderNoPresentValues guards the fast path against dictionary-
// encoded pages that carry definition levels but no present leaf values (e.g. an
// all-null optional page or a repeated page of only empty lists). There indices
// is empty, so naively treating slots as present would index out of bounds. The
// page must yield no matches, advance the row-number cursor once per slot (as the
// per-row path does), and never panic.
func TestIndexValueReaderNoPresentValues(t *testing.T) {
	cases := []struct {
		name      string
		defLevels []byte
		repLevels []byte
	}{
		// All-null optional page: every slot below the present level, so pageMaxDef
		// collapses to the null level and only the valueN bound prevents the panic.
		{name: "all-null optional", defLevels: []byte{0, 0, 0, 0}},
		// Repeated page of empty lists: rep=0 starts each row, def below present.
		{name: "all-empty repeated", defLevels: []byte{0, 0, 0}, repLevels: []byte{0, 0, 0}},
		// Pathological: a slot sits at pageMaxDef (looks present) but no index backs
		// it. The bounds guard must treat it as null rather than indexing past indices.
		{name: "present level without index", defLevels: []byte{1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &indexValueReader{
				matches:    nil, // never consulted: no present values
				dict:       nil,
				indices:    nil, // empty: no present values on the page
				defLevels:  tc.defLevels,
				repLevels:  tc.repLevels,
				pageMaxDef: maxByte(tc.defLevels), // mirrors newIndexValueReader
			}

			require.NotPanics(t, func() {
				curr := EmptyRowNumber()
				pageN := 0
				v, ok := r.nextMatch(&curr, &pageN, MaxDefinitionLevel)
				require.False(t, ok, "an all-null page must yield no matches")
				require.Nil(t, v)
				// Every slot must still have been walked (row numbers advanced).
				require.Equal(t, len(tc.defLevels), pageN, "every slot should advance the cursor")
			})
		})
	}
}
