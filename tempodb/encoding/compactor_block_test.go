package encoding

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/index"

	"github.com/stretchr/testify/assert"
)

const (
	testTenantID = "fake"
)

func TestCompactorBlockError(t *testing.T) {
	_, err := NewCompactorBlock(uuid.New(), "", 0, 0, nil, 0)
	assert.Error(t, err)
}

func TestCompactorBlockAddObject(t *testing.T) {
	indexDownsample := 3
	bloomFP := .01

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
	cb, err := NewCompactorBlock(uuid.New(), testTenantID, bloomFP, indexDownsample, metas, numObjects)
	assert.NoError(t, err)

	var minID index.ID
	var maxID index.ID

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

		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
	}
	cb.appender.Complete()

	assert.Equal(t, numObjects, cb.Length())

	// test meta
	meta := cb.BlockMeta()

	assert.Equal(t, time.Unix(10000, 0), meta.StartTime)
	assert.Equal(t, time.Unix(25000, 0), meta.EndTime)
	assert.Equal(t, minID, index.ID(meta.MinID))
	assert.Equal(t, maxID, index.ID(meta.MaxID))
	assert.Equal(t, testTenantID, meta.TenantID)
	assert.Equal(t, numObjects, meta.TotalObjects)

	// bloom
	for _, id := range ids {
		has := cb.bloom.Test(id)
		assert.True(t, has)
	}

	records := cb.appender.Records()
	assert.Equal(t, math.Ceil(float64(numObjects)/float64(indexDownsample)), float64(len(records)))

	assert.Equal(t, numObjects, cb.CurrentBufferedObjects())
}
