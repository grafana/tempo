package godirwalk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner(t *testing.T) {
	t.Run("collect names", func(t *testing.T) {
		var actual []string

		scanner, err := NewScanner(filepath.Join(scaffolingRoot, "d0"))
		ensureError(t, err)

		for scanner.Scan() {
			actual = append(actual, scanner.Name())
		}
		ensureError(t, scanner.Err())

		expected := []string{maxName, "d1", "f1", "skips", "symlinks"}
		ensureStringSlicesMatch(t, actual, expected)
	})

	t.Run("collect dirents", func(t *testing.T) {
		var actual []*Dirent

		testroot := filepath.Join(scaffolingRoot, "d0")

		scanner, err := NewScanner(testroot)
		ensureError(t, err)

		for scanner.Scan() {
			dirent, err := scanner.Dirent()
			ensureError(t, err)
			actual = append(actual, dirent)
		}
		ensureError(t, scanner.Err())

		expected := Dirents{
			&Dirent{
				name:     maxName,
				path:     testroot,
				modeType: os.FileMode(0),
			},
			&Dirent{
				name:     "d1",
				path:     testroot,
				modeType: os.ModeDir,
			},
			&Dirent{
				name:     "f1",
				path:     testroot,
				modeType: os.FileMode(0),
			},
			&Dirent{
				name:     "skips",
				path:     testroot,
				modeType: os.ModeDir,
			},
			&Dirent{
				name:     "symlinks",
				path:     testroot,
				modeType: os.ModeDir,
			},
		}

		ensureDirentsMatch(t, actual, expected)
	})

	t.Run("symlink to directory", func(t *testing.T) {
		scanner, err := NewScanner(filepath.Join(scaffolingRoot, "d0/symlinks"))
		ensureError(t, err)

		var found bool

		for scanner.Scan() {
			if scanner.Name() != "toD1" {
				continue
			}
			found = true

			de, err := scanner.Dirent()
			ensureError(t, err)

			got, err := de.IsDirOrSymlinkToDir()
			ensureError(t, err)

			if want := true; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		}

		ensureError(t, scanner.Err())

		if got, want := found, true; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})
}
