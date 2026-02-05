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
		return NewSyncIterator(context.TODO(), pf.RowGroups(), idx, SyncIteratorOptSelectAs(selectAs), SyncIteratorOptPredicate(filter), SyncIteratorOptMaxDefinitionLevel(MaxDefinitionLevel))
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

	iter := NewSyncIterator(ctx, pf.RowGroups(), 0)

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

func BenchmarkIteratorResultAppend(b *testing.B) {
	testCases := []struct {
		name        string
		inputResult *IteratorResult
	}{
		{
			name:        "empty",
			inputResult: &IteratorResult{},
		},
		{
			name: "both results",
			inputResult: &IteratorResult{
				Entries: []struct {
					Key   string
					Value parquet.Value
				}{
					{Key: "A", Value: parquet.Int32Value(1)},
				},
				OtherEntries: []struct {
					Key   string
					Value interface{}
				}{
					{Key: "B", Value: "test"},
				},
			},
		},
		{
			name: "only entries",
			inputResult: &IteratorResult{
				Entries: []struct {
					Key   string
					Value parquet.Value
				}{
					{Key: "A", Value: parquet.Int32Value(1)},
				},
			},
		},
	}

	ir := IteratorResult{}
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ir.Reset()
				ir.Append(tc.inputResult)
			}
		})
	}
}

type testIterator struct {
	rows []IteratorResult
	i    int
}

var _ Iterator = (*testIterator)(nil)

func (t *testIterator) Next() (*IteratorResult, error) {
	if t.i >= len(t.rows) {
		return nil, nil
	}

	r := &t.rows[t.i]
	t.i++
	return r, nil
}

func (t *testIterator) SeekTo(to RowNumber, d int) (*IteratorResult, error) {
	for j := t.i; j < len(t.rows); j++ {
		if EqualRowNumber(d, t.rows[j].RowNumber, to) {
			t.i = j + 1
			return &t.rows[j], nil
		}
	}
	return nil, nil
}

func (t *testIterator) Close() {
}

func (t *testIterator) String() string {
	return "testIterator"
}

func TestLeftJoinUp(t *testing.T) {
	// Defined at level 0
	iter1 := &testIterator{rows: []IteratorResult{
		testResult("x", "A", 1),
		testResult("x", "B", 2),
	}}

	// Defined at level 1
	iter2 := &testIterator{rows: []IteratorResult{
		testResult("y", "A", 1, 0),
		testResult("y", "B", 1, 1),
		testResult("y", "C", 2, 0),
	}}

	expected := []IteratorResult{
		{
			RowNumber: rn(1),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "x", Value: "A"},
				{Key: "y", Value: "A"},
				{Key: "y", Value: "B"},
			},
		},
		{
			RowNumber: rn(2),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "x", Value: "B"},
				{Key: "y", Value: "C"},
			},
		},
	}

	got := []IteratorResult{}

	j, err := NewLeftJoinIterator(0, []Iterator{iter1, iter2}, nil, nil)
	require.NoError(t, err)

	for {
		res, err := j.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
		got = append(got, clone(res))
	}

	require.Equal(t, expected, got)
}

func TestLeftJoinDown(t *testing.T) {
	iter1DefLevel := 2
	iter1 := &testIterator{rows: []IteratorResult{
		testResult("x", "A", 0, 0, 1),
		testResult("x", "B", 0, 0, 2),
		testResult("x", "C", 0, 0, 3),
	}}

	// Defined at level iter2DefLevel
	iter2DefLevel := 3
	iter2 := &testIterator{rows: []IteratorResult{
		testResult("y", "A1", 0, 0, 1, 0),
		testResult("y", "A2", 0, 0, 1, 1),
		testResult("y", "B1", 0, 0, 2, 0),
		testResult("y", "B2", 0, 0, 2, 1),
		testResult("y", "C", 0, 0, 3, 0),
	}}

	var buf IteratorResult

	testCollector := &testCollector{
		reset: func(rowNumber RowNumber) {
			buf.Reset()
			buf.RowNumber = rowNumber
		},
		collect: func(r *IteratorResult, _ any) {
			buf.Append(r)
		},
		result: func() *IteratorResult {
			return &buf
		},
	}

	expected := []IteratorResult{
		{
			RowNumber: rn(0, 0, 1, 0),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "y", Value: "A1"},
				{Key: "x", Value: "A"},
			},
		},
		{
			RowNumber: rn(0, 0, 1, 1),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "y", Value: "A2"},
			},
		},
		{
			RowNumber: rn(0, 0, 2, 0),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "y", Value: "B1"},
				{Key: "x", Value: "B"},
			},
		},
		{
			RowNumber: rn(0, 0, 2, 1),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "y", Value: "B2"},
			},
		},
		{
			RowNumber: rn(0, 0, 3, 0),
			OtherEntries: []struct {
				Key   string
				Value interface{}
			}{
				{Key: "y", Value: "C"},
				{Key: "x", Value: "C"},
			},
		},
	}

	got := []IteratorResult{}

	j, err := NewLeftJoinIterator(1, nil, nil, nil,
		// Primary iterator is down to level 1.
		WithIterator(iter2DefLevel, iter2, false, nil),
		WithIterator(iter1DefLevel, iter1, true, nil),
		WithCollector(testCollector),
	)
	require.NoError(t, err)

	for {
		res, err := j.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
		got = append(got, clone(res))
	}

	require.Equal(t, expected, got)
}

func rn(rowNumbers ...int32) RowNumber {
	rn := EmptyRowNumber()
	copy(rn[:], rowNumbers)
	return rn
}

func testResult(key, value string, rowNumbers ...int32) IteratorResult {
	return IteratorResult{
		RowNumber: rn(rowNumbers...),
		OtherEntries: []struct {
			Key   string
			Value interface{}
		}{
			{Key: key, Value: value},
		},
	}
}

func clone(r *IteratorResult) IteratorResult {
	o := IteratorResult{
		RowNumber: r.RowNumber,
	}
	o.Append(r)
	return o
}

type testCollector struct {
	reset   func(rowNumber RowNumber)
	collect func(r *IteratorResult, param any)
	result  func() *IteratorResult
}

var _ Collector = (*testCollector)(nil)

func (c *testCollector) Reset(rowNumber RowNumber) {
	c.reset(rowNumber)
}

func (c *testCollector) Collect(r *IteratorResult, param any) {
	c.collect(r, param)
}

func (c *testCollector) Result() *IteratorResult {
	return c.result()
}

func (c *testCollector) Close() {}
