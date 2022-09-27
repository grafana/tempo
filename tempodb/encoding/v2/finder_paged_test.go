package v2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedFinder relies on the record finder to return the index of the first record if there are multiple
// with the same id
func TestCommonRecordsFind(t *testing.T) {
	id := []byte{0x01}
	recs := []Record{
		{
			ID:    id,
			Start: 0,
		},
		{
			ID:    id,
			Start: 1,
		},
		{
			ID:    id,
			Start: 2,
		},
		{
			ID:    id,
			Start: 2,
		},
	}

	rec, i, err := Records(recs).Find(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), rec.Start)
	assert.Equal(t, 0, i)
}
