package jpath

import (
	"os"
	"path/filepath"
)

const DefaultEntrypoint = "main.jsonnet"

// Resolve the given path and resolves the jPath around it. This means it:
// - figures out the project root (the one with .jsonnetfile, vendor/ and lib/)
// - figures out the environments base directory (usually the main.jsonnet)
//
// It then constructs a jPath with the base directory, vendor/ and lib/.
// This results in predictable imports, as it doesn't matter whether the user called
// called the command further down tree or not. A little bit like git.
func Resolve(path string, allowMissingBase bool) (jpath []string, base, root string, err error) {
	root, err = FindRoot(path)
	if err != nil {
		return nil, "", "", err
	}

	base, err = FindBase(path, root)
	if err != nil && allowMissingBase {
		base, err = FsDir(path)
		if err != nil {
			return nil, "", "", err
		}
	} else if err != nil {
		return nil, "", "", err
	}

	// The importer iterates through this list in reverse order
	return []string{
		filepath.Join(root, "vendor"),
		filepath.Join(base, "vendor"), // Look for a vendor folder in the base dir before using the root vendor
		filepath.Join(root, "lib"),
		base,
	}, base, root, nil
}

// Filename returns the name of the entrypoint file.
// It DOES NOT return an absolute path, only a plain name like "main.jsonnet"
// To obtain an absolute path, use Entrypoint() instead.
func Filename(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		return DefaultEntrypoint, nil
	}

	return filepath.Base(fi.Name()), nil
}

// Entrypoint returns the absolute path of the environments entrypoint file (the
// one passed to jsonnet.EvaluateFile)
func Entrypoint(path string) (string, error) {
	root, err := FindRoot(path)
	if err != nil {
		return "", err
	}

	base, err := FindBase(path, root)
	if err != nil {
		return "", err
	}

	filename, err := Filename(path)
	if err != nil {
		return "", err
	}

	return filepath.Join(base, filename), nil
}
