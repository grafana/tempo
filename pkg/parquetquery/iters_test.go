package parquetquery

import (
	"context"
	"io"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

type makeTestIterFn func(pf *parquet.File, idx int, filter Predicate, selectAs string) Iterator

var iterTestCases = []struct {
	name     string
	makeIter makeTestIterFn
}{
	{"sync", func(pf *parquet.File, idx int, filter Predicate, selectAs string) Iterator {
		return NewSyncIterator(context.TODO(), pf.RowGroups(), idx, selectAs, 1000, filter, selectAs, MaxDefinitionLevel)
	}},
}

// TestTruncate compares the unrolled TruncateRowNumber() with the original truncateRowNumberSlow() to
// prevent drift
func TestTruncateRowNumber(t *testing.T) {
	for i := 0; i <= MaxDefinitionLevel; i++ {
		rn := RowNumber{1, 2, 3, 4, 5, 6, 7, 8}

		newR := TruncateRowNumber(i, rn)
		oldR := truncateRowNumberSlow(i, rn)

		require.Equal(t, newR, oldR)
	}
}

func TestInvalidDefinitionLevelTruncate(t *testing.T) {
	t.Run("TruncateRowNumber -1", func(t *testing.T) {
		assertPanic(t, func() {
			rn := RowNumber{1, 2, 3, 4, 5, 6, 7, 8}
			d := -1
			TruncateRowNumber(d, rn)
		})
	})
	t.Run("TruncateRowNumber Max+1", func(t *testing.T) {
		assertPanic(t, func() {
			rn := RowNumber{1, 2, 3, 4, 5, 6, 7, 8}
			d := MaxDefinitionLevel + 1
			TruncateRowNumber(d, rn)
		})
	})
}

func TestRowNumberNext(t *testing.T) {
	tr := EmptyRowNumber()
	require.Equal(t, RowNumber{-1, -1, -1, -1, -1, -1, -1, -1}, tr)

	steps := []struct {
		repetitionLevel    int
		definitionLevel    int
		maxDefinitionLevel int
		expected           RowNumber
	}{
		// Name.Language.Country examples from the Dremel whitepaper
		{0, 3, 3, RowNumber{0, 0, 0, 0, -1, -1, -1, -1}},
		{2, 2, 3, RowNumber{0, 0, 1, -1, -1, -1, -1, -1}},
		{1, 1, 3, RowNumber{0, 1, -1, -1, -1, -1, -1, -1}},
		{1, 3, 3, RowNumber{0, 2, 0, 0, -1, -1, -1, -1}},
		{0, 1, 3, RowNumber{1, 0, -1, -1, -1, -1, -1, -1}},
	}

	for _, step := range steps {
		tr.Next(step.repetitionLevel, step.definitionLevel, step.maxDefinitionLevel)
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
		require.Equal(t, tc.expected, CompareRowNumbers(MaxDefinitionLevel, tc.a, tc.b))
	}
}

func TestRowNumberPreceding(t *testing.T) {
	testCases := []struct {
		start, preceding RowNumber
	}{
		{RowNumber{1000, -1, -1, -1, -1, -1, -1, -1}, RowNumber{999, -1, -1, -1, -1, -1, -1, -1}},
		{RowNumber{1000, 0, 0, 0, 0, 0, 0, 0}, RowNumber{999, math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32}},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.preceding, tc.start.Preceding())
	}
}

func TestColumnIterator(t *testing.T) {
	for _, tc := range iterTestCases {
		t.Run(tc.name, func(t *testing.T) {
			testColumnIterator(t, tc.makeIter)
		})
	}
}

func testColumnIterator(t *testing.T, makeIter makeTestIterFn) {
	count := 100_000
	pf := createTestFile(t, count)

	idx, _, _ := GetColumnIndexByPath(pf, "A")
	iter := makeIter(pf, idx, nil, "A")
	defer iter.Close()

	for i := 0; i < count; i++ {
		res, err := iter.Next()
		require.NoError(t, err)
		require.NotNil(t, res, "i=%d", i)
		require.Equal(t, RowNumber{int32(i), -1, -1, -1, -1, -1, -1, -1}, res.RowNumber)
		require.Equal(t, int64(i), res.ToMap()["A"][0].Int64())
	}

	res, err := iter.Next()
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestColumnIteratorSeek(t *testing.T) {
	for _, tc := range iterTestCases {
		t.Run(tc.name, func(t *testing.T) {
			testColumnIteratorSeek(t, tc.makeIter)
		})
	}
}

func testColumnIteratorSeek(t *testing.T, makeIter makeTestIterFn) {
	count := 10_000
	pf := createTestFile(t, count)

	idx, _, _ := GetColumnIndexByPath(pf, "A")
	iter := makeIter(pf, idx, nil, "A")
	defer iter.Close()

	seekTos := []int32{
		100,
		1234,
		4567,
		5000,
		7890,
	}

	for _, seekTo := range seekTos {
		rn := EmptyRowNumber()
		rn[0] = seekTo
		res, err := iter.SeekTo(rn, 0)
		require.NoError(t, err)
		require.NotNil(t, res, "seekTo=%v", seekTo)
		require.Equal(t, RowNumber{seekTo, -1, -1, -1, -1, -1, -1, -1}, res.RowNumber)
		require.Equal(t, seekTo, res.ToMap()["A"][0].Int32())
	}
}

func TestColumnIteratorPredicate(t *testing.T) {
	for _, tc := range iterTestCases {
		t.Run(tc.name, func(t *testing.T) {
			testColumnIteratorPredicate(t, tc.makeIter)
		})
	}
}

func testColumnIteratorPredicate(t *testing.T, makeIter makeTestIterFn) {
	count := 10_000
	pf := createTestFile(t, count)

	pred := NewIntBetweenPredicate(7001, 7003)

	idx, _, _ := GetColumnIndexByPath(pf, "A")
	iter := makeIter(pf, idx, pred, "A")
	defer iter.Close()

	expectedResults := []int32{
		7001,
		7002,
		7003,
	}

	for _, expectedResult := range expectedResults {
		res, err := iter.Next()
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, RowNumber{expectedResult, -1, -1, -1, -1, -1, -1, -1}, res.RowNumber)
		require.Equal(t, expectedResult, res.ToMap()["A"][0].Int32())
	}
}

