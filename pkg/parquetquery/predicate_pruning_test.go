package parquetquery

import (
	"bytes"
	"cmp"
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

// noIndexChunk is a chunk written without a column index.
type noIndexChunk struct{ parquet.ColumnChunk }

func (noIndexChunk) ColumnIndex() (parquet.ColumnIndex, error) {
	return nil, parquet.ErrMissingColumnIndex
}

// boundlessPage is a page that records no bounds.
type boundlessPage struct{ parquet.Page }

func (boundlessPage) Bounds() (parquet.Value, parquet.Value, bool) {
	return parquet.Value{}, parquet.Value{}, false
}

// chunkOf writes rows to a file of one row group and returns its first column chunk.
func chunkOf[T any](t *testing.T, rows []T, opts ...parquet.WriterOption) parquet.ColumnChunk {
	t.Helper()
	buf := new(bytes.Buffer)
	w := parquet.NewGenericWriter[T](buf, opts...)
	for _, row := range rows { // one write per row, so a tiny page buffer cuts a page per row
		_, err := w.Write([]T{row})
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	f, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return f.RowGroups()[0].ColumnChunks()[0]
}

func firstPage(t *testing.T, cc parquet.ColumnChunk) parquet.Page {
	t.Helper()
	h := &ColumnChunkHelper{ColumnChunk: cc}
	t.Cleanup(func() { _ = h.Close() })
	pg, err := h.NextPage()
	require.NoError(t, err)
	return pg
}

func TestKeepColumnChunkDecisions(t *testing.T) {
	s := func(v string) *string { return &v }
	n := func(v int64) *int64 { return &v }
	dictWithNulls := chunkOf(t, []testOptString{{s("abc")}, {nil}})
	dictNoNulls := chunkOf(t, []testOptString{{s("abc")}, {s("bcd")}})
	plainWithNulls := chunkOf(t, []testOptInt{{n(5)}, {nil}})
	plainNoNulls := chunkOf(t, []testOptInt{{n(5)}, {n(6)}})
	nullPageFirst := chunkOf(t, []testOptInt{{nil}, {n(5)}}, parquet.PageBufferSize(1))
	// Keeps nulls without supporting column index pruning.
	keepsNulls := NewIncludeNilStringEqualPredicate(nil)

	tests := []struct {
		name       string
		chunk      parquet.ColumnChunk
		pred       Predicate
		keep       bool
		indexReads int
	}{
		{"no filter", dictWithNulls, nil, true, 0},
		{"a dictionary entry matches", dictWithNulls, NewStringInPredicate([]string{"abc"}), true, 0},
		{"no entry matches and the filter rejects nulls", dictWithNulls, NewStringInPredicate([]string{"x"}), false, 0},
		{"no entry matches, the filter keeps nulls and the chunk has some", dictWithNulls, NewNilValuePredicate(), true, 1},
		{"no entry matches, the filter keeps nulls but the chunk has none", dictNoNulls, NewNilValuePredicate(), false, 1},
		{"the null check runs for a filter without index pruning", dictWithNulls, keepsNulls, true, 1},
		{"and prunes when there are no nulls", dictNoNulls, keepsNulls, false, 1},
		{"no column index assumes nulls", noIndexChunk{dictNoNulls}, NewNilValuePredicate(), true, 1},
		{"a page with nulls the filter keeps", plainWithNulls, NewNilValuePredicate(), true, 1},
		{"page bounds pass the range", plainNoNulls, NewIntBetweenPredicate(0, 10), true, 1},
		{"page bounds fail the range", plainNoNulls, NewIntBetweenPredicate(50, 60), false, 1},
		// Its empty bounds read as 0, which would wrongly pass [0,3].
		{"a null page is skipped for the range test", nullPageFirst, NewIntBetweenPredicate(0, 3), false, 1},
		{"no column index keeps the chunk", noIndexChunk{plainNoNulls}, NewIntBetweenPredicate(50, 60), true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &countingChunk{ColumnChunk: tt.chunk}
			it := &SyncIterator{filter: tt.pred}
			require.Equal(t, tt.keep, it.keepColumnChunk(&ColumnChunkHelper{ColumnChunk: cc}))
			require.Equal(t, tt.indexReads, cc.indexReads, "column index reads")
		})
	}
}

func TestKeepPageDecisions(t *testing.T) {
	s := func(v string) *string { return &v }
	n := func(v int64) *int64 { return &v }
	mixed := firstPage(t, chunkOf(t, []testOptInt{{n(5)}, {nil}}))
	onlyNulls := firstPage(t, chunkOf(t, []testOptInt{{nil}, {nil}}))
	dictionary := firstPage(t, chunkOf(t, []testOptString{{s("abc")}}))
	require.NotNil(t, dictionary.Dictionary())

	tests := []struct {
		name string
		page parquet.Page
		pred Predicate
		keep bool
	}{
		{"nulls the filter keeps", mixed, NewNilValuePredicate(), true},
		{"only nulls, which the filter keeps", onlyNulls, NewNilValuePredicate(), true},
		{"only nulls, which the filter rejects", onlyNulls, NewIntBetweenPredicate(0, 10), false},
		{"bounds pass the range", mixed, NewIntBetweenPredicate(0, 10), true},
		{"bounds fail the range", mixed, NewIntBetweenPredicate(50, 60), false},
		{"a dictionary page skips the range test", dictionary, NewStringInPredicate([]string{"x"}), true},
		{"no bounds recorded", boundlessPage{mixed}, NewIntBetweenPredicate(50, 60), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &SyncIterator{filter: tt.pred}
			require.Equal(t, tt.keep, it.keepPage(tt.page))
		})
	}
}

