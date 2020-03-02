package wal

import (
	"os"
	"time"

	bloom "github.com/dgraph-io/ristretto/z"
	"github.com/grafana/frigg/friggdb/backend"
)

// completeBlock represent a block that has been "cut", is ready to be flushed and is not appendable
type completeBlock struct {
	block

	bloom       *bloom.Bloom
	records     []*backend.Record
	timeWritten time.Time
}

type ReplayBlock interface {
	Iterator() (backend.Iterator, error)
	TenantID() string
	Clear() error
}

type CompleteBlock interface {
	WriteableBlock
	ReplayBlock

	Find(id backend.ID) ([]byte, error)
	TimeWritten() time.Time
}

type WriteableBlock interface {
	BlockMeta() *backend.BlockMeta
	BloomFilter() *bloom.Bloom
	BlockWroteSuccessfully(t time.Time)
	Records() []*backend.Record
	ObjectFilePath() string
}

func (c *completeBlock) TenantID() string {
	return c.meta.TenantID
}

func (c *completeBlock) Records() []*backend.Record {
	return c.records
}

func (c *completeBlock) ObjectFilePath() string {
	return c.fullFilename()
}

func (c *completeBlock) Find(id backend.ID) ([]byte, error) {
	file, err := c.file()
	if err != nil {
		return nil, err
	}

	finder := backend.NewFinder(c.records, file)

	return finder.Find(id)
}

func (c *completeBlock) Iterator() (backend.Iterator, error) {
	name := c.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	return backend.NewIterator(f), nil
}

func (c *completeBlock) Clear() error {
	if c.readFile != nil {
		err := c.readFile.Close()
		if err != nil {
			return err
		}
	}

	name := c.fullFilename()
	return os.Remove(name)
}

func (c *completeBlock) TimeWritten() time.Time {
	return c.timeWritten
}

func (c *completeBlock) BlockWroteSuccessfully(t time.Time) {
	c.timeWritten = t
}

func (c *completeBlock) BlockMeta() *backend.BlockMeta {
	return c.meta
}

func (c *completeBlock) BloomFilter() *bloom.Bloom {
	return c.bloom
}
