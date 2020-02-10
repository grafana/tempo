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

func TestJanitor(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err, "unexpected error creating temp dir")

	// 1KB per file
	missBytes := make([]byte, 1024, 1024)
	missFunc := func(blockID uuid.UUID, tenantID string) ([]byte, error) {
		return missBytes, nil
	}

	cache, err := New(nil, &Config{
		Path:           tempDir,
		MaxDiskMBs:     30,
		DiskPruneCount: 10,
		DiskCleanRate:  time.Hour,
		MaxMemoryMBs:   1024,
	})
	assert.NoError(t, err)

	// test
	for i := 0; i < 10; i++ {
		blockID := uuid.New()
		tenantID := "fake"

		bytes, err := cache.(*reader).readOrCacheKeyToDisk(blockID, tenantID, "type", missFunc)
		assert.NoError(t, err)
		assert.Equal(t, missBytes, bytes)
	}

	// force prune. should do nothing b/c we don't have enough files
	cleaned := clean(tempDir, 1, 10)
	assert.False(t, cleaned)

	// now make enough files to prune
	fi, err := ioutil.ReadDir(tempDir)
	assert.Len(t, fi, 10)
	assert.NoError(t, err)

	// create 1024 files
	for i := 0; i < 1024; i++ {
		blockID := uuid.New()
		tenantID := "fake"

		bytes, err := cache.(*reader).readOrCacheKeyToDisk(blockID, tenantID, "type", missFunc)
		assert.NoError(t, err)
		assert.Equal(t, missBytes, bytes)
	}

	fi, err = ioutil.ReadDir(tempDir)
	assert.Len(t, fi, 1034)
	assert.NoError(t, err)

	// force clean at 1MB and see only 1033 (b/c prunecount = 1)
	cleaned = clean(tempDir, 1, 1)
	assert.True(t, cleaned)

	fi, err = ioutil.ReadDir(tempDir)
	assert.Len(t, fi, 1033)
	assert.NoError(t, err)

	// force clean at 1MB and see only 1023 (b/c prunecount = 10)
	cleaned = clean(tempDir, 1, 10)
	assert.True(t, cleaned)

	fi, err = ioutil.ReadDir(tempDir)
	assert.Len(t, fi, 1023)
	assert.NoError(t, err)

	// force clean at 1MB and see only 1023 (b/c we're less than 1MB)
	cleaned = clean(tempDir, 1, 10)
	assert.False(t, cleaned)

	fi, err = ioutil.ReadDir(tempDir)
	assert.Len(t, fi, 1023)
	assert.NoError(t, err)
}
