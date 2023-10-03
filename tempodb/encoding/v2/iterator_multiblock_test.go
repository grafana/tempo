package v2

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

var testLogger log.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

var _ BytesIterator = (*testIterator)(nil)

// testIterator iterates over in-memory contents. Doesn't require tempodb or a block
type testIterator struct {
	ids    []common.ID
	data   [][]byte
	errors []error
	i      atomic.Int32
}

func (i *testIterator) Add(id common.ID, data []byte, err error) {
	i.ids = append(i.ids, id)
	i.data = append(i.data, data)
	i.errors = append(i.errors, err)
}

func (i *testIterator) NextBytes(context.Context) (common.ID, []byte, error) {
	idx := int(i.i.Load())

	if idx == len(i.ids) {
		return nil, nil, io.EOF
	}

	id := i.ids[idx]
	data := i.data[idx]
	err := i.errors[idx]
	i.i.Inc()

	return id, data, err
}

func (*testIterator) Close() {}

func TestBookmarkIteration(t *testing.T) {
	recordCount := 100
	iter := &testIterator{}
	for i := 0; i < recordCount; i++ {
		iter.Add([]byte{uint8(i)}, []byte{uint8(i)}, nil)
	}

	bm := newBookmark(iter)

	i := 0
	for {
		id, data, err := bm.current(context.Background())
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		require.Equal(t, uint8(i), id[0])
		require.Equal(t, uint8(i), data[0])

		i++
		bm.clear()
	}
	require.Equal(t, recordCount, i)
}

func TestMultiblockSorts(t *testing.T) {
	iterEvens := &testIterator{}
	iterEvens.Add([]byte{0}, []byte{0}, nil)
	iterEvens.Add([]byte{2}, []byte{2}, nil)
	iterEvens.Add([]byte{4}, []byte{4}, nil)

	iterOdds := &testIterator{}
	iterOdds.Add([]byte{1}, []byte{1}, nil)
	iterOdds.Add([]byte{3}, []byte{3}, nil)
	iterOdds.Add([]byte{5}, []byte{5}, nil)

	iter := NewMultiblockIterator(context.TODO(), []BytesIterator{iterEvens, iterOdds}, 10, &mockCombiner{}, "", testLogger)

	count := 0
	lastID := -1
	for {
		id, _, err := iter.NextBytes(context.TODO())
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		i := int(id[0])

		require.Equal(t, lastID+1, i)
		lastID = i
		count++
	}

	require.Equal(t, 6, count)
}

func TestMultiblockIteratorCanBeCancelled(t *testing.T) {
	recordCount := 100

	testCases := []struct {
		name   string
		close  bool
		cancel bool
	}{
		{
			name:  "close iterator",
			close: true,
		},
		{
			name:   "cancel context",
			cancel: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &testIterator{}
			for i := 0; i < recordCount; i++ {
				inner.Add(make([]byte, i), make([]byte, i), nil)
			}

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)

			// Create iterator and cancel/close it after 100ms
			iter := NewMultiblockIterator(ctx, []BytesIterator{inner}, recordCount/2, &mockCombiner{}, "", testLogger)
			time.Sleep(100 * time.Millisecond)
			if tc.close {
				iter.Close()
			}
			if tc.cancel {
				cancel()
			}

			// Exhaust iterator and verify fewer than recordcount records are received.
			count := 0
			for {
				_, _, err := iter.NextBytes(context.TODO())
				if err != nil {
					break
				}
				count++
			}

			require.Less(t, count, recordCount)
			cancel()
		})
	}
}

func TestMultiblockIteratorCanBeCancelledMultipleTimes(*testing.T) {
	inner := &testIterator{}

	iter := NewMultiblockIterator(context.TODO(), []BytesIterator{inner}, 1, &mockCombiner{}, "", testLogger)

	iter.Close()
	iter.Close()
	iter.Close()
}

func TestMultiblockIteratorPropogatesErrors(t *testing.T) {
	ctx := context.TODO()

	inner := &testIterator{}
	inner.Add([]byte{1}, []byte{1}, nil)
	inner.Add(nil, nil, io.ErrClosedPipe)

	inner2 := &testIterator{}
	inner2.Add([]byte{2}, []byte{2}, nil)
	inner2.Add([]byte{3}, []byte{3}, nil)

	iter := NewMultiblockIterator(ctx, []BytesIterator{inner, inner2}, 10, &mockCombiner{}, "", testLogger)

	_, _, err := iter.NextBytes(ctx)
	require.NoError(t, err)

	_, _, err = iter.NextBytes(ctx)

	require.Equal(t, io.ErrClosedPipe, err)
}

func TestMultiblockIteratorSkipsEmptyObjects(t *testing.T) {
	ctx := context.TODO()

	// Empty objects a beginning, middle, and end.
	inner := &testIterator{}
	inner.Add([]byte{1}, []byte{}, nil)
	inner.Add([]byte{2}, []byte{2}, nil)
	inner.Add([]byte{3}, []byte{3}, nil)
	inner.Add([]byte{4}, []byte{}, nil) // Two empties in a row
	inner.Add([]byte{5}, []byte{}, nil)
	inner.Add([]byte{6}, []byte{6}, nil)
	inner.Add([]byte{7}, []byte{7}, nil)
	inner.Add([]byte{8}, []byte{}, nil)

	expected := []struct {
		id  common.ID
		obj []byte
		err error
	}{
		{[]byte{2}, []byte{2}, nil},
		{[]byte{3}, []byte{3}, nil},
		{[]byte{6}, []byte{6}, nil},
		{[]byte{7}, []byte{7}, nil},
		{nil, nil, io.EOF},
	}

	iter := NewMultiblockIterator(ctx, []BytesIterator{inner}, 10, &mockCombiner{}, "", testLogger)
	for i := 0; i < len(expected); i++ {
		id, obj, err := iter.NextBytes(ctx)
		require.Equal(t, expected[i].err, err)
		require.Equal(t, expected[i].id, id)
		require.Equal(t, expected[i].obj, obj)
	}
}
