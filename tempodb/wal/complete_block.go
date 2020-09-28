package wal

import (
	"os"
	"time"

	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/willf/bloom"
	"go.uber.org/atomic"
)

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable.
// A CompleteBlock also knows the filepath of the append wal file it was cut from.  It is responsible for
// cleaning this block up once it has been flushed to the backend.
type CompleteBlock struct {
	block

	bloom   *bloom.BloomFilter
	records []*encoding.Record

	flushedTime atomic.Int64 // protecting flushedTime b/c it's accessed from the store on flush and from the ingester instance checking flush time
	walFilename string
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
		_ = c.readFile.Close()
	}

	name := c.fullFilename()
	return os.Remove(name)
}

func (c *CompleteBlock) FlushedTime() time.Time {
	unixTime := c.flushedTime.Load()
	if unixTime == 0 {
		return time.Time{} // return 0 time.  0 unix time is jan 1, 1970
	}
	return time.Unix(unixTime, 0)
}

func (c *CompleteBlock) Flushed() error {
	c.flushedTime.Store(time.Now().Unix())
	return os.Remove(c.walFilename) // now that we are flushed, remove our wal file
}

func (c *CompleteBlock) BlockMeta() *encoding.BlockMeta {
	return c.meta
}

func (c *CompleteBlock) BloomFilter() *bloom.BloomFilter {
	return c.bloom
}
