package godirwalk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirent(t *testing.T) {
	// TODO: IsDevice() should be tested, but that would require updating
	// scaffolding to create a device.

	t.Run("file", func(t *testing.T) {
		de, err := NewDirent(filepath.Join(scaffolingRoot, "d0", "f1"))
		ensureError(t, err)

		if got, want := de.Name(), "f1"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.ModeType(), os.FileMode(0); got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsDir(), false; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		got, err := de.IsDirOrSymlinkToDir()
		ensureError(t, err)

		if want := false; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsRegular(), true; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsSymlink(), false; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})

	t.Run("directory", func(t *testing.T) {
		de, err := NewDirent(filepath.Join(scaffolingRoot, "d0"))
		ensureError(t, err)

		if got, want := de.Name(), "d0"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.ModeType(), os.ModeDir; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsDir(), true; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		got, err := de.IsDirOrSymlinkToDir()
		ensureError(t, err)

		if want := true; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsRegular(), false; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		if got, want := de.IsSymlink(), false; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		t.Run("to file", func(t *testing.T) {
			de, err := NewDirent(filepath.Join(scaffolingRoot, "d0", "symlinks", "toF1"))
			ensureError(t, err)

			if got, want := de.Name(), "toF1"; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.ModeType(), os.ModeSymlink; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsDir(), false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			got, err := de.IsDirOrSymlinkToDir()
			ensureError(t, err)

			if want := false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsRegular(), false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsSymlink(), true; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})

		t.Run("to directory", func(t *testing.T) {
			de, err := NewDirent(filepath.Join(scaffolingRoot, "d0", "symlinks", "toD1"))
			ensureError(t, err)

			if got, want := de.Name(), "toD1"; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.ModeType(), os.ModeSymlink; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsDir(), false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			got, err := de.IsDirOrSymlinkToDir()
			ensureError(t, err)

			if want := true; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsRegular(), false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			if got, want := de.IsSymlink(), true; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
	})
}
