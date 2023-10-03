package v2

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

type mockIterator struct {
	ids  []common.ID
	objs [][]byte
}

func (i *mockIterator) NextBytes(context.Context) (common.ID, []byte, error) {
	if len(i.ids) == 0 {
		return nil, nil, io.EOF
	}

	id := i.ids[0]
	i.ids = i.ids[1:]
	obj := i.objs[0]
	i.objs = i.objs[1:]

	return id, obj, nil
}
func (i *mockIterator) Close() {}

type mockCombiner struct{}

func (*mockCombiner) Combine(_ string, objs ...[]byte) ([]byte, bool, error) {
	var ret []byte
	for _, obj := range objs {
		ret = append(ret, obj...)
	}
	return ret, false, nil
}

func TestEmptyNestedIterator(t *testing.T) {
	r := bytes.NewReader([]byte{})
	i := NewIterator(r, NewObjectReaderWriter())

	id, obj, err := i.NextBytes(context.Background())
	assert.Nil(t, id)
	assert.Nil(t, obj)
	assert.Equal(t, io.EOF, err)
}

func TestDedupingIterator(t *testing.T) {
	tests := []struct {
		ids          []common.ID
		objs         [][]byte
		expectedIDs  []common.ID
		expectedObjs [][]byte
	}{
		// nothing!
		{},
		// one object
		{
			ids:          []common.ID{{0x01}},
			objs:         [][]byte{{0x01}},
			expectedIDs:  []common.ID{{0x01}},
			expectedObjs: [][]byte{{0x01}},
		},
		// two objects
		{
			ids:          []common.ID{{0x01}, {0x02}},
			objs:         [][]byte{{0x01}, {0x02}},
			expectedIDs:  []common.ID{{0x01}, {0x02}},
			expectedObjs: [][]byte{{0x01}, {0x02}},
		},
		// combines stuff!
		{
			ids:          []common.ID{{0x01}, {0x01}},
			objs:         [][]byte{{0x01}, {0x01}},
			expectedIDs:  []common.ID{{0x01}},
			expectedObjs: [][]byte{{0x01, 0x01}},
		},
		// combines a bunch of stuff!
		{
			ids:          []common.ID{{0x01}, {0x01}, {0x01}, {0x01}, {0x02}, {0x02}, {0x02}, {0x02}},
			objs:         [][]byte{{0x01}, {0x01}, {0x01}, {0x01}, {0x02}, {0x02}, {0x02}, {0x02}},
			expectedIDs:  []common.ID{{0x01}, {0x02}},
			expectedObjs: [][]byte{{0x01, 0x01, 0x01, 0x01}, {0x02, 0x02, 0x02, 0x02}},
		},
		// only works with ordered input
		{
			ids:          []common.ID{{0x01}, {0x02}, {0x01}},
			objs:         [][]byte{{0x01}, {0x02}, {0x01}},
			expectedIDs:  []common.ID{{0x01}, {0x02}, {0x01}},
			expectedObjs: [][]byte{{0x01}, {0x02}, {0x01}},
		},
		// rando
		{
			ids:          []common.ID{{0x01}, {0x02}, {0x02}, {0x03}, {0x03}, {0x03}, {0x04}, {0x05}},
			objs:         [][]byte{{0x01}, {0x02}, {0x02}, {0x03}, {0x03}, {0x03}, {0x04}, {0x05}},
			expectedIDs:  []common.ID{{0x01}, {0x02}, {0x03}, {0x04}, {0x05}},
			expectedObjs: [][]byte{{0x01}, {0x02, 0x02}, {0x03, 0x03, 0x03}, {0x04}, {0x05}},
		},
	}

	for _, tc := range tests {
		iter, err := NewDedupingIterator(&mockIterator{ids: tc.ids, objs: tc.objs}, &mockCombiner{}, "")
		require.NoError(t, err)

		var actualIDs []common.ID
		var actualObjs [][]byte

		for {
			id, obj, err := iter.NextBytes(context.Background())
			if errors.Is(err, io.EOF) {
				break
			}
			assert.NoError(t, err)
			actualIDs = append(actualIDs, id)
			actualObjs = append(actualObjs, obj)
		}

		assert.Equal(t, tc.expectedIDs, actualIDs)
		assert.Equal(t, tc.expectedObjs, actualObjs)
	}
}
