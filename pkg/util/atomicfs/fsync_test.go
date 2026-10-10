// SPDX-License-Identifier: AGPL-3.0-only

package atomicfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestCreateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")

	require.NoError(t, CreateFile(path, strings.NewReader("content")))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "content", string(got))
	require.NoFileExists(t, tempPath(path))
}

func TestCreateFileFailedCopyPublishesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	readErr := errors.New("reader failed")

	err := CreateFile(path, io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr)))

	require.ErrorIs(t, err, readErr)
	require.NoFileExists(t, path)
	require.NoFileExists(t, tempPath(path))
}

func TestCreateFileFailedCopyKeepsPreviousFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, CreateFile(path, strings.NewReader("previous")))

	err := CreateFile(path, iotest.ErrReader(errors.New("reader failed")))

	require.Error(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "previous", string(got))
}
