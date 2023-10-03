package vparquet

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type intIterator []*uint8 // making this a pointer makes the below code a little gross but allows multiblockiterator to treat all types as nillable

func (i *intIterator) Next(_ context.Context) (common.ID, *uint8, error) {
	s := *i
	if len(s) == 0 {
		return nil, nil, io.EOF
	}
	ret := s[0]
	*i = s[1:]
	return []byte{*ret}, ret, nil
}

func (i *intIterator) Close() {}

func (i *intIterator) peekNextID(_ context.Context) (common.ID, error) { //nolint:unused //this is being marked as unused, but it's literally used about 30 lines south
	s := *i
	if len(s) == 0 {
		return nil, io.EOF
	}

	return []byte{*s[0]}, nil
}

func TestMultiBlockIterator(t *testing.T) {
	ptr := func(n uint8) *uint8 {
		return &n
	}

	tcs := []struct {
		iters    []*intIterator
		expected []uint8
	}{
		{
			iters:    []*intIterator{{}},
			expected: []uint8{},
		},
		{
			iters:    []*intIterator{{}, {}},
			expected: []uint8{},
		},
		{
			iters:    []*intIterator{{ptr(1), ptr(2), ptr(3)}, {ptr(4), ptr(5), ptr(6)}},
			expected: []uint8{1, 2, 3, 4, 5, 6},
		},
		{
			iters:    []*intIterator{{ptr(1), ptr(3), ptr(5)}, {ptr(2), ptr(4), ptr(6)}},
			expected: []uint8{1, 2, 3, 4, 5, 6},
		},
		{
			iters:    []*intIterator{{ptr(1), ptr(3)}, {ptr(2), ptr(6)}, {ptr(4), ptr(5)}},
			expected: []uint8{1, 2, 3, 4, 5, 6},
		},
		{
			iters:    []*intIterator{{ptr(1), ptr(3)}, {ptr(1), ptr(6)}, {ptr(4), ptr(6)}},
			expected: []uint8{1, 3, 4, 6},
		},
	}

	for _, tc := range tcs {
		bookmarks := make([]*bookmark[*uint8], 0, len(tc.iters))
		for _, iter := range tc.iters {
			bookmarks = append(bookmarks, newBookmark[*uint8](iter))
		}

		mbi := newMultiblockIterator(bookmarks, func(i []*uint8) (*uint8, error) {
			return i[0], nil
		})
		defer mbi.Close()

		ctx := context.Background()
		for {
			id, val, err := mbi.Next(ctx)
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			require.Equal(t, common.ID([]byte{tc.expected[0]}), id)
			require.Equal(t, tc.expected[0], *val)
			tc.expected = tc.expected[1:]
		}
		require.Len(t, tc.expected, 0)
	}
}
