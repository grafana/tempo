package parquetquery

import (
	"bytes"
	"context"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

type testOptString struct {
	S *string `parquet:",dict,optional"`
}

func TestPredicateNullValueIsNull(t *testing.T) {
	require.True(t, predicateNullValue().IsNull())
}

// TestKeepHelpersMatchValueOracle asserts the generic keepColumnChunk/keepPage
// helpers agree with the authoritative per-row semantic: a chunk/page must be
// kept iff some value in it (nulls included) satisfies KeepValue.
func TestKeepHelpersMatchValueOracle(t *testing.T) {
	sp := func(s string) *string { return &s }

	cases := []struct {
		name      string
		predicate Predicate
		writeData func(w *parquet.Writer) //nolint:all
	}{
		{
			name:      "dict string IN, match present",
			predicate: NewStringInPredicate([]string{"abc", "acd"}),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"acd"}))
				require.NoError(t, w.Write(&testDictString{"cde"}))
			},
		},
		{
			name:      "dict string IN, no match (chunk skippable)",
			predicate: NewStringInPredicate([]string{"x"}),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
			},
		},
		{
			name:      "dict string substring match",
			predicate: NewSubstringPredicate("b"),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"bcd"}))
			},
		},
		{
			name:      "dict string substring no match",
			predicate: NewSubstringPredicate("zz"),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testDictString{"abc"}))
				require.NoError(t, w.Write(&testDictString{"abc"}))
			},
		},
		{
			name:      "int between, no match",
			predicate: NewIntBetweenPredicate(5, 10),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{1}))
				require.NoError(t, w.Write(&testInt{2}))
				require.NoError(t, w.Write(&testInt{3}))
			},
		},
		{
			name:      "int between, match",
			predicate: NewIntBetweenPredicate(2, 3),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testInt{1}))
				require.NoError(t, w.Write(&testInt{2}))
				require.NoError(t, w.Write(&testInt{3}))
			},
		},
		{
			name:      "nil value predicate, nulls present",
			predicate: NewNilValuePredicate(),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testOptString{sp("abc")}))
				require.NoError(t, w.Write(&testOptString{nil}))
			},
		},
		{
			name:      "nil value predicate, no nulls",
			predicate: NewNilValuePredicate(),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testOptString{sp("abc")}))
				require.NoError(t, w.Write(&testOptString{sp("def")}))
			},
		},
		{
			name:      "skip nils predicate, present values",
			predicate: NewSkipNilsPredicate(),
			writeData: func(w *parquet.Writer) { //nolint:all
				require.NoError(t, w.Write(&testOptString{sp("abc")}))
				require.NoError(t, w.Write(&testOptString{nil}))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := buildFile(t, tc.writeData)

			oracle := oracleKeep(t, r, tc.predicate)

			it := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptPredicate(tc.predicate))
			defer it.Close()

			cc := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
			defer cc.Close()
			require.Equal(t, oracle, it.keepColumnChunk(cc), "keepColumnChunk vs oracle")

			ccPage := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
			defer ccPage.Close()
			pg, err := ccPage.NextPage()
			require.NoError(t, err)
			require.NotNil(t, pg)
			// keepPage must never under-keep: whenever a value matches (oracle true)
			// the page must be kept. It may conservatively over-keep for predicates
			// with no usable page-level range (e.g. substring/regex), whose tight skip
			// happens at the chunk level (dictionary) instead.
			if oracle {
				require.True(t, it.keepPage(pg), "keepPage must keep a matching page")
			}
		})
	}
}

func TestKeepStatsCounters(t *testing.T) {
	r := buildFile(t, func(w *parquet.Writer) { //nolint:all
		require.NoError(t, w.Write(&testDictString{"abc"}))
	})

	// An InstrumentedPredicate sets c.stats, so keepColumnChunk counts into it.
	keepIP := &InstrumentedPredicate{Pred: NewStringInPredicate([]string{"abc"})}
	keepIt := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptPredicate(keepIP))
	defer keepIt.Close()
	ccKeep := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
	defer ccKeep.Close()
	require.True(t, keepIt.keepColumnChunk(ccKeep))
	require.Equal(t, int64(1), keepIP.InspectedColumnChunks)
	require.Equal(t, int64(1), keepIP.KeptColumnChunks)

	skipIP := &InstrumentedPredicate{Pred: NewStringInPredicate([]string{"zzz"})}
	skipIt := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptPredicate(skipIP))
	defer skipIt.Close()
	ccSkip := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
	defer ccSkip.Close()
	require.False(t, skipIt.keepColumnChunk(ccSkip))
	require.Equal(t, int64(1), skipIP.InspectedColumnChunks)
	require.Equal(t, int64(0), skipIP.KeptColumnChunks) // chunk skipped
}

func buildFile(t *testing.T, writeData func(w *parquet.Writer)) *parquet.File { //nolint:all
	t.Helper()
	buf := new(bytes.Buffer)
	w := parquet.NewWriter(buf)
	writeData(w)
	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())
	r, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return r
}

// oracleKeep enumerates every value in column 0 (nulls included) with an
// unfiltered iterator and reports whether any satisfies pred.KeepValue.
func oracleKeep(t *testing.T, r *parquet.File, pred Predicate) bool {
	t.Helper()
	it := NewSyncIterator(context.TODO(), r.RowGroups(), 0, SyncIteratorOptSelectAs("v"))
	defer it.Close()
	for {
		res, err := it.Next()
		require.NoError(t, err)
		if res == nil {
			return false
		}
		for _, e := range res.Entries {
			if pred.KeepValue(e.Value) {
				return true
			}
		}
	}
}
