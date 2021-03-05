package v2

import (
	"bytes"
	"context"
	"math/rand"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/base"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexWriterReaderAt(t *testing.T) {
	numRecords := 100000
	pageSize := 210

	randomRecords := randomOrderedRecords(t, numRecords)
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

func TestIndexHeaderMinMax(t *testing.T) {
	recordsPerPage := 4
	numRecords := 20
	pageSize := int(baseHeaderSize) + IndexHeaderLength + base.RecordLength*recordsPerPage

	randomRecords := randomOrderedRecords(t, numRecords)
	indexWriter := NewIndexWriter(pageSize)
	indexBytes, err := indexWriter.Write(randomRecords)
	require.NoError(t, err)

	r, err := NewIndexReader(backend.NewContextReaderWithAllReader(bytes.NewReader(indexBytes)), pageSize, numRecords)

	numPages := numRecords / recordsPerPage
	minID := common.ID(constMinID)
	// fill cache
	for i := 0; i < numPages; i++ {
		page, err := r.(*indexReader).getPage(context.Background(), i)
		require.NoError(t, err)
		maxID := randomRecords[(i+1)*recordsPerPage-1].ID
		assert.Equal(t, minID, page.header.(*indexHeader).minID)
		assert.Equal(t, maxID, page.header.(*indexHeader).maxID)
		minID = randomRecords[i*recordsPerPage].ID
	}
}

func TestIndexHeaderChecksum(t *testing.T) {
	recordsPerPage := 4
	numRecords := 4
	pageSize := int(baseHeaderSize) + IndexHeaderLength + base.RecordLength*recordsPerPage

	randomRecords := randomOrderedRecords(t, numRecords)
	indexWriter := NewIndexWriter(pageSize)
	indexBytes, err := indexWriter.Write(randomRecords)
	require.NoError(t, err)

	indexBytes[len(indexBytes)-1]++

	r, err := NewIndexReader(backend.NewContextReaderWithAllReader(bytes.NewReader(indexBytes)), pageSize, numRecords)
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

	base.SortRecords(randomRecords)

	return randomRecords
}
