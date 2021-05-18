package backend

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
)

func TestBlockMeta(t *testing.T) {
	testVersion := "blerg"
	testEncoding := EncLZ4_256k
	testDataEncoding := "blarg"

	id := uuid.New()
	b := NewBlockMeta(testTenantID, id, testVersion, testEncoding, testDataEncoding)

	assert.Equal(t, id, b.BlockID)
	assert.Equal(t, testTenantID, b.TenantID)
	assert.Equal(t, testVersion, b.Version)
	assert.Equal(t, testEncoding, b.Encoding)
	assert.Equal(t, testDataEncoding, b.DataEncoding)

	randID1 := make([]byte, 10)
	randID2 := make([]byte, 10)

	rand.Read(randID1)
	rand.Read(randID2)

	assert.Equal(t, b.StartTime, b.EndTime)

	b.ObjectAdded(randID1)
	b.ObjectAdded(randID2)
	assert.True(t, b.EndTime.After(b.StartTime))
	assert.Equal(t, 1, bytes.Compare(b.MaxID, b.MinID))
	assert.Equal(t, 2, b.TotalObjects)
}

func TestBlockMetaParsing(t *testing.T) {
	inputJSON := `
{
    "format": "v0",
    "blockID": "00000000-0000-0000-0000-000000000000",
    "minID": "AAAAAAAAAAAAOO0z0LnnHg==",
    "maxID": "AAAAAAAAAAD/o61w2bYIDg==",
    "tenantID": "single-tenant",
    "startTime": "2021-01-01T00:00:00.0000000Z",
    "endTime": "2021-01-02T00:00:00.0000000Z",
    "totalObjects": 10,
    "size": 12345,
    "compactionLevel": 0,
    "encoding": "zstd",
    "indexPageSize": 250000,
    "totalRecords": 124356,
    "dataEncoding": "",
    "bloomShards": 244
}
`

	blockMeta := BlockMeta{}
	err := json.Unmarshal([]byte(inputJSON), &blockMeta)
	assert.NoError(t, err, "expected to be able to unmarshal from JSON")
}
