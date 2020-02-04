package local

import (
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

	err = w.Write(blockID, tenantID, fakeMeta, fakeBloom, fakeIndex, fakeTracesFile.Name())
	assert.NoError(t, err, "unexpected error writing")

	actualIndex, err := r.Index(blockID, tenantID)
	assert.NoError(t, err, "unexpected error reading indexes")
	assert.Equal(t, fakeIndex, actualIndex)

	actualTrace, err := r.Object(blockID, tenantID, 100, 20)
	assert.NoError(t, err, "unexpected error reading traces")
	assert.Equal(t, fakeTraces[100:120], actualTrace)

	actualBloom, err := r.Bloom(blockID, tenantID)
	assert.NoError(t, err, "unexpected error reading bloom")
	assert.Equal(t, fakeBloom, actualBloom)

	list, err := r.Blocklist(tenantID)
	assert.NoError(t, err, "unexpected error reading blocklist")
	assert.Len(t, list, 1)
	assert.Equal(t, fakeMeta, list[0])
}
