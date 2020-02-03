package local

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
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

	r, w, err := New(Config{
		Path: tempDir,
	})
	assert.NoError(t, err, "unexpected error creating local backend")

	blockID := uuid.New()
	tenantID := "fake"

	fakeBloom := make([]byte, 20)
	fakeIndex := make([]byte, 20)
	fakeTraces := make([]byte, 200)

	_, err = rand.Read(fakeBloom)
	assert.NoError(t, err, "unexpected error creating fakeBloom")
	_, err = rand.Read(fakeIndex)
	assert.NoError(t, err, "unexpected error creating fakeIndex")
	_, err = rand.Read(fakeTraces)
	assert.NoError(t, err, "unexpected error creating fakeTraces")
	_, err = fakeTracesFile.Write(fakeTraces)
	assert.NoError(t, err, "unexpected error writing fakeTraces")

	err = w.Write(context.Background(), blockID, tenantID, fakeBloom, fakeIndex, fakeTracesFile.Name())
	assert.NoError(t, err, "unexpected error writing")

	actualIndex, err := r.Index(blockID, tenantID)
	assert.NoError(t, err, "unexpected error reading indexes")
	assert.Equal(t, fakeIndex, actualIndex)

	actualTrace, err := r.Object(blockID, tenantID, 100, 20)
	assert.NoError(t, err, "unexpected error reading traces")
	assert.Equal(t, fakeTraces[100:120], actualTrace)

	i := 0
	err = r.Bloom(tenantID, func(actualBloomBytes []byte, actualBlockID uuid.UUID) (bool, error) {
		assert.Equal(t, blockID, actualBlockID)
		assert.Equal(t, fakeBloom, actualBloomBytes)
		i++

		return true, nil
	})
	assert.NoError(t, err, "unexpected error iterating bloom")
	assert.Equal(t, 1, i, "should only be 1 bloom filter")
}
