package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/encoding"
)

const (
	workDir = "work"
)

type WAL interface {
	AllBlocks() ([]ReplayBlock, error)
	NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error)
	config() *Config
}

type wal struct {
	c *Config
}

type Config struct {
	Filepath        string `yaml:"path"`
	WorkFilepath    string
	IndexDownsample int     `yaml:"index-downsample"`
	BloomFP         float64 `yaml:"bloom-filter-false-positive"`
}

func New(c *Config) (WAL, error) {
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

	return &wal{
		c: c,
	}, nil
}

func (w *wal) AllBlocks() ([]ReplayBlock, error) {
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

		blocks = append(blocks, &headBlock{
			completeBlock: completeBlock{
				meta:     encoding.NewBlockMeta(tenantID, blockID),
				filepath: w.c.Filepath,
			},
		})
	}

	return blocks, nil
}

func (w *wal) NewBlock(id uuid.UUID, tenantID string) (HeadBlock, error) {
	return newBlock(id, tenantID, w.c.Filepath)
}

func (w *wal) config() *Config {
	return w.c
}

func newBlock(id uuid.UUID, tenantID string, filepath string) (*headBlock, error) {
	h := &headBlock{
		completeBlock: completeBlock{
			meta:     encoding.NewBlockMeta(tenantID, id),
			filepath: filepath,
		},
	}

	name := h.fullFilename()
	_, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	return h, nil
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