// TestKeepRangeAgreesWithKeepValue checks every predicate's KeepRange against its own
// KeepValue over each range [lo,hi] of a small sorted domain:
//   - a range holding a value KeepValue keeps is never pruned;
//   - on integers, where every value in a range is in the domain, a predicate that
//     supports column index pruning keeps exactly the ranges holding such a value;
//   - a predicate that doesn't support it answers regardless of the bounds, as
//     keepColumnChunk relies on when it passes null ones.
func TestKeepRangeAgreesWithKeepValue(t *testing.T) {
	ints := []parquet.Value{parquet.Int64Value(-2), parquet.Int64Value(-1), parquet.Int64Value(0), parquet.Int64Value(1), parquet.Int64Value(2)}
	floats := []parquet.Value{parquet.DoubleValue(-1), parquet.DoubleValue(-0.5), parquet.DoubleValue(0), parquet.DoubleValue(0.5), parquet.DoubleValue(1)}
	strs := []parquet.Value{}
	for _, s := range []string{"", "a", "ab", "b", "ba", "c"} {
		strs = append(strs, parquet.ByteArrayValue([]byte(s)))
	}
	bools := []parquet.Value{parquet.BooleanValue(false), parquet.BooleanValue(true)}
	regexIn, err := NewRegexInPredicate([]string{"b.*"})
	require.NoError(t, err)
	regexNotIn, err := NewRegexNotInPredicate([]string{"b.*"})
	require.NoError(t, err)

	tests := []struct {
		domain []parquet.Value
		exact  bool // every value of a range is in the domain
		preds  []Predicate
	}{
		{ints, true, []Predicate{
			NewIntEqualPredicate(0), NewIntNotEqualPredicate(0),
			NewIntGreaterPredicate(0), NewIntGreaterEqualPredicate(0), NewIntLessPredicate(0), NewIntLessEqualPredicate(0),
			NewIntInPredicate([]int64{-2, 1}), NewIntNotInPredicate([]int64{0}),
			NewIntBetweenPredicate(-1, 1),
			NewGenericPredicate(func(v int64) bool { return v == 1 }, func(lo, hi int64) bool { return lo <= 1 && 1 <= hi }, func(v parquet.Value) int64 { return v.Int64() }),
			NewGenericPredicate(func(v int64) bool { return v == 1 }, nil, func(v parquet.Value) int64 { return v.Int64() }),
			NewOrPredicate(NewIntEqualPredicate(-2), NewIntEqualPredicate(2)),
		}},
		{floats, false, []Predicate{
			NewFloatEqualPredicate(0), NewFloatNotEqualPredicate(0),
			NewFloatGreaterPredicate(0), NewFloatGreaterEqualPredicate(0), NewFloatLessPredicate(0), NewFloatLessEqualPredicate(0),
			NewFloatInPredicate([]float64{-1, 0.5}), NewFloatNotInPredicate([]float64{0}),
		}},
		{strs, false, []Predicate{
			NewStringEqualPredicate("b"), NewStringNotEqualPredicate("b"),
			NewStringGreaterPredicate([]byte("b")), NewStringGreaterEqualPredicate([]byte("b")),
			NewStringLessPredicate([]byte("b")), NewStringLessEqualPredicate([]byte("b")),
			NewStringInPredicate([]string{"ab", "c"}), NewStringNotInPredicate([]string{"b"}),
			NewSubstringPredicate("b"), regexIn, regexNotIn,
			NewIncludeNilStringEqualPredicate([]byte("b")), NewSkipNilsPredicate(),
			NewOrPredicate(NewStringEqualPredicate("a"), NewSubstringPredicate("c")),
		}},
		{bools, true, []Predicate{
			NewBoolEqualPredicate(true), NewBoolNotEqualPredicate(true),
			NewBoolInPredicate([]bool{true}), NewBoolNotInPredicate([]bool{true}),
		}},
	}

	null := predicateNullValue()
	for _, tc := range tests {
		for _, p := range tc.preds {
			t.Run(p.String(), func(t *testing.T) {
				for i, lo := range tc.domain {
					for _, hi := range tc.domain[i:] {
						matches := false
						for _, v := range tc.domain {
							if v.Kind() == lo.Kind() && compareValues(v, lo) >= 0 && compareValues(v, hi) <= 0 && p.KeepValue(v) {
								matches = true
							}
						}
						keep := p.KeepRange(lo, hi)
						if matches {
							require.True(t, keep, "pruned [%v,%v] holding a match", lo, hi)
						}
						if !p.SupportsColumnIndexPruning() {
							require.Equal(t, p.KeepRange(null, null), keep, "[%v,%v] depends on bounds", lo, hi)
						} else if tc.exact {
							require.Equal(t, matches, keep, "[%v,%v]", lo, hi)
						}
					}
				}
			})
		}
	}
}

func compareValues(a, b parquet.Value) int {
	switch a.Kind() {
	case parquet.Int64:
		return cmp.Compare(a.Int64(), b.Int64())
	case parquet.Double:
		return cmp.Compare(a.Double(), b.Double())
	case parquet.Boolean:
		return cmp.Compare(b2i(a.Boolean()), b2i(b.Boolean()))
	default:
		return bytes.Compare(a.ByteArray(), b.ByteArray())
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
