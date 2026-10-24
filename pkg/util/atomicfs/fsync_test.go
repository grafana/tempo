// SPDX-License-Identifier: AGPL-3.0-only

package atomicfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	requireOnlyFile(t, filepath.Dir(path), "file")
}

func TestCreateFileFailedCopyPublishesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	readErr := errors.New("reader failed")

	err := CreateFile(path, io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr)))

	require.ErrorIs(t, err, readErr)
	requireOnlyFile(t, filepath.Dir(path))
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

func TestCreateFileConcurrentWritersToSamePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	const writers = 16
	payloads := make([]string, writers)
	for i := range payloads {
		payloads[i] = strings.Repeat(string(rune('a'+i)), 64<<10)
	}

	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = CreateFile(path, strings.NewReader(payloads[i]))
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, payloads, string(got), "the final file mixes the content of several writers")
	requireOnlyFile(t, dir, "file")
}

// requireOnlyFile asserts that no temporary file was left beside the final one.
func requireOnlyFile(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	got := make([]string, 0, len(entries))
	for _, e := range entries {
		got = append(got, e.Name())
	}
	require.ElementsMatch(t, names, got)
}
