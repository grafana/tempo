package parquetquery

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

type dictRow struct {
	S *string  `parquet:",dict,optional"`
	L []string `parquet:",dict"`
}

// dictTestFile writes rows mixing values, nulls and empty lists over several row groups
// and pages, all dictionary-encoded.
func dictTestFile(t *testing.T) *parquet.File {
	t.Helper()
	words := []string{"", "api", "db", "shop", "web", "zz"}
	var rows []dictRow
	for i := range 600 {
		r := dictRow{}
		if i%5 != 0 {
			s := words[i%len(words)]
			r.S = &s
		}
		for j := range i % 4 {
			r.L = append(r.L, words[(i+j)%len(words)])
		}
		rows = append(rows, r)
	}

	buf := new(bytes.Buffer)
	w := parquet.NewGenericWriter[dictRow](buf, parquet.PageBufferSize(64), parquet.MaxRowsPerRowGroup(200))
	_, err := w.Write(rows)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	f, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, f.RowGroups(), 3)
	return f
}

type dictHit struct {
	rn    RowNumber
	value string
	null  bool
}

func collectHits(t *testing.T, it Iterator, seekStep int32) []dictHit {
	t.Helper()
	var hits []dictHit
	var to RowNumber
	for {
		var res *IteratorResult
		var err error
		if seekStep > 0 && len(hits) > 0 {
			to = EmptyRowNumber()
			to[0] = hits[len(hits)-1].rn[0] + seekStep
			res, err = it.SeekTo(to, 0)
		} else {
			res, err = it.Next()
		}
		require.NoError(t, err)
		if res == nil {
			return hits
		}
		v := res.Entries[0].Value
		hits = append(hits, dictHit{rn: res.RowNumber, value: string(v.ByteArray()), null: v.IsNull()})
	}
}

// oracleHits reads the column without a filter and applies pred itself, row by row.
func oracleHits(t *testing.T, f *parquet.File, column, maxDef int, pred Predicate, seekStep int32) []dictHit {
	t.Helper()
	it := NewSyncIterator(context.TODO(), f.RowGroups(), column, SyncIteratorOptSelectAs("v"), SyncIteratorOptMaxDefinitionLevel(maxDef))
	defer it.Close()
	var hits []dictHit
	for _, h := range collectHits(t, it, 0) {
		v := parquet.ValueOf(h.value)
		if h.null {
			v = parquet.Value{}
		}
		if pred.KeepValue(v) {
			if seekStep > 0 && len(hits) > 0 && h.rn[0] < hits[len(hits)-1].rn[0]+seekStep {
				continue
			}
			hits = append(hits, h)
		}
	}
	return hits
}

// TestDictPushdownMatchesPerRow: on dictionary chunks, every predicate returns exactly
// the rows and values the per-row path would, including nulls, empty lists and seeks.
func TestDictPushdownMatchesPerRow(t *testing.T) {
	f := dictTestFile(t)
	regexIn, err := NewRegexInPredicate([]string{"(sh|w).*"})
	require.NoError(t, err)
	regexNotIn, err := NewRegexNotInPredicate([]string{"a.*"})
	require.NoError(t, err)

	preds := []Predicate{
		NewStringInPredicate([]string{"shop"}),
		NewStringInPredicate([]string{"api", "zz", "missing"}),
		NewStringInPredicate([]string{"missing"}),
		NewStringNotInPredicate([]string{"db"}),
		NewStringEqualPredicate(""),
		NewStringNotEqualPredicate("web"),
		NewStringGreaterPredicate([]byte("db")),
		NewStringLessEqualPredicate([]byte("db")),
		regexIn,
		regexNotIn,
		NewSubstringPredicate("o"),
		NewNilValuePredicate(),
		NewSkipNilsPredicate(),
		NewIncludeNilStringEqualPredicate([]byte("shop")),
		NewOrPredicate(NewStringEqualPredicate("db"), NewSubstringPredicate("e")),
		NewCallbackPredicate(func() bool { return true }),
	}

	columns := []struct {
		name          string
		index, maxDef int
	}{
		{"optional", 0, 1},
		{"repeated", 1, 1},
	}

	for _, col := range columns {
		for _, pred := range preds {
			for _, step := range []int32{0, 7} {
				t.Run(fmt.Sprintf("%s/%s/seek=%d", col.name, pred, step), func(t *testing.T) {
					it := NewSyncIterator(context.TODO(), f.RowGroups(), col.index, SyncIteratorOptSelectAs("v"),
						SyncIteratorOptMaxDefinitionLevel(col.maxDef), SyncIteratorOptPredicate(pred))
					defer it.Close()
					got := collectHits(t, it, step)
					require.Equal(t, oracleHits(t, f, col.index, col.maxDef, pred, step), got)
				})
			}
		}
	}
}

// TestDictPushdownEngages: a filtered dictionary chunk is served by the index reader.
func TestDictPushdownEngages(t *testing.T) {
	f := dictTestFile(t)
	it := NewSyncIterator(context.TODO(), f.RowGroups(), 0, SyncIteratorOptSelectAs("v"), SyncIteratorOptPredicate(NewStringInPredicate([]string{"shop"})))
	defer it.Close()
	res, err := it.Next()
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "shop", string(res.Entries[0].Value.ByteArray()))
	require.NotNil(t, it.dictPageReader.matches)
	require.True(t, it.dictPageReader.active)

	unfiltered := NewSyncIterator(context.TODO(), f.RowGroups(), 0, SyncIteratorOptSelectAs("v"))
	defer unfiltered.Close()
	_, err = unfiltered.Next()
	require.NoError(t, err)
	require.False(t, unfiltered.dictPageReader.active, "no filter, nothing to push down")
}

