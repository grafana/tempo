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

func TestFromVersionErrors(t *testing.T) {
	encoding, err := FromVersion("definitely-not-a-real-version")
	assert.Error(t, err)
	assert.Nil(t, encoding)
}

func TestAllVersions(t *testing.T) {
	for _, v := range allEncodings() {
		encoding, err := FromVersion(v.Version())

		require.Equal(t, v.Version(), encoding.Version())
		require.NoError(t, err)

		for _, e := range backend.SupportedEncoding {
			testDataWriterReader(t, v, e)
		}
	}
}

func testDataWriterReader(t *testing.T, v VersionedEncoding, e backend.Encoding) {
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
		dataWriter, err := v.NewDataWriter(buff, e)
		require.NoError(t, err)

		_, err = dataWriter.Write([]byte{0x01}, tc.readerBytes)
		require.NoError(t, err)

		bytesWritten, err := dataWriter.CutPage()
		require.NoError(t, err)

		err = dataWriter.Complete()
		require.NoError(t, err)

		reader := bytes.NewReader(buff.Bytes())
		dataReader, err := v.NewDataReader(backend.NewContextReaderWithAllReader(reader), e)
		require.NoError(t, err)
		defer dataReader.Close()

		actual, _, err := dataReader.Read(context.Background(), []common.Record{
			{
				Start:  0,
				Length: uint32(bytesWritten),
			},
		}, nil, nil)
		require.NoError(t, err)
		require.Len(t, actual, 1)

		i := NewIterator(bytes.NewReader(actual[0]), v.NewObjectReaderWriter())
		defer i.Close()

		id, obj, err := i.Next(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, tc.readerBytes, obj)
		assert.Equal(t, []byte{0x01}, []byte(id))
	}
}
