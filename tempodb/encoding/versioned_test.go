package encoding

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllVersions(t *testing.T) {
	for _, v := range allEncodings() {
		for _, e := range backend.SupportedEncoding {
			testDataWriterReader(t, v, e)
		}
	}
}

func testDataWriterReader(t *testing.T, v versionedEncoding, e backend.Encoding) {
	tests := []struct {
		readerBytes []byte
	}{
		{
			readerBytes: []byte{0x01, 0x02},
		},
		{
			readerBytes: []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	for _, tc := range tests {
		buff := bytes.NewBuffer([]byte{})
		dataWriter, err := v.newDataWriter(buff, e)
		require.NoError(t, err)

		_, err = dataWriter.Write([]byte{0x01}, tc.readerBytes)
		require.NoError(t, err)

		bytesWritten, err := dataWriter.CutPage()
		require.NoError(t, err)

		err = dataWriter.Complete()
		require.NoError(t, err)

		reader := bytes.NewReader(buff.Bytes())
		dataReader, err := v.newDataReader(backend.NewContextReaderWithAllReader(reader), e)
		require.NoError(t, err)
		defer dataReader.Close()

		actual, err := dataReader.Read(context.Background(), []*common.Record{
			{
				Start:  0,
				Length: uint32(bytesWritten),
			},
		})
		require.NoError(t, err)
		require.Len(t, actual, 1)

		i := NewIterator(bytes.NewReader(actual[0]))
		defer i.Close()

		id, obj, err := i.Next(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, tc.readerBytes, obj)
		assert.Equal(t, []byte{0x01}, []byte(id))
	}
}
