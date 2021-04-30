package v0

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataReader(t *testing.T) {
	tests := []struct {
		readerBytes   []byte
		records       []common.Record
		expectedBytes [][]byte
		expectedError bool
	}{
		{},
		{
			records: []common.Record{
				{
					Start:  0,
					Length: 1,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []common.Record{
				{
					Start:  0,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x01},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []common.Record{
				{
					Start:  1,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x02},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []common.Record{
				{
					Start:  0,
					Length: 1,
				},
				{
					Start:  1,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x01},
				{0x02},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []common.Record{
				{
					Start:  0,
					Length: 5,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02},
			records: []common.Record{
				{
					Start:  5,
					Length: 5,
				},
			},
			expectedError: true,
		},
		{
			readerBytes: []byte{0x01, 0x02, 0x03},
			records: []common.Record{
				{
					Start:  1,
					Length: 1,
				},
				{
					Start:  2,
					Length: 1,
				},
			},
			expectedBytes: [][]byte{
				{0x02},
				{0x03},
			},
		},
		{
			readerBytes: []byte{0x01, 0x02, 0x03},
			records: []common.Record{
				{
					Start:  0,
					Length: 1,
				},
				{
					Start:  2,
					Length: 1,
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		reader := NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(tc.readerBytes)))
		actual, _, err := reader.Read(context.Background(), tc.records, nil)
		reader.Close()

		if tc.expectedError {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		assert.Equal(t, tc.expectedBytes, actual)
	}
}

func TestWriterReaderNextPage(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	writer := NewDataWriter(buff)

	pageCount := 10
	pages := make([][]byte, 0, pageCount)
	written := make([]uint32, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		data := make([]byte, 200)
		rand.Read(data)
		pages = append(pages, data)

		bytesWritten, err := writer.Write([]byte{0x01}, data)
		require.NoError(t, err)
		_, err = writer.CutPage()
		require.NoError(t, err)
		written = append(written, uint32(bytesWritten))
	}
	err := writer.Complete()
	require.NoError(t, err)

	reader := NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(buff.Bytes())))
	i := 0
	for {
		page, totalLength, err := reader.NextPage(nil)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		_, id, obj, err := staticObject.UnmarshalAndAdvanceBuffer(page)
		require.NoError(t, err)

		assert.Equal(t, pages[i], obj)
		assert.Equal(t, written[i], totalLength)
		assert.Equal(t, []byte{0x01}, []byte(id))
		i++
	}
	assert.Equal(t, pageCount, i)
}
