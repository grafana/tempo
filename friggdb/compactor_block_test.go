package friggdb

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCompactorBlockNoMeta(t *testing.T) {
	_, err := newCompactorBlock(testTenantID, &walConfig{}, nil)
	assert.Error(t, err)
}

func TestCompactorBlockWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	walCfg := &walConfig{
		workFilepath:    tempDir,
		indexDownsample: 2,
	}

	metas := []*blockMeta{
		&blockMeta{
			StartTime: time.Unix(10000, 0),
			EndTime:   time.Unix(20000, 0),
		},
		&blockMeta{
			StartTime: time.Unix(15000, 0),
			EndTime:   time.Unix(25000, 0),
		},
	}

	cb, err := newCompactorBlock(testTenantID, walCfg, metas)
	assert.NoError(t, err)

	var minID ID
	var maxID ID
	ids := make([][]byte, 0)
	objects := make([][]byte, 0)
	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		_, err = rand.Read(id)
		assert.NoError(t, err)

		object := make([]byte, rand.Int()%1024)
		_, err = rand.Read(object)
		assert.NoError(t, err)

		ids = append(ids, id)
		objects = append(objects, object)

		err = cb.write(id, object)
		assert.NoError(t, err)

		if len(minID) == 0 || bytes.Compare(id, minID) == -1 {
			minID = id
		}
		if len(maxID) == 0 || bytes.Compare(id, maxID) == 1 {
			maxID = id
		}
	}

	// test meta
	metaBytes, err := cb.meta()
	assert.NoError(t, err)

	meta := &blockMeta{}
	err = json.Unmarshal(metaBytes, meta)
	assert.NoError(t, err)

	assert.Equal(t, time.Unix(10000, 0), meta.StartTime)
	assert.Equal(t, time.Unix(25000, 0), meta.EndTime)
	assert.Equal(t, minID, meta.MinID)
	assert.Equal(t, maxID, meta.MaxID)
	assert.Equal(t, testTenantID, meta.TenantID)
}
