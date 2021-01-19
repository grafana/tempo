package wal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	completedDir = "completed"
)

// WAL allows for interaction with the WAL
type WAL struct {
	c *Config
}

// File represents a walfile entry
type File struct {
	Filepath string
	TenantID string
	BlockID  uuid.UUID
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

// AllWALFiles returns a File entry for every WAL file.  Used for replay.
func (w *WAL) AllWALFiles() ([]*File, error) {
	files, err := ioutil.ReadDir(w.c.Filepath)
	if err != nil {
		return nil, err
	}

	walfiles := make([]*File, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		blockID, tenantID, err := parseFilename(name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", name, err)
		}

		walfiles = append(walfiles, &File{
			BlockID:  blockID,
			TenantID: tenantID,
			Filepath: w.c.Filepath,
		})
	}

	return walfiles, nil
}

// NewBlock creates and returns a new AppendBlock configured correctly to write to the WAL
func (w *WAL) NewBlock(id uuid.UUID, tenantID string) (*AppendBlock, error) {
	return newAppendBlock(id, tenantID, w.c.Filepath)
}

// NewBlockWithWalFile creates an AppendBlock preloaded with contents of a WalFile
func (w *WAL) NewBlockWithWalFile(f *File) (*AppendBlock, error) {
	return newAppendBlockFromWal(f)
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
