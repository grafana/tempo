package local

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/tempodb/backend"
)

const objectName = "test"
const objectReaderName = "test-reader"

func TestReadWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	fakeTracesFile, err := ioutil.TempFile("/tmp", "")
	defer os.Remove(fakeTracesFile.Name())
	assert.NoError(t, err, "unexpected error creating temp file")

	r, w, _, err := New(&Config{
		Path: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating local backend")

	blockID := uuid.New()
	tenantIDs := []string{"fake"}

	for i := 0; i < 10; i++ {
		tenantIDs = append(tenantIDs, fmt.Sprintf("%d", rand.Int()))
	}

	fakeMeta := &backend.BlockMeta{
		BlockID: blockID,
	}

	fakeObject := make([]byte, 20)

	_, err = rand.Read(fakeObject)
	assert.NoError(t, err, "unexpected error creating fakeObject")

	ctx := context.Background()
	for _, id := range tenantIDs {
		fakeMeta.TenantID = id
		err = w.Write(ctx, objectName, backend.KeyPathForBlock(fakeMeta.BlockID, id), fakeObject)
		assert.NoError(t, err, "unexpected error writing")
		err = w.WriteReader(ctx, objectReaderName, backend.KeyPathForBlock(fakeMeta.BlockID, id), bytes.NewBuffer(fakeObject), int64(len(fakeObject)))
		assert.NoError(t, err, "unexpected error writing reader")
	}

	actualObject, err := r.Read(ctx, objectName, backend.KeyPathForBlock(blockID, tenantIDs[0]))
	assert.NoError(t, err, "unexpected error reading")
	assert.Equal(t, fakeObject, actualObject)

	actualReadRange := make([]byte, 5)
	err = r.ReadRange(ctx, objectReaderName, backend.KeyPathForBlock(blockID, tenantIDs[0]), 5, actualReadRange)
	assert.NoError(t, err, "unexpected error range")
	assert.Equal(t, fakeObject[5:10], actualReadRange)

	list, err := r.List(ctx, backend.KeyPath{tenantIDs[0]})
	assert.NoError(t, err, "unexpected error reading blocklist")
	assert.Len(t, list, 1)
	assert.Equal(t, blockID.String(), list[0])
}
