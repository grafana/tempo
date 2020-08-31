package main

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// LocalAssets contains the local objects to be uploaded
func LocalAssets(path string) ([]string, error) {
	if path == "" {
		return make([]string, 0, 0), nil
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get abs path")
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file stat")
	}

	if !fi.IsDir() {
		return []string{path}, nil
	}

	// Glob all files in the given path
	files, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to glob files")
	}

	assets := make([]string, 0, len(files))
	for _, f := range files {

		// Exclude directory.
		if fi, _ := os.Stat(f); fi.IsDir() {
			continue
		}

		// Exclude hidden file
		if filepath.Base(f)[0] == '.' {
			continue
		}

		assets = append(assets, f)
	}

	return assets, nil
}
