package v2

import (
	"bytes"
	"context"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v0 "github.com/grafana/tempo/tempodb/encoding/v0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexReaderWriter(t *testing.T) {
	numRecords := 10000
	pageSize := 100

	randomRecords := []*common.Record{}

	for i := 0; i < numRecords; i++ {
		id := make([]byte, 16)
		_, err := rand.Read(id)
		require.NoError(t, err)

		rec := &common.Record{
			Start:  rand.Uint64(),
			Length: rand.Uint32(),
			ID:     id,
		}

		randomRecords = append(randomRecords, rec)
	}

	v0.SortRecords(randomRecords)

	indexWriter := NewIndexWriter(pageSize)
	indexBytes, err := indexWriter.Write(randomRecords)
	require.NoError(t, err)

	indexReader, err := NewIndexReader(backend.NewContextReaderWithAllReader(bytes.NewReader(indexBytes)), pageSize, numRecords)

	i := 0
	for i = 0; i < numRecords; i++ {
		actualRecord, err := indexReader.At(context.Background(), i)
		assert.NoError(t, err)
		expectedRecord := randomRecords[i]

		assert.Equal(t, expectedRecord, actualRecord)
	}

	// next read should return nil, nil to indicate end
	actualRecord, err := indexReader.At(context.Background(), i)
	assert.NoError(t, err)
	assert.Nil(t, actualRecord)

	actualRecord, err = indexReader.At(context.Background(), -1)
	assert.NoError(t, err)
	assert.Nil(t, actualRecord)
}
