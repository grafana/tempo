package v2

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
)

func TestPrefetchIterates(t *testing.T) {
	tests := []struct {
		ids  []common.ID
		objs [][]byte
		err  error
	}{
		{
			ids: []common.ID{
				{0x01},
				{0x02},
				{0x03},
			},
			objs: [][]byte{
				{0x05},
				{0x06},
				{0x06},
			},
			err: nil,
		},
		{
			ids: []common.ID{
				{0x01},
				{0x02},
				{0x03},
			},
			objs: [][]byte{
				{0x05},
				{0x06},
				{0x06},
			},
			err: errors.New("wups"),
		},
	}

	ctx := context.Background()
	for _, tc := range tests {
		iter := &testIterator{}
		for i := 0; i < len(tc.ids); i++ {
			iter.Add(tc.ids[i], tc.objs[i], tc.err)
		}
		prefetchIter := NewPrefetchIterator(ctx, iter, 10)

		count := 0
		for {
			id, obj, err := prefetchIter.NextBytes(context.TODO())
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				assert.Equal(t, tc.err, err)
				break
			}
			assert.Equal(t, tc.ids[count], id)
			assert.Equal(t, tc.objs[count], obj)
			count++
		}

		if tc.err == nil {
			assert.Equal(t, len(tc.ids), count)
		}
	}
}

func TestPrefetchPrefetches(t *testing.T) {
	ctx := context.Background()

	iter := &testIterator{}
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)

	_ = NewPrefetchIterator(ctx, iter, 10)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(3), iter.i.Load()) // prefetch all 3

	iter = &testIterator{}
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)
	iter.Add([]byte{0x01}, []byte{0x01}, nil)

	prefetchIter := NewPrefetchIterator(ctx, iter, 3)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(4), iter.i.Load()) // prefetch only the buffer. this happens to be 1 more than the passed buffer. maybe one day we will "correct" that
	_, _, _ = prefetchIter.NextBytes(ctx)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(5), iter.i.Load()) // get all
}