// TestDictPushdownCountsRows: the stats count every row the fast path inspects and keeps,
// as the per-row path does.
func TestDictPushdownCountsRows(t *testing.T) {
	f := dictTestFile(t)
	var stats PredicateStats
	it := NewSyncIterator(context.TODO(), f.RowGroups(), 0, SyncIteratorOptPredicate(NewStringInPredicate([]string{"shop"})), SyncIteratorOptStats(&stats))
	defer it.Close()
	drain(t, it)

	var kept int64
	for i := range 600 {
		if i%5 != 0 && i%6 == 3 {
			kept++
		}
	}
	require.Equal(t, int64(600), stats.InspectedValues)
	require.Equal(t, kept, stats.KeptValues)
}

// TestDictPushdownFileSchemaMismatch: a file opened with a schema that doesn't match
// its columns (parquet.FileSchema, as Tempo does from block metadata) still reads each
// chunk's definition levels from the chunk's own column.
func TestDictPushdownFileSchemaMismatch(t *testing.T) {
	type fileRow struct {
		A *string `parquet:",optional"`
		S *string `parquet:",dict,optional"`
	}
	type readRow struct {
		A *string `parquet:",optional"`
		X *struct {
			Y *string `parquet:",optional"`
		} `parquet:",optional"`
		S *string `parquet:",dict,optional"`
	}

	var rows []fileRow
	for i := range 100 {
		s := []string{"api", "shop", "web"}[i%3]
		rows = append(rows, fileRow{S: &s})
	}
	buf := new(bytes.Buffer)
	w := parquet.NewGenericWriter[fileRow](buf)
	_, err := w.Write(rows)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	f, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()), parquet.FileSchema(parquet.SchemaOf(readRow{})))
	require.NoError(t, err)
	column, _, maxDef := GetColumnIndexByPath(f, "S")
	require.Equal(t, 1, column)

	schema := f.RowGroups()[0].Schema()
	leaf, ok := schema.Lookup(schema.Columns()[column]...)
	require.True(t, ok)
	require.NotEqual(t, maxDef, leaf.MaxDefinitionLevel, "the read schema must map the index to another column")

	pred := NewStringInPredicate([]string{"shop"})
	it := NewSyncIterator(context.TODO(), f.RowGroups(), column, SyncIteratorOptSelectAs("v"),
		SyncIteratorOptMaxDefinitionLevel(maxDef), SyncIteratorOptPredicate(pred))
	defer it.Close()
	require.Equal(t, oracleHits(t, f, column, maxDef, pred, 0), collectHits(t, it, 0))
}

// TestDictPushdownNeedsFileColumn: a row group that isn't read from a file has no
// file column to take definition levels from, so it isn't pushed down, and still
// returns every matching row.
func TestDictPushdownNeedsFileColumn(t *testing.T) {
	buf := parquet.NewGenericBuffer[dictRow]()
	_, err := buf.Write([]dictRow{{S: strPtr("shop")}, {}, {S: strPtr("api")}, {S: strPtr("shop")}})
	require.NoError(t, err)

	pred := NewStringInPredicate([]string{"shop"})
	it := NewSyncIterator(context.TODO(), []parquet.RowGroup{buf}, 0, SyncIteratorOptSelectAs("v"),
		SyncIteratorOptMaxDefinitionLevel(1), SyncIteratorOptPredicate(pred))
	defer it.Close()
	hits := collectHits(t, it, 0)
	require.False(t, it.dictPageReader.active)
	require.Equal(t, []dictHit{
		{rn: RowNumber{0, 0, -1, -1, -1, -1, -1, -1}, value: "shop"},
		{rn: RowNumber{3, 0, -1, -1, -1, -1, -1, -1}, value: "shop"},
	}, hits)
}

// TestDictPushdownLongSeek: a seek past more than 1000 values (seekWithinPage's
// magicThreshold) reslices the page and points the reader at the slice.
func TestDictPushdownLongSeek(t *testing.T) {
	var rows []dictRow
	for i := range 20_000 {
		if i%7 != 0 {
			rows = append(rows, dictRow{S: strPtr([]string{"api", "shop", "web"}[i%3])})
		} else {
			rows = append(rows, dictRow{})
		}
	}
	buf := new(bytes.Buffer)
	w := parquet.NewGenericWriter[dictRow](buf, parquet.PageBufferSize(1<<20))
	_, err := w.Write(rows)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	f, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, f.RowGroups(), 1)
	require.Greater(t, f.RowGroups()[0].ColumnChunks()[0].NumValues(), int64(2_500), "one page holds more than a seek")

	pred := NewStringInPredicate([]string{"shop"})
	it := NewSyncIterator(context.TODO(), f.RowGroups(), 0, SyncIteratorOptSelectAs("v"),
		SyncIteratorOptMaxDefinitionLevel(1), SyncIteratorOptPredicate(pred))
	defer it.Close()
	require.Equal(t, oracleHits(t, f, 0, 1, pred, 2_500), collectHits(t, it, 2_500))
}

func strPtr(s string) *string { return &s }