func TestSyncIteratorPropagatesErrors(t *testing.T) {
	type T struct{ A int }

	rows := []T{}
	count := 10_000
	for i := 0; i < count; i++ {
		rows = append(rows, T{i})
	}

	ctx, cancel := context.WithCancel(context.Background())

	pf := createFileWith(t, ctx, rows)

	idx, _, _ := GetColumnIndexByPath(pf, "A")
	iter := NewSyncIterator(ctx, pf.RowGroups(), idx, "", 1, nil, "A", MaxDefinitionLevel)

	_, err := iter.Next()
	require.NoError(t, err)

	cancel()

	// iterate until we get an error and confirm it's because of the context cancellation
	for {
		_, err = iter.Next()
		if err != nil {
			break
		}
	}
	require.ErrorContains(t, err, "context canceled")
}

func BenchmarkColumnIterator(b *testing.B) {
	for _, tc := range iterTestCases {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkColumnIterator(b, tc.makeIter)
		})
	}
}

func benchmarkColumnIterator(b *testing.B, makeIter makeTestIterFn) {
	count := 100_000
	pf := createTestFile(b, count)

	idx, _, _ := GetColumnIndexByPath(pf, "A")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iter := makeIter(pf, idx, nil, "A")
		actualCount := 0
		for {
			res, err := iter.Next()
			require.NoError(b, err)
			if res == nil {
				break
			}
			actualCount++
		}
		iter.Close()
		require.Equal(b, count, actualCount)
	}
}

func createTestFile(t testing.TB, count int) *parquet.File {
	type T struct{ A int }

	rows := []T{}
	for i := 0; i < count; i++ {
		rows = append(rows, T{i})
	}

	pf := createFileWith(t, context.Background(), rows)
	return pf
}

type ctxReaderAt struct {
	readerAt io.ReaderAt
	ctx      context.Context
}

func (r *ctxReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.readerAt.ReadAt(p, off)
}

func createFileWith[T any](t testing.TB, ctx context.Context, rows []T) *parquet.File { //nolint:revive
	f, err := os.CreateTemp(t.TempDir(), "data.parquet")
	require.NoError(t, err)

	half := len(rows) / 2

	w := parquet.NewGenericWriter[T](f)
	_, err = w.Write(rows[0:half])
	require.NoError(t, err)
	require.NoError(t, w.Flush())

	_, err = w.Write(rows[half:])
	require.NoError(t, err)
	require.NoError(t, w.Flush())

	require.NoError(t, w.Close())

	stat, err := f.Stat()
	require.NoError(t, err)

	pf, err := parquet.OpenFile(&ctxReaderAt{
		readerAt: f,
		ctx:      ctx,
	}, stat.Size(), parquet.FileReadMode(parquet.ReadModeSync))
	require.NoError(t, err)

	return pf
}

func TestEqualRowNumber(t *testing.T) {
	r1 := RowNumber{1, 2, 3, 4, 5, 6}
	r2 := RowNumber{1, 2, 3, 5, 7, 9}

	require.True(t, EqualRowNumber(0, r1, r2))
	require.True(t, EqualRowNumber(1, r1, r2))
	require.True(t, EqualRowNumber(2, r1, r2))
	require.False(t, EqualRowNumber(3, r1, r2))
	require.False(t, EqualRowNumber(4, r1, r2))
	require.False(t, EqualRowNumber(5, r1, r2))
}

func BenchmarkEqualRowNumber(b *testing.B) {
	r1 := RowNumber{1, 2, 3, 4, 5, 6, 7, 8}
	r2 := RowNumber{1, 2, 3, 5, 7, 9, 11, 13}

	for d := 0; d <= MaxDefinitionLevel; d++ {
		b.Run(strconv.Itoa(d), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				EqualRowNumber(d, r1, r2)
			}
		})
	}
}

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("no panic")
		}
	}()
	f()
}
