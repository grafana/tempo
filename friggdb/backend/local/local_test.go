package local

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReadWrite(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	fakeTracesFile, err := ioutil.TempFile("/tmp", "")
	defer os.Remove(fakeTracesFile.Name())
	assert.NoError(t, err, "unexpected error creating temp file")

	r, w, err := New(&Config{
		Path: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating local backend")

	blockID := uuid.New()
	tenantIDs := []string{"fake"}

	for i := 0; i < 10; i++ {
		tenantIDs = append(tenantIDs, fmt.Sprintf("%d", rand.Int()))
	}

	fakeMeta := make([]byte, 20)
	fakeBloom := make([]byte, 20)
	fakeIndex := make([]byte, 20)
	fakeTraces := make([]byte, 200)

	_, err = rand.Read(fakeMeta)
	assert.NoError(t, err, "unexpected error creating fakeMeta")
	_, err = rand.Read(fakeBloom)
	assert.NoError(t, err, "unexpected error creating fakeBloom")
	_, err = rand.Read(fakeIndex)
	assert.NoError(t, err, "unexpected error creating fakeIndex")
	_, err = rand.Read(fakeTraces)
	assert.NoError(t, err, "unexpected error creating fakeTraces")
	_, err = fakeTracesFile.Write(fakeTraces)
	assert.NoError(t, err, "unexpected error writing fakeTraces")

	for _, id := range tenantIDs {
		err = w.Write(context.Background(), blockID, id, fakeMeta, fakeBloom, fakeIndex, fakeTracesFile.Name())
		assert.NoError(t, err, "unexpected error writing")
	}

	actualMeta, err := r.BlockMeta(blockID, tenantIDs[0])
	assert.NoError(t, err, "unexpected error reading indexes")
	assert.Equal(t, fakeMeta, actualMeta)

	actualIndex, err := r.Index(blockID, tenantIDs[0])
	assert.NoError(t, err, "unexpected error reading indexes")
	assert.Equal(t, fakeIndex, actualIndex)

	actualTrace, err := r.Object(blockID, tenantIDs[0], 100, 20)
	assert.NoError(t, err, "unexpected error reading traces")
	assert.Equal(t, fakeTraces[100:120], actualTrace)

	actualBloom, err := r.Bloom(blockID, tenantIDs[0])
	assert.NoError(t, err, "unexpected error reading bloom")
	assert.Equal(t, fakeBloom, actualBloom)

	list, err := r.Blocks(tenantIDs[0])
	assert.NoError(t, err, "unexpected error reading blocklist")
	assert.Len(t, list, 1)
	assert.Equal(t, blockID, list[0])

	tenants, err := r.Tenants()
	assert.NoError(t, err, "unexpected error reading tenants")
	assert.Len(t, tenants, len(tenantIDs))
}

func TestWriteFail(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	_, w, err := New(&Config{
		Path: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating local backend")

	blockID := uuid.New()
	tenantID := "fake"

	err = w.Write(context.Background(), blockID, tenantID, nil, nil, nil, "file-that-doesnt-exist")
	assert.Error(t, err)

	_, err = os.Stat(path.Join(tempDir, tenantID, blockID.String()))
	assert.True(t, os.IsNotExist(err))
}
