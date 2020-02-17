package godirwalk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDirents(t *testing.T) {
	t.Run("without symlinks", func(t *testing.T) {
		testroot := filepath.Join(scaffolingRoot, "d0")

		actual, err := ReadDirents(testroot, nil)
		ensureError(t, err)

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

	t.Run("with symlinks", func(t *testing.T) {
		testroot := filepath.Join(scaffolingRoot, "d0/symlinks")

		actual, err := ReadDirents(testroot, nil)
		ensureError(t, err)

		// Because some platforms set multiple mode type bits, when we create
		// the expected slice, we need to ensure the mode types are set
		// appropriately for this platform. We have another test function to
		// ensure NewDirent does this correctly, so let's call NewDirent for
		// each of the expected children entries.
		var expected Dirents
		for _, child := range []string{"nothing", "toAbs", "toD1", "toF1", "d4"} {
			de, err := NewDirent(filepath.Join(testroot, child))
			if err != nil {
				t.Fatal(err)
			}
			expected = append(expected, de)
		}

		ensureDirentsMatch(t, actual, expected)
	})
}

func TestReadDirnames(t *testing.T) {
	actual, err := ReadDirnames(filepath.Join(scaffolingRoot, "d0"), nil)
	ensureError(t, err)
	expected := []string{maxName, "d1", "f1", "skips", "symlinks"}
	ensureStringSlicesMatch(t, actual, expected)
}

func BenchmarkReadDirnamesStandardLibrary(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	f := func(osDirname string) ([]string, error) {
		dh, err := os.Open(osDirname)
		if err != nil {
			return nil, err
		}
		return dh.Readdirnames(-1)
	}

	var count int

	for i := 0; i < b.N; i++ {
		actual, err := f(goPrefix)
		if err != nil {
			b.Fatal(err)
		}
		count = len(actual)
	}
	_ = count
}

func BenchmarkReadDirnamesGodirwalk(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	var count int

	for i := 0; i < b.N; i++ {
		actual, err := ReadDirnames(goPrefix, nil)
		if err != nil {
			b.Fatal(err)
		}
		count = len(actual)
	}
	_ = count
}
