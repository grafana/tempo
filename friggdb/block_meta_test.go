package friggdb

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBlockMeta(t *testing.T) {
	id := uuid.New()
	b := newBlockMeta(testTenantID, id)

	assert.Equal(t, id, b.BlockID)
	assert.Equal(t, testTenantID, b.TenantID)

	randID1 := make([]byte, 10)
	randID2 := make([]byte, 10)

	rand.Read(randID1)
	rand.Read(randID2)

	assert.Equal(t, b.StartTime, b.EndTime)

	b.objectAdded(randID1)
	b.objectAdded(randID2)
	assert.True(t, b.EndTime.After(b.StartTime))
	assert.Equal(t, 1, bytes.Compare(b.MaxID, b.MinID))
}
