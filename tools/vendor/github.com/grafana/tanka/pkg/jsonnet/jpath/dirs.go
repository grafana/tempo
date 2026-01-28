package jpath

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dirs returns the project-root (root) and environment directory (base)
func Dirs(path string) (root string, base string, err error) {
	root, err = FindRoot(path)
	if err != nil {
		return "", "", err
	}

	base, err = FindBase(path, root)
	if err != nil {
		return root, "", err
	}

	return root, base, err
}

// FindRoot returns the absolute path of the project root, being the directory
// that directly holds `tkrc.yaml` if it exists, otherwise the directory that
// directly holds `jsonnetfile.json`
func FindRoot(path string) (dir string, err error) {
	start, err := FsDir(path)
	if err != nil {
		return "", err
	}

	// root path based on os
	stop := "/"
	if runtime.GOOS == "windows" {
		stop = filepath.VolumeName(start) + "\\"
	}

	// try tkrc.yaml first
	root, err := FindParentFile("tkrc.yaml", start, stop)
	if err == nil {
		return root, nil
	}

	// otherwise use jsonnetfile.json
	root, err = FindParentFile("jsonnetfile.json", start, stop)
	if _, ok := err.(ErrorFileNotFound); ok {
		return "", ErrorNoRoot
	} else if err != nil {
		return "", err
	}

	return root, nil
}

// FindBase returns the absolute path of the environments base directory, the
// one which directly holds the entrypoint file.
func FindBase(path string, root string) (string, error) {
	dir, err := FsDir(path)
	if err != nil {
		return "", err
	}

	filename, err := Filename(path)
	if err != nil {
		return "", err
	}

	base, err := FindParentFile(filename, dir, root)

	if _, ok := err.(ErrorFileNotFound); ok {
		return "", ErrorNoBase{filename: filename}
	} else if err != nil {
		return "", err
	}

	return base, nil
}

// FindParentFile traverses the parent directory tree for the given `file`,
// starting from `start` and ending in `stop`. If the file is not found an error is returned.
func FindParentFile(file, start, stop string) (string, error) {
	files, err := os.ReadDir(start)
	if err != nil {
		return "", err
	}

	if dirContainsFile(files, file) {
		return start, nil
	} else if start == stop {
		return "", ErrorFileNotFound{file}
	}
	return FindParentFile(file, filepath.Dir(start), stop)
}

// dirContainsFile returns whether a file is included in a directory.
func dirContainsFile(files []os.DirEntry, filename string) bool {
	for _, f := range files {
		if f.Name() == filename {
			return true
		}
	}
	return false
}

// FsDir returns the most inner directory of path, as reported by the local
// filesystem
func FsDir(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		return path, nil
	}

	return filepath.Dir(path), nil
}
