package goimpl

import (
	"path/filepath"

	jsonnet "github.com/google/go-jsonnet"
)

const locationInternal = "<internal>"

// extendedImporter wraps jsonnet.FileImporter to add additional functionality:
// - `import "file.yaml"`
// - `import "tk"`
type extendedImporter struct {
	loaders    []importLoader    // for loading jsonnet from somewhere. First one that returns non-nil is used
	processors []importProcessor // for post-processing (e.g. yaml -> json)
}

// importLoader are executed before the actual importing. If they return
// something, this value is used.
type importLoader func(importedFrom, importedPath string) (c *jsonnet.Contents, foundAt string, err error)

// importProcessor are executed after the file import and may modify the result
// further
type importProcessor func(contents, foundAt string) (c *jsonnet.Contents, err error)

// newExtendedImporter returns a new instance of ExtendedImporter with the
// correct jpaths set up
func newExtendedImporter(jpath []string) *extendedImporter {
	return &extendedImporter{
		loaders: []importLoader{
			tkLoader,
			newFileLoader(&jsonnet.FileImporter{
				JPaths: jpath,
			})},
		processors: []importProcessor{},
	}
}

// Import implements the functionality offered by the ExtendedImporter
func (i *extendedImporter) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	// load using loader
	for _, loader := range i.loaders {
		c, f, err := loader(importedFrom, importedPath)
		if err != nil {
			return jsonnet.Contents{}, "", err
		}
		if c != nil {
			contents = *c
			foundAt = f
			break
		}
	}

	// check if needs postprocessing
	for _, processor := range i.processors {
		c, err := processor(contents.String(), foundAt)
		if err != nil {
			return jsonnet.Contents{}, "", err
		}
		if c != nil {
			contents = *c
			break
		}
	}

	return contents, foundAt, nil
}

// tkLoader provides `tk.libsonnet` from memory (builtin)
func tkLoader(_, importedPath string) (contents *jsonnet.Contents, foundAt string, err error) {
	if importedPath != "tk" {
		return nil, "", nil
	}

	return &tkLibsonnet, filepath.Join(locationInternal, "tk.libsonnet"), nil
}

// newFileLoader returns an importLoader that uses jsonnet.FileImporter to source
// files from the local filesystem
func newFileLoader(fi *jsonnet.FileImporter) importLoader {
	return func(importedFrom, importedPath string) (contents *jsonnet.Contents, foundAt string, err error) {
		var c jsonnet.Contents
		c, foundAt, err = fi.Import(importedFrom, importedPath)
		return &c, foundAt, err
	}
}
