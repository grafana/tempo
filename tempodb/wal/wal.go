package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding"
)

const (
	completedDir = "completed"
)

type WAL struct {
	c *Config
}

type Config struct {
	Filepath          string `yaml:"path"`
	CompletedFilepath string
	IndexDownsample   int     `yaml:"index_downsample"`
	BloomFP           float64 `yaml:"bloom_filter_false_positive"`
}

func New(c *Config) (*WAL, error) {
	if c.Filepath == "" {
		return nil, fmt.Errorf("please provide a path for the WAL")
	}

	if c.IndexDownsample == 0 {
		return nil, fmt.Errorf("Non-zero index downsample required")
	}

	if c.BloomFP <= 0.0 {
		return nil, fmt.Errorf("invalid bloom filter fp rate %v", c.BloomFP)
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

		name := f.Name()
		blockID, tenantID, err := parseFilename(name)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &ReplayBlock{
			block: block{
				meta:     encoding.NewBlockMeta(tenantID, blockID),
				filepath: w.c.Filepath,
			},
		})
	}

	return blocks, nil
}

func (w *WAL) NewBlock(id uuid.UUID, tenantID string) (*AppendBlock, error) {
	return newAppendBlock(id, tenantID, w.c.Filepath)
}

func parseFilename(name string) (uuid.UUID, string, error) {
	i := strings.Index(name, ":")

	if i < 0 {
		return uuid.UUID{}, "", fmt.Errorf("unable to parse %s", name)
	}

	blockIDString := name[:i]
	tenantID := name[i+1:]

	blockID, err := uuid.Parse(blockIDString)
	if err != nil {
		return uuid.UUID{}, "", err
	}

	return blockID, tenantID, nil
}
