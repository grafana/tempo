package wal

import (
	"os"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/grafana/frigg/friggdb/backend"
)

type ReplayBlock interface {
	Iterator() (backend.Iterator, error)
	TenantID() string
	Clear() error
}

type WriteableBlock interface {
	BlockMeta() *backend.BlockMeta
	BloomFilter() *bloom.Bloom
	BlockWroteSuccessfully(t time.Time)
	Records() []*backend.Record
	ObjectFilePath() string
}

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable
type CompleteBlock struct {
	block

	bloom       *bloom.Bloom
	records     []*backend.Record
	timeWritten time.Time
}

func (c *CompleteBlock) TenantID() string {
	return c.meta.TenantID
}

func (c *CompleteBlock) Records() []*backend.Record {
	return c.records
}

func (c *CompleteBlock) ObjectFilePath() string {
	return c.fullFilename()
}

func (c *CompleteBlock) Find(id backend.ID) ([]byte, error) {
	file, err := c.file()
	if err != nil {
		return nil, err
	}

	finder := backend.NewFinder(c.records, file)

	return finder.Find(id)
}

func (c *CompleteBlock) Iterator() (backend.Iterator, error) {
	name := c.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	return backend.NewIterator(f), nil
}

func (c *CompleteBlock) Clear() error {
	if c.readFile != nil {
		err := c.readFile.Close()
		if err != nil {
			return err
		}
	}

	name := c.fullFilename()
	return os.Remove(name)
}

func (c *CompleteBlock) TimeWritten() time.Time {
	return c.timeWritten
}

func (c *CompleteBlock) BlockWroteSuccessfully(t time.Time) {
	c.timeWritten = t
}

func (c *CompleteBlock) BlockMeta() *backend.BlockMeta {
	return c.meta
}

func (c *CompleteBlock) BloomFilter() *bloom.Bloom {
	return c.bloom
}
