package encoding

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

var _ Iterator = (*testIterator)(nil)

// testIterator iterates over in-memory contents. Doesn't require tempodb or a block
type testIterator struct {
	ids    []common.ID
	data   [][]byte
	errors []error
	i      int
}

func (i *testIterator) Add(id common.ID, data []byte, err error) {
	i.ids = append(i.ids, id)
	i.data = append(i.data, data)
	i.errors = append(i.errors, err)
}

func (i *testIterator) Next(context.Context) (common.ID, []byte, error) {
	if i.i == len(i.ids) {
		return nil, nil, io.EOF
	}

	id := i.ids[i.i]
	data := i.data[i.i]
	err := i.errors[i.i]
	i.i++

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
		if err == io.EOF {
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

	iter := NewMultiblockIterator(context.TODO(), []Iterator{iterEvens, iterOdds}, 10, nil, "")

	count := 0
	lastID := -1
	for {
		id, _, err := iter.Next(context.TODO())
		if err == io.EOF {
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

	inner := &testIterator{}
	for i := 0; i < recordCount; i++ {
		inner.Add(make([]byte, i), make([]byte, i), nil)
	}

	// Create iterator and close it after 100ms
	iter := NewMultiblockIterator(context.TODO(), []Iterator{inner}, recordCount/2, nil, "")
	time.Sleep(100 * time.Millisecond)
	iter.Close()

	// Exhaust iterator and verify fewer than recordcount records are received.
	count := 0
	for {
		_, _, err := iter.Next(context.TODO())
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		count++
	}

	require.Less(t, count, recordCount)
}

func TestMultiblockIteratorCanBeCancelledMultipleTimes(t *testing.T) {
	inner := &testIterator{}

	iter := NewMultiblockIterator(context.TODO(), []Iterator{inner}, 1, nil, "")

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

	iter := NewMultiblockIterator(ctx, []Iterator{inner, inner2}, 10, nil, "")

	_, _, err := iter.Next(ctx)
	require.NoError(t, err)

	_, _, err = iter.Next(ctx)

	require.Equal(t, io.ErrClosedPipe, err)
}
