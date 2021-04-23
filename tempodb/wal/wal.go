package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
)

const (
	completedDir = "completed"
	blocksDir    = "blocks"
)

type WAL struct {
	c *Config
	l *local.Backend
}

type Config struct {
	Filepath          string `yaml:"path"`
	CompletedFilepath string
	BlocksFilepath    string
	Encoding          backend.Encoding `yaml:"encoding"`
}

func New(c *Config) (*WAL, error) {
	if c.Filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	// make folder
	err := os.MkdirAll(c.Filepath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// The /completed/ folder is now obsolete and no new data is written,
	// but it needs to be cleared out one last time for any files left
	// from a previous version.
	if c.CompletedFilepath == "" {
		completedFilepath := filepath.Join(c.Filepath, completedDir)
		err = os.RemoveAll(completedFilepath)
		if err != nil {
			return nil, err
		}

		c.CompletedFilepath = completedFilepath
	}

	// Setup local backend in /blocks/
	p := filepath.Join(c.Filepath, blocksDir)
	err = os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return nil, err
	}
	c.BlocksFilepath = p

	l, err := local.NewBackend(&local.Config{
		Path: p,
	})
	if err != nil {
		return nil, err
	}

	return &WAL{
		c: c,
		l: l,
	}, nil
}

// AllBlocks returns a slice of append blocks from the wal folder, a list of warnings encountered while
// rebuilding and an error if there was a fatal issue. It is meant for wal replay.
func (w *WAL) AllBlocks() ([]*AppendBlock, []error, error) {
	var warnings []error
	files, err := ioutil.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, nil, err
	}

	blocks := make([]*AppendBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		r, err := newAppendBlockFromFile(f.Name(), w.c.Filepath)
		if err != nil {
			// wal replay failed, clear and warn
			warnings = append(warnings, fmt.Errorf("failed to replay block. removing %s: %w", f.Name(), err))
			err = os.Remove(filepath.Join(w.c.Filepath, f.Name()))
			if err != nil {
				return nil, nil, err
			}

			continue
		}

		blocks = append(blocks, r)
	}

	return blocks, warnings, nil
}

func (w *WAL) NewBlock(id uuid.UUID, tenantID string) (*AppendBlock, error) {
	return newAppendBlock(id, tenantID, w.c.Filepath, w.c.Encoding)
}

func (w *WAL) LocalBackend() *local.Backend {
	return w.l
}
