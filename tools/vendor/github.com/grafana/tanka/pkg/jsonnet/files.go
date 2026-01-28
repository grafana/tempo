package jsonnet

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
)

// FindFiles takes a file / directory and finds all Jsonnet files
func FindFiles(target string, excludes []glob.Glob) ([]string, error) {
	// if it's a file, don't try to find children
	fi, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if fi.Mode().IsRegular() {
		return []string{target}, nil
	}

	var files []string

	err = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = filepath.ToSlash(path)
		if d.IsDir() {
			return nil
		}

		// excluded?
		for _, g := range excludes {
			if g.Match(path) {
				return nil
			}
		}

		// only .jsonnet or .libsonnet
		if ext := filepath.Ext(path); ext == ".jsonnet" || ext == ".libsonnet" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
