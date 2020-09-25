package wal

import (
	"os"
	"time"

	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/willf/bloom"
)

type ReplayBlock interface {
	Iterator() (encoding.Iterator, error)
	TenantID() string
	Clear() error
}

type WriteableBlock interface {
	BlockMeta() *encoding.BlockMeta
	BloomFilter() *bloom.BloomFilter
	Records() []*encoding.Record
	ObjectFilePath() string

	Flushed(flushTime time.Time) error
}

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable.
// A CompleteBlock also knows the filepath of the wal block it was cut from.  It is responsible for
// cleaning this block up once it has been flushed to the backend.
type CompleteBlock struct {
	block

	bloom   *bloom.BloomFilter
	records []*encoding.Record

	flushedTime time.Time
	walFilename string
}

func (c *CompleteBlock) TenantID() string {
	return c.meta.TenantID
}

func (c *CompleteBlock) Records() []*encoding.Record {
	return c.records
}

func (c *CompleteBlock) ObjectFilePath() string {
	return c.fullFilename()
}

func (c *CompleteBlock) Find(id encoding.ID, combiner encoding.ObjectCombiner) ([]byte, error) {
	file, err := c.file()
	if err != nil {
		return nil, err
	}

	finder := encoding.NewDedupingFinder(c.records, file, combiner)

	return finder.Find(id)
}

func (c *CompleteBlock) Iterator() (encoding.Iterator, error) {
	name := c.fullFilename()
	f, err := os.OpenFile(name, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	return encoding.NewIterator(f), nil
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

func (c *CompleteBlock) FlushedTime() time.Time {
	return c.flushedTime
}

func (c *CompleteBlock) Flushed(flushTime time.Time) error {
	c.flushedTime = flushTime

	return os.Remove(c.walFilename) // now that we are flushed, remove our wal file
}

func (c *CompleteBlock) BlockMeta() *encoding.BlockMeta {
	return c.meta
}

func (c *CompleteBlock) BloomFilter() *bloom.BloomFilter {
	return c.bloom
}
