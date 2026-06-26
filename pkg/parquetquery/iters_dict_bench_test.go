package parquetquery

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkSyncIteratorDictPushdown compares the dictionary fast path against the
// per-row byte-compare path for an exact-match string filter, the dominant shape
// of live-store metrics queries.
func BenchmarkSyncIteratorDictPushdown(b *testing.B) {
	type row struct {
		S string `parquet:",dict"`
	}
	// Low-cardinality column (like span:name / service.name) with a small set of
	// distinct values repeated across many rows.
	alphabet := make([]string, 32)
	for i := range alphabet {
		alphabet[i] = fmt.Sprintf("operation-name-%02d", i)
	}
	rows := make([]row, 200_000)
	for i := range rows {
		rows[i] = row{S: alphabet[i%len(alphabet)]}
	}

	ctx := context.Background()
	pf := createFileWith(b, ctx, rows)
	idx, _, _ := GetColumnIndexByPath(pf, "S")
	targets := []string{alphabet[3], alphabet[17], alphabet[28]}

	run := func(b *testing.B, disable bool) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			opts := []SyncIteratorOpt{
				SyncIteratorOptSelectAs("S"),
				SyncIteratorOptPredicate(NewStringInPredicate(targets)),
			}
			if disable {
				opts = append(opts, SyncIteratorOptDisableDictPushdown())
			}
			iter := NewSyncIterator(ctx, pf.RowGroups(), idx, opts...)
			var count int
			for {
				res, err := iter.Next()
				if err != nil {
					b.Fatal(err)
				}
				if res == nil {
					break
				}
				count++
			}
			iter.Close()
			if count == 0 {
				b.Fatal("expected matches")
			}
		}
	}

	b.Run("per-row", func(b *testing.B) { run(b, true) })
	b.Run("dict-pushdown", func(b *testing.B) { run(b, false) })
}
