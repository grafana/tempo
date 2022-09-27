package parquetquery

import (
	"context"
	"os"
	"testing"

	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/require"
)

func TestRowNumber(t *testing.T) {
	tr := EmptyRowNumber()
	require.Equal(t, RowNumber{-1, -1, -1, -1, -1, -1}, tr)

	steps := []struct {
		repetitionLevel int
		definitionLevel int
		expected        RowNumber
	}{
		// Name.Language.Country examples from the Dremel whitepaper
		{0, 3, RowNumber{0, 0, 0, 0, -1, -1}},
		{2, 2, RowNumber{0, 0, 1, -1, -1, -1}},
		{1, 1, RowNumber{0, 1, -1, -1, -1, -1}},
		{1, 3, RowNumber{0, 2, 0, 0, -1, -1}},
		{0, 1, RowNumber{1, 0, -1, -1, -1, -1}},
	}

	for _, step := range steps {
		tr.Next(step.repetitionLevel, step.definitionLevel)
		require.Equal(t, step.expected, tr)
	}
}

func TestCompareRowNumbers(t *testing.T) {
	testCases := []struct {
		a, b     RowNumber
		expected int
	}{
		{RowNumber{-1}, RowNumber{0}, -1},
		{RowNumber{0}, RowNumber{0}, 0},
		{RowNumber{1}, RowNumber{0}, 1},

		{RowNumber{0, 1}, RowNumber{0, 2}, -1},
		{RowNumber{0, 2}, RowNumber{0, 1}, 1},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, CompareRowNumbers(5, tc.a, tc.b))
	}
}

func TestColumnIterator(t *testing.T) {
	type T struct{ A int }

	rows := []T{}
	count := 10_000
	for i := 0; i < count; i++ {
		rows = append(rows, T{i})
	}

	pf := createFileWith(t, rows)

	idx, _ := GetColumnIndexByPath(pf, "A")
	iter := NewColumnIterator(context.TODO(), pf.RowGroups(), idx, "", 1000, nil, "A")
	defer iter.Close()

	for i := 0; i < count; i++ {
		res, err := iter.Next()
		require.NoError(t, err)

		require.Equal(t, RowNumber{int64(i), -1, -1, -1, -1, -1}, res.RowNumber)
		require.Equal(t, int64(i), res.ToMap()["A"][0].Int64())
	}

	res, err := iter.Next()
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestColumnIteratorContextCanceled(t *testing.T) {
	type T struct{ A int }

	rows := []T{}
	count := 10_000
	for i := 0; i < count; i++ {
		rows = append(rows, T{i})
	}

	pf := createFileWith(t, rows)
	idx, _ := GetColumnIndexByPath(pf, "A")

	ctx, cancel := context.WithCancel(context.TODO())
	cancel()

	iter := NewColumnIterator(ctx, pf.RowGroups(), idx, "", 1000, nil, "A")
	defer iter.Close()

	// Verify iterator exits early
	received := 0
	for {
		res, err := iter.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
		received++
	}
	require.Less(t, received, count)
}

func BenchmarkColumnIterator(b *testing.B) {
	type T struct{ A int }
	rows := []T{}
	count := 10_000
	for i := 0; i < count; i++ {
		rows = append(rows, T{i})
	}

	pf := createFileWith(b, rows)
	idx, _ := GetColumnIndexByPath(pf, "A")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iter := NewColumnIterator(context.TODO(), pf.RowGroups(), idx, "", 1000, nil, "A")
		for {
			res, err := iter.Next()
			require.NoError(b, err)
			if res == nil {
				break
			}
		}
		iter.Close()
	}
}

func createFileWith[T any](t testing.TB, rows []T) *parquet.File {
	f, err := os.CreateTemp(t.TempDir(), "data.parquet")
	require.NoError(t, err)

	w := parquet.NewGenericWriter[T](f)
	count, err := w.Write(rows)
	require.Equal(t, len(rows), count)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	stat, err := f.Stat()
	require.NoError(t, err)

	pf, err := parquet.OpenFile(f, stat.Size())
	require.NoError(t, err)

	return pf
}
