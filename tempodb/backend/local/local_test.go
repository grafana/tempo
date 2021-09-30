package local

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/grafana/tempo/pkg/io"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/tempodb/backend"
)

const objectName = "test"

func TestReadWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	fakeTracesFile, err := os.CreateTemp("/tmp", "")
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
		err = w.Write(ctx, objectName, backend.KeyPathForBlock(fakeMeta.BlockID, id), bytes.NewReader(fakeObject), int64(len(fakeObject)), false)
		assert.NoError(t, err, "unexpected error writing")
	}

	actualObject, size, err := r.Read(ctx, objectName, backend.KeyPathForBlock(blockID, tenantIDs[0]), false)
	assert.NoError(t, err, "unexpected error reading")
	actualObjectBytes, err := io.ReadAllWithEstimate(actualObject, size)
	assert.NoError(t, err, "unexpected error reading")
	assert.Equal(t, fakeObject, actualObjectBytes)

	actualReadRange := make([]byte, 5)
	err = r.ReadRange(ctx, objectName, backend.KeyPathForBlock(blockID, tenantIDs[0]), 5, actualReadRange)
	assert.NoError(t, err, "unexpected error range")
	assert.Equal(t, fakeObject[5:10], actualReadRange)

	list, err := r.List(ctx, backend.KeyPath{tenantIDs[0]})
	assert.NoError(t, err, "unexpected error reading blocklist")
	assert.Len(t, list, 1)
	assert.Equal(t, blockID.String(), list[0])
}
