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
	BlockWroteSuccessfully(t time.Time) // jpe - wrote -> flushed
	Records() []*encoding.Record
	ObjectFilePath() string
}

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable
type CompleteBlock struct {
	block

	bloom       *bloom.BloomFilter
	records     []*encoding.Record
	timeWritten time.Time
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

func (c *CompleteBlock) TimeWritten() time.Time {
	return c.timeWritten
}

func (c *CompleteBlock) BlockWroteSuccessfully(t time.Time) {
	c.timeWritten = t
}

func (c *CompleteBlock) BlockMeta() *encoding.BlockMeta {
	return c.meta
}

func (c *CompleteBlock) BloomFilter() *bloom.BloomFilter {
	return c.bloom
}
