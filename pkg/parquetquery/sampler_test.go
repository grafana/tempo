package parquetquery

import (
	"bytes"
	"context"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"
)

// fakeSampler keeps every keepEvery-th value that passes the predicate and records
// the population reported via Expect.
type fakeSampler struct {
	keepEvery int
	seen      int
	expected  uint64
}

func (s *fakeSampler) Expect(count uint64) { s.expected += count }

func (s *fakeSampler) Sample() bool {
	s.seen++
	return s.seen%s.keepEvery == 1
}

// writeInts writes the given int64 values as testInt rows. Each break index forces
// a Flush after that many rows, which parquet-go emits as a separate row group (one
// page each here) — letting callers exercise per-page sampler bookkeeping.
func writeInts(t *testing.T, values []int64, breaks ...int) []parquet.RowGroup {
	t.Helper()
	breakAt := map[int]bool{}
	for _, b := range breaks {
		breakAt[b] = true
	}

	buf := new(bytes.Buffer)
	w := parquet.NewWriter(buf)
	for i, v := range values {
		require.NoError(t, w.Write(&testInt{v}))
		if breakAt[i+1] {
			require.NoError(t, w.Flush())
		}
	}
	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())

	r, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return r.RowGroups()
}

func drainInt64(t *testing.T, it Iterator) []int64 {
	t.Helper()
	var got []int64
	for {
		res, err := it.Next()
		require.NoError(t, err)
		if res == nil {
			return got
		}
		got = append(got, res.Entries[0].Value.Int64())
	}
}

func TestSyncIteratorSampler(t *testing.T) {
	const n = 6
	rgs := writeInts(t, []int64{0, 1, 2, 3, 4, 5})

	s := &fakeSampler{keepEvery: 2}
	it := NewSyncIterator(context.TODO(), rgs, 0,
		SyncIteratorOptSelectAs("v"),
		SyncIteratorOptSampler(s))
	defer it.Close()

	got := drainInt64(t, it)

	// Sampler was offered every value that passed the (absent) predicate...
	require.Equal(t, n, s.seen, "Sample called once per value")
	// ...and Expect was told the page population.
	require.Equal(t, uint64(n), s.expected, "Expect called with page value count")
	// keepEvery=2 keeps the 1st, 3rd, 5th offered values (0-indexed 0,2,4).
	require.Equal(t, []int64{0, 2, 4}, got, "only sampled values returned")
}

// TestSyncIteratorSamplerAfterPredicate pins the ordering that motivated moving
// sampling off the predicate: Sample() must only be offered values that already
// passed the predicate, while Expect() still reports the full (unfiltered) page
// population so the sampler can reason about the whole dataset.
func TestSyncIteratorSamplerAfterPredicate(t *testing.T) {
	const n = 8
	rgs := writeInts(t, []int64{0, 1, 2, 3, 4, 5, 6, 7})

	// Predicate keeps [2,6] inclusive: 2,3,4,5,6 (5 values).
	s := &fakeSampler{keepEvery: 2}
	it := NewSyncIterator(context.TODO(), rgs, 0,
		SyncIteratorOptSelectAs("v"),
		SyncIteratorOptPredicate(NewIntBetweenPredicate(2, 6)),
		SyncIteratorOptSampler(s))
	defer it.Close()

	got := drainInt64(t, it)

	// Sample() is called only on the 5 values that passed the predicate, not all 8.
	require.Equal(t, 5, s.seen, "Sample offered only predicate-passing values")
	// Expect() still sees the whole page population, not the filtered count.
	require.Equal(t, uint64(n), s.expected, "Expect reports unfiltered page count")
	// keepEvery=2 over the passing values 2,3,4,5,6 keeps the 1st/3rd/5th: 2,4,6.
	require.Equal(t, []int64{2, 4, 6}, got, "sampling applied after the predicate")
}

// TestSyncIteratorSamplerMultiPage verifies Expect() is called once per accepted
// page (accumulating the total population) and that sampling carries across page
// boundaries rather than resetting per page. Two row groups, one page each.
func TestSyncIteratorSamplerMultiPage(t *testing.T) {
	const n = 6
	// Break after index 3 -> two row groups (one page each): [0,1,2] and [3,4,5].
	rgs := writeInts(t, []int64{0, 1, 2, 3, 4, 5}, 3)
	require.Len(t, rgs, 2, "break should produce two row groups")

	s := &fakeSampler{keepEvery: 2}
	it := NewSyncIterator(context.TODO(), rgs, 0,
		SyncIteratorOptSelectAs("v"),
		SyncIteratorOptSampler(s))
	defer it.Close()

	got := drainInt64(t, it)

	require.Equal(t, n, s.seen, "Sample called once per value across pages")
	// Expect fires per page; the two pages sum to the full population.
	require.Equal(t, uint64(n), s.expected, "Expect accumulated across pages")
	// The sampler's counter is not reset at the page boundary, so the kept set is
	// the 1st/3rd/5th values overall: 0,2,4.
	require.Equal(t, []int64{0, 2, 4}, got, "sampling continues across page boundary")
}

// TestSyncIteratorNoSampler confirms the default (nil sampler) path returns every
// value untouched.
func TestSyncIteratorNoSampler(t *testing.T) {
	rgs := writeInts(t, []int64{0, 1, 2, 3, 4, 5})

	it := NewSyncIterator(context.TODO(), rgs, 0,
		SyncIteratorOptSelectAs("v"))
	defer it.Close()

	require.Equal(t, []int64{0, 1, 2, 3, 4, 5}, drainInt64(t, it))
}
