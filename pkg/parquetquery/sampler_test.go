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

func TestSyncIteratorSampler(t *testing.T) {
	buf := new(bytes.Buffer)
	w := parquet.NewWriter(buf)
	const n = 6
	for i := 0; i < n; i++ {
		require.NoError(t, w.Write(&testInt{int64(i)}))
	}
	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())
	r, err := parquet.OpenFile(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	s := &fakeSampler{keepEvery: 2}
	it := NewSyncIterator(context.TODO(), r.RowGroups(), 0,
		SyncIteratorOptSelectAs("v"),
		SyncIteratorOptSampler(s))
	defer it.Close()

	var got []int64
	for {
		res, err := it.Next()
		require.NoError(t, err)
		if res == nil {
			break
		}
		got = append(got, res.Entries[0].Value.Int64())
	}

	// Sampler was offered every value that passed the (absent) predicate...
	require.Equal(t, n, s.seen, "Sample called once per value")
	// ...and Expect was told the page population.
	require.Equal(t, uint64(n), s.expected, "Expect called with page value count")
	// keepEvery=2 keeps the 1st, 3rd, 5th offered values (0-indexed 0,2,4).
	require.Equal(t, []int64{0, 2, 4}, got, "only sampled values returned")
}
