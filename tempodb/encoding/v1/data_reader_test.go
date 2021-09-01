package v1

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllEncodings(t *testing.T) {
	tests := []struct {
		readerBytes []byte
	}{
		{
			readerBytes: []byte{0x01, 0x02},
		},
	}

	for _, tc := range tests {
		for _, enc := range backend.SupportedEncoding {
			t.Run(enc.String(), func(t *testing.T) {
				wPool, err := GetWriterPool(enc)
				require.NoError(t, err)

				buff := bytes.NewBuffer([]byte{})
				mw := &meteredWriter{
					wrappedWriter: buff,
				}
				writer, err := wPool.GetWriter(mw)
				require.NoError(t, err)

				_, err = writer.Write(tc.readerBytes)
				require.NoError(t, err)
				err = writer.Close()
				require.NoError(t, err)

				encryptedBytes := buff.Bytes()
				reader, err := NewDataReader(backend.NewContextReaderWithAllReader(bytes.NewReader(encryptedBytes)), enc)
				require.NoError(t, err)
				defer reader.Close()

				actual, _, err := reader.Read(context.Background(), []common.Record{
					{
						Start:  0,
						Length: uint32(mw.bytesWritten),
					},
				}, nil, nil)

				assert.NoError(t, err)
				assert.Len(t, actual, 1)
				assert.Equal(t, tc.readerBytes, actual[0])
			})
		}
	}
}
