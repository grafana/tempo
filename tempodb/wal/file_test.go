package wal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("/tmp", "")
	defer os.RemoveAll(tempDir)
	require.NoError(t, err, "unexpected error creating temp dir")

	testTenantID := "test"
	blockID := uuid.New()

	expectedFile := &File{
		Filepath: tempDir,
		TenantID: testTenantID,
		BlockID:  blockID,
	}
	filename := fullFilename(expectedFile)

	osFile, err := os.Create(filename)
	require.NoError(t, err)
	require.FileExists(t, filename)

	name := filepath.Base(osFile.Name())
	actualFile, err := newFile(name, tempDir)
	require.NoError(t, err)
	assert.Equal(t, expectedFile, actualFile)

	err = actualFile.Clear()
	assert.NoError(t, err)
	assert.NoFileExists(t, filename)
}
