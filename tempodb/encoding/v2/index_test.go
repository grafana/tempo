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

func TestIndexWriterReader(t *testing.T) {
	numRecords := 100000
	pageSize := 210

	randomRecords := randomOrderedRecords(t, numRecords)
	indexWriter := NewIndexWriter(pageSize)
	indexBytes, err := indexWriter.Write(randomRecords)
	require.NoError(t, err)

	indexReader, err := NewIndexReader(backend.NewContextReaderWithAllReader(bytes.NewReader(indexBytes)), pageSize, numRecords)
	require.NoError(t, err)

	i := 0
	for i = 0; i < numRecords; i++ {
		expectedRecord := randomRecords[i]

		actualRecord, err := indexReader.At(context.Background(), i)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecord, actualRecord)

		actualRecord, actualIdx, err := indexReader.Find(context.Background(), expectedRecord.ID)
		assert.NoError(t, err)
		assert.Equal(t, i, actualIdx)
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

func TestIndexHeaderChecksum(t *testing.T) {
	recordsPerPage := 4
	numRecords := 4
	pageSize := int(baseHeaderSize) + IndexHeaderLength + NewRecordReaderWriter().RecordLength()*recordsPerPage

	randomRecords := randomOrderedRecords(t, numRecords)
	indexWriter := NewIndexWriter(pageSize)
	indexBytes, err := indexWriter.Write(randomRecords)
	require.NoError(t, err)

	indexBytes[len(indexBytes)-1]++

	r, err := NewIndexReader(backend.NewContextReaderWithAllReader(bytes.NewReader(indexBytes)), pageSize, numRecords)
	assert.NoError(t, err)
	_, err = r.(*indexReader).getPage(context.Background(), 0)
	assert.Error(t, err)
}

func randomOrderedRecords(t *testing.T, num int) []*common.Record {
	randomRecords := []*common.Record{}

	for i := 0; i < num; i++ {
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

	return randomRecords
}
