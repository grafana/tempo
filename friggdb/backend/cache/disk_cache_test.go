package cache

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReadOrCache(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	missBytes := []byte{0x01}
	missCalled := 0
	missFunc := func(blockID uuid.UUID, tenantID string) ([]byte, error) {
		missCalled++
		return missBytes, nil
	}

	cache, err := New(nil, &Config{
		Path:           tempDir,
		MaxDiskMBs:     1024,
		DiskPruneCount: 10,
		DiskCleanRate:  time.Hour,
		MaxMemoryMBs:   1024,
	})
	assert.NoError(t, err)

	blockID := uuid.New()
	tenantID := "fake"

	bytes, err := cache.(*reader).readOrCacheKeyToDisk(blockID, tenantID, "type", missFunc)
	assert.NoError(t, err)
	assert.Equal(t, missBytes, bytes)
	assert.Equal(t, missCalled, 1)

	bytes, err = cache.(*reader).readOrCacheKeyToDisk(blockID, tenantID, "type", missFunc)
	assert.NoError(t, err)
	assert.Equal(t, missBytes, bytes)
	assert.Equal(t, missCalled, 1)
}
