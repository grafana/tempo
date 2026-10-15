package parquetquery

import (
	"bytes"
	"context"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

// testPlainInt is deliberately not dict-encoded, so pruning falls to the column index
// and page bounds rather than the dictionary.
type testPlainInt struct {
	N int64
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

	// SyncIteratorOptStats attaches a PredicateStats, so keepColumnChunk counts into it.
	var keepStats PredicateStats
	keepIt := NewSyncIterator(context.TODO(), r.RowGroups(), 0,
		SyncIteratorOptPredicate(NewStringInPredicate([]string{"abc"})), SyncIteratorOptStats(&keepStats))
	defer keepIt.Close()
	ccKeep := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
	defer ccKeep.Close()
	require.True(t, keepIt.keepColumnChunk(ccKeep))
	require.Equal(t, int64(1), keepStats.InspectedColumnChunks)
	require.Equal(t, int64(1), keepStats.KeptColumnChunks)

	var skipStats PredicateStats
	skipIt := NewSyncIterator(context.TODO(), r.RowGroups(), 0,
		SyncIteratorOptPredicate(NewStringInPredicate([]string{"zzz"})), SyncIteratorOptStats(&skipStats))
	defer skipIt.Close()
	ccSkip := &ColumnChunkHelper{ColumnChunk: r.RowGroups()[0].ColumnChunks()[0]}
	defer ccSkip.Close()
	require.False(t, skipIt.keepColumnChunk(ccSkip))
	require.Equal(t, int64(1), skipStats.InspectedColumnChunks)
	require.Equal(t, int64(0), skipStats.KeptColumnChunks) // chunk skipped
}

// TestKeepPagePrunesByBounds covers page-level KeepRange pruning on a plain (non-dict)
// column, where the chunk is kept but individual pages are skippable by their bounds.
// Dict-encoded columns skip at the chunk level instead, so they never reach this path.
func TestKeepPagePrunesByBounds(t *testing.T) {
	const rows = 2000

	buf := new(bytes.Buffer)
	w := parquet.NewWriter(buf, parquet.PageBufferSize(1024))
	for i := 0; i < rows; i++ {
		require.NoError(t, w.Write(&testPlainInt{int64(i)}))
	}
	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())
	r, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	// Matches only the tail of an ascending column, so the leading pages are out of range.
	var stats PredicateStats
	it := NewSyncIterator(context.TODO(), r.RowGroups(), 0,
		SyncIteratorOptPredicate(NewIntBetweenPredicate(rows-5, rows)), SyncIteratorOptStats(&stats))
	defer it.Close()
	drain(t, it)

	require.Equal(t, int64(1), stats.KeptColumnChunks, "chunk holds matching values, must be kept")
	require.Greater(t, stats.InspectedPages, int64(1), "need multiple pages to exercise pruning")
	require.Less(t, stats.KeptPages, stats.InspectedPages, "out-of-range pages must be pruned")
	require.Equal(t, int64(5), stats.KeptValues)
}

// TestStatsCountOnlyRowValues pins the invariant that replaced InstrumentedPredicate's
// inner-predicate unwrap: the null and dictionary KeepValue probes issued by
// keepColumnChunk / keepPage must not inflate the value counters, which count rows only.
func TestStatsCountOnlyRowValues(t *testing.T) {
	r := buildFile(t, func(w *parquet.Writer) { //nolint:all
		require.NoError(t, w.Write(&testDictString{"abc"}))
		require.NoError(t, w.Write(&testDictString{"bcd"}))
		require.NoError(t, w.Write(&testDictString{"cde"}))
	})

	var stats PredicateStats
	it := NewSyncIterator(context.TODO(), r.RowGroups(), 0,
		SyncIteratorOptPredicate(NewStringInPredicate([]string{"abc"})), SyncIteratorOptStats(&stats))
	defer it.Close()
	drain(t, it)

	// Exactly one inspection per row: the 3-entry dictionary scan and the null probe
	// that keepColumnChunk performed are not counted.
	require.Equal(t, int64(3), stats.InspectedValues)
	require.Equal(t, int64(1), stats.KeptValues)
}

func drain(t *testing.T, it Iterator) {
	t.Helper()
	for {
		res, err := it.Next()
		require.NoError(t, err)
		if res == nil {
			return
		}
	}
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

// countingChunk counts column index reads, each a backend request on a real block.
type countingChunk struct {
	parquet.ColumnChunk
	indexReads int
}

func (c *countingChunk) ColumnIndex() (parquet.ColumnIndex, error) {
	c.indexReads++
	return c.ColumnChunk.ColumnIndex()
}

// TestKeepColumnChunkReadsIndexOnlyWhenNeeded: a predicate whose chunk decision
// the index cannot help must not pay for reading it.
func TestKeepColumnChunkReadsIndexOnlyWhenNeeded(t *testing.T) {
	pf := createTestFile(t, 100) // plain int column A = 0..99

	tests := []struct {
		name       string
		pred       Predicate
		keep       bool
		indexReads int
	}{
		{"range predicate in range", NewIntBetweenPredicate(10, 20), true, 1},
		{"range predicate out of range", NewIntBetweenPredicate(500, 600), false, 1},
		{"nil predicate needs null counts", NewNilValuePredicate(), false, 1},
		{"skip nils", NewSkipNilsPredicate(), true, 0},
		{"callback keeps", NewCallbackPredicate(func() bool { return true }), true, 0},
		{"callback prunes", NewCallbackPredicate(func() bool { return false }), false, 0},
		{"or of rangeless branches", NewOrPredicate(NewSkipNilsPredicate(), NewCallbackPredicate(func() bool { return false })), true, 0},
		{"or with a range branch", NewOrPredicate(NewCallbackPredicate(func() bool { return false }), NewIntBetweenPredicate(500, 600)), false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &countingChunk{ColumnChunk: pf.RowGroups()[0].ColumnChunks()[0]}
			it := &SyncIterator{filter: tt.pred}
			require.Equal(t, tt.keep, it.keepColumnChunk(&ColumnChunkHelper{ColumnChunk: cc}))
			require.Equal(t, tt.indexReads, cc.indexReads)
		})
	}
}
