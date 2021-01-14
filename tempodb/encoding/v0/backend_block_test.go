package v0

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/util"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willf/bloom"
)

func TestBackendBlock(t *testing.T) {
	id := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	object := []byte{0x01, 0x02}

	bloom := bloom.New(1, 1)
	bloom.Add(id)

	bloomBuffer := bytes.Buffer{}
	_, err := bloom.WriteTo(&bloomBuffer)
	require.NoError(t, err)

	objectBuffer := bytes.Buffer{}
	_, err = marshalObjectToWriter(id, object, &objectBuffer)
	require.NoError(t, err)

	record := newRecord()
	record.ID = id
	record.Length = uint32(objectBuffer.Len())
	index, err := MarshalRecords([]*encoding.Record{record})
	require.NoError(t, err)

	tests := []struct {
		name               string
		id                 []byte
		readerError        error
		readerBloom        []byte
		readerIndex        []byte
		readerRange        []byte
		expected           []byte
		expectedBloomReads int32
		expectedBloomBytes int32
		expectedIndexReads int32
		expectedIndexBytes int32
		expectedBlockReads int32
		expectedBlockBytes int32
	}{
		{
			name:        "error",
			id:          id,
			readerError: errors.New("wups"),
		},
		{
			name:               "bloom passes",
			id:                 id,
			readerBloom:        bloomBuffer.Bytes(),
			expectedBloomReads: 1,
			expectedBloomBytes: int32(bloomBuffer.Len()),
			expectedIndexReads: 1,
		},
		{
			name:               "index passes",
			id:                 id,
			readerBloom:        bloomBuffer.Bytes(),
			readerIndex:        index,
			expectedBloomReads: 1,
			expectedBloomBytes: int32(bloomBuffer.Len()),
			expectedIndexReads: 1,
			expectedIndexBytes: int32(len(index)),
			expectedBlockReads: 1,
			expectedBlockBytes: int32(objectBuffer.Len()),
		},
		{
			name:               "obj found",
			id:                 id,
			readerBloom:        bloomBuffer.Bytes(),
			readerIndex:        index,
			readerRange:        objectBuffer.Bytes(),
			expected:           object,
			expectedBloomReads: 1,
			expectedBloomBytes: int32(bloomBuffer.Len()),
			expectedIndexReads: 1,
			expectedIndexBytes: int32(len(index)),
			expectedBlockReads: 1,
			expectedBlockBytes: int32(objectBuffer.Len()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := func(name string, blockID uuid.UUID, tenantID string) ([]byte, error) {
				if tt.readerError != nil {
					return nil, tt.readerError
				}

				if strings.Contains(name, nameBloomPrefix) {
					return tt.readerBloom, nil
				}

				return tt.readerIndex, nil
			}

			mockR := &util.MockReader{
				ReadFn: fn,
				Range:  tt.readerRange,
			}

			findMetrics := encoding.NewFindMetrics()
			block := NewBackendBlock(&backend.BlockMeta{})
			actual, err := block.Find(context.Background(), mockR, tt.id, &findMetrics)

			assert.True(t, errors.Is(err, tt.readerError))
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.expectedBloomReads, findMetrics.BloomFilterReads.Load())
			assert.Equal(t, tt.expectedBloomBytes, findMetrics.BloomFilterBytesRead.Load())
			assert.Equal(t, tt.expectedIndexReads, findMetrics.IndexReads.Load())
			assert.Equal(t, tt.expectedIndexBytes, findMetrics.IndexBytesRead.Load())
			assert.Equal(t, tt.expectedBlockReads, findMetrics.BlockReads.Load())
			assert.Equal(t, tt.expectedBlockBytes, findMetrics.BlockBytesRead.Load())
		})
	}

}
