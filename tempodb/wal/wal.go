package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	workDir = "work"
)

type WAL struct {
	c *Config
}

type Config struct {
	Filepath        string `yaml:"path"`
	WorkFilepath    string
	IndexDownsample int     `yaml:"index-downsample"`
	BloomFP         float64 `yaml:"bloom-filter-false-positive"`
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

	if c.WorkFilepath == "" {
		workFilepath := path.Join(c.Filepath, workDir)
		err = os.RemoveAll(workFilepath)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(workFilepath, os.ModePerm)
		if err != nil {
			return nil, err
		}

		c.WorkFilepath = workFilepath
	}

	return &WAL{
		c: c,
	}, nil
}

func (w *WAL) AllBlocks() ([]ReplayBlock, error) {
	files, err := ioutil.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	blocks := make([]ReplayBlock, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		blockID, tenantID, err := parseFilename(name)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &CompleteBlock{
			block: block{
				meta:     backend.NewBlockMeta(tenantID, blockID),
				filepath: w.c.Filepath,
			},
		})
	}

	return blocks, nil
}

func (w *WAL) NewBlock(id uuid.UUID, tenantID string) (*HeadBlock, error) {
	return newHeadBlock(id, tenantID, w.c.Filepath)
}

func (w *WAL) NewCompactorBlock(id uuid.UUID, tenantID string, metas []*backend.BlockMeta, estimatedObjects int) (*CompactorBlock, error) {
	return newCompactorBlock(id, tenantID, w.c.BloomFP, w.c.IndexDownsample, metas, w.c.WorkFilepath, estimatedObjects)
}

func (w *WAL) config() *Config {
	return w.c
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
