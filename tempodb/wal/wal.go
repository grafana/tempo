package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	completedDir = "completed"
)

type WAL struct {
	c *Config
}

type Config struct {
	Filepath          string           `yaml:"path"`
	CompletedFilepath string           `yaml:"completed_file_path"`
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

	if c.CompletedFilepath == "" {
		completedFilepath := filepath.Join(c.Filepath, completedDir)
		err = os.RemoveAll(completedFilepath)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(completedFilepath, os.ModePerm)
		if err != nil {
			return nil, err
		}

		c.CompletedFilepath = completedFilepath
	}

	return &WAL{
		c: c,
	}, nil
}

func (w *WAL) AllBlocks() ([]*ReplayBlock, error) {
	files, err := ioutil.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	blocks := make([]*ReplayBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		r, err := NewReplayBlock(f.Name(), w.c.Filepath)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, r)
	}

	return blocks, nil
}

func (w *WAL) NewBlock(id uuid.UUID, tenantID string) (*AppendBlock, error) {
	return newAppendBlock(id, tenantID, w.c.Filepath, w.c.Encoding)
}
