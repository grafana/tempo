package parquetquery

import (
	"context"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

func TestNilIterator(t *testing.T) {
	count := 90_000
	pf := createNilIterTestFile(t, count)
	column := "attrs.list.element.Key"

	idx, _, _ := GetColumnIndexByPath(pf, column)
	filter := NewIncludeNilStringEqualPredicate([]byte("special-key"))
	iter := NewNilSyncIterator(context.TODO(), pf.RowGroups(), idx, SyncIteratorOptSelectAs(column), SyncIteratorOptPredicate(filter), SyncIteratorOptMaxDefinitionLevel(1))
	defer iter.Close()

	matchedRow := []int{}
	for i := 0; i < count; i++ {
		if i%3 != 0 {
			matchedRow = append(matchedRow, i)
		}
	}

	// special key is added to every 3rd row, so we should get 60,000 results where the key is NOT present
	for j := 0; j < count*2/3; j++ {
		res, err := iter.Next()
		require.NoError(t, err)
		require.NotNil(t, res, "j=%d", j)
		rowEqual := EqualRowNumber(0, RowNumber{int32(matchedRow[j]), -1, -1, -1, -1, -1, -1, -1}, res.RowNumber)
		require.True(t, rowEqual, "expected row %v but got %v", matchedRow[j], res.RowNumber)
		require.Len(t, res.Entries, 1)
		require.Equal(t, column, res.Entries[0].Key)
		require.True(t, res.Entries[0].Value.IsNull())
	}

	res, err := iter.Next()
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestNilIteratorSeekTo(t *testing.T) {
	count := 9_000
	pf := createNilIterTestFile(t, count)
	column := "attrs.list.element.Key"

	idx, _, _ := GetColumnIndexByPath(pf, column)
	filter := NewIncludeNilStringEqualPredicate([]byte("special-key"))
	iter := NewNilSyncIterator(context.TODO(), pf.RowGroups(), idx, SyncIteratorOptSelectAs(column), SyncIteratorOptPredicate(filter), SyncIteratorOptMaxDefinitionLevel(1), SyncIteratorOptIntern())
	defer iter.Close()

	// special key is added to every 3rd row, so seeking to a multiple of 3
	// lands on the next row, and the following Next returns the one after it.
	for _, to := range []int32{0, 3, 301, 3000, 8997} {
		res, err := iter.SeekTo(RowNumber{to, -1, -1, -1, -1, -1, -1, -1}, 0)
		require.NoError(t, err)
		require.NotNil(t, res, "to=%d", to)
		want := to
		if want%3 == 0 {
			want++
		}
		require.Equal(t, want, res.RowNumber[0], "to=%d", to)
		require.Equal(t, column, res.Entries[0].Key)
		require.True(t, res.Entries[0].Value.IsNull())

		res, err = iter.Next()
		require.NoError(t, err)
		require.NotNil(t, res, "to=%d", to)
		want++
		if want%3 == 0 {
			want++
		}
		require.Equal(t, want, res.RowNumber[0], "to=%d", to)
		require.True(t, res.Entries[0].Value.IsNull())
	}

	res, err := iter.SeekTo(RowNumber{int32(count), -1, -1, -1, -1, -1, -1, -1}, 0)
	require.NoError(t, err)
	require.Nil(t, res)
}

func BenchmarkNilIterator(b *testing.B) {
	count := 90_000
	pf := createNilIterTestFile(b, count)
	column := "attrs.list.element.Key"
	idx, _, _ := GetColumnIndexByPath(pf, column)

	b.ReportAllocs()
	for b.Loop() {
		filter := NewIncludeNilStringEqualPredicate([]byte("special-key"))
		iter := NewNilSyncIterator(context.TODO(), pf.RowGroups(), idx, SyncIteratorOptSelectAs(column), SyncIteratorOptPredicate(filter), SyncIteratorOptMaxDefinitionLevel(1), SyncIteratorOptIntern())
		n := 0
		for {
			res, err := iter.Next()
			if err != nil {
				b.Fatal(err)
			}
			if res == nil {
				break
			}
			n++
		}
		iter.Close()
		require.Equal(b, count*2/3, n)
	}
}

func createNilIterTestFile(t testing.TB, count int) *parquet.File {
	type Attr struct {
		Key string `parquet:",snappy,dict"`
	}
	type T struct {
		Attrs []Attr `parquet:"attrs,list"`
	}
	genericKeys := []string{"key-1", "key-2", "key-3", "key-4", "key-5"}

	rows := []T{}
	for i := range count {
		keys := genericKeys
		if i%3 == 0 {
			keys = append(keys, "special-key")
		}
		attrs := make([]Attr, 0, len(keys))
		for _, key := range keys {
			attrs = append(attrs, Attr{Key: key})
		}
		rows = append(rows, T{Attrs: attrs})
	}

	pf := createFileWith(t, context.Background(), rows)
	return pf
}
