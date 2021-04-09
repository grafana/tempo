package encoding

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
)

func TestCompactorBlockError(t *testing.T) {
	_, err := NewStreamingBlock(nil, uuid.New(), "", nil, 0)
	assert.Error(t, err)
}

func TestCompactorBlockAddObject(t *testing.T) {
	indexDownsample := 500

	metas := []*backend.BlockMeta{
		{
			StartTime: time.Unix(10000, 0),
			EndTime:   time.Unix(20000, 0),
		},
		{
			StartTime: time.Unix(15000, 0),
			EndTime:   time.Unix(25000, 0),
		},
	}

	numObjects := (rand.Int() % 20) + 1
	cb, err := NewStreamingBlock(&BlockConfig{
		BloomFP:              .01,
		IndexDownsampleBytes: indexDownsample,
		Encoding:             backend.EncGZIP,
	}, uuid.New(), testTenantID, metas, numObjects)
	assert.NoError(t, err)

	var minID common.ID
	var maxID common.ID

	expectedRecords := 0
	byteCounter := 0

	ids := make([][]byte, 0)
	for i := 0; i < numObjects; i++ {
		id := make([]byte, 16)
		_, err = rand.Read(id)
		assert.NoError(t, err)

		object := make([]byte, rand.Int()%1024)
		_, err = rand.Read(object)
		assert.NoError(t, err)

		ids = append(ids, id)

		err = cb.AddObject(id, object)
		assert.NoError(t, err)

		byteCounter += len(id) + len(object) + 4 + 4
		if byteCounter > indexDownsample {
			byteCounter = 0
			expectedRecords++
		}

		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
	}
	if byteCounter > 0 {
		expectedRecords++
	}

	err = cb.appender.Complete()
	assert.NoError(t, err)
	assert.Equal(t, numObjects, cb.Length())

	// test meta
	meta := cb.BlockMeta()

	assert.Equal(t, time.Unix(10000, 0), meta.StartTime)
	assert.Equal(t, time.Unix(25000, 0), meta.EndTime)
	assert.Equal(t, minID, common.ID(meta.MinID))
	assert.Equal(t, maxID, common.ID(meta.MaxID))
	assert.Equal(t, testTenantID, meta.TenantID)
	assert.Equal(t, numObjects, meta.TotalObjects)
	assert.Greater(t, meta.Size, uint64(0))

	// bloom
	for _, id := range ids {
		has := cb.bloom.Test(id)
		assert.True(t, has)
	}

	records := cb.appender.Records()
	assert.Equal(t, expectedRecords, len(records))
	assert.Equal(t, numObjects, cb.CurrentBufferedObjects())
}
