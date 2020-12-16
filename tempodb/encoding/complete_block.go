package encoding

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/encoding/bloom"
	"go.uber.org/atomic"
)

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable.
// A CompleteBlock also knows the filepath of the append wal file it was cut from.  It is responsible for
// cleaning this block up once it has been flushed to the backend.
type CompleteBlock struct {
	meta    *BlockMeta
	bloom   *bloom.ShardedBloomFilter
	records []*Record

	flushedTime atomic.Int64 // protecting flushedTime b/c it's accessed from the store on flush and from the ingester instance checking flush time
	walFilename string

	filepath string
	readFile *os.File
	once     sync.Once
}

// NewCompleteBlock creates a new block and takes _ALL_ the parameters necessary to build the ordered, deduped file on disk
func NewCompleteBlock(originatingMeta *BlockMeta, iterator Iterator, bloomFP float64, estimatedObjects int, indexDownsample int, filepath string, walFilename string) (*CompleteBlock, error) {
	c := &CompleteBlock{
		meta:        NewBlockMeta(originatingMeta.TenantID, uuid.New()),
		bloom:       bloom.NewWithEstimates(uint(estimatedObjects), bloomFP),
		records:     make([]*Record, 0),
		filepath:    filepath,
		walFilename: walFilename,
	}

	_, err := os.Create(c.fullFilename())
	if err != nil {
		return nil, err
	}

	appendFile, err := os.OpenFile(c.fullFilename(), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	appender := NewBufferedAppender(appendFile, indexDownsample, estimatedObjects)
	for {
		bytesID, bytesObject, err := iterator.Next()
		if bytesID == nil {
			break
		}
		if err != nil {
			_ = appendFile.Close()
			_ = os.Remove(c.fullFilename())
			return nil, err
		}

		c.meta.ObjectAdded(bytesID)
		c.bloom.Add(bytesID)
		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), bytesID...)
		err = appender.Append(writeID, bytesObject)
		if err != nil {
			_ = appendFile.Close()
			_ = os.Remove(c.fullFilename())
			return nil, err
		}
	}
	appender.Complete()
	appendFile.Close()
	c.records = appender.Records()
	c.meta.StartTime = originatingMeta.StartTime
	c.meta.EndTime = originatingMeta.EndTime

	return c, nil
}

// Records implements WriteableBlock
func (c *CompleteBlock) Records() []*Record {
	return c.records
}

// ObjectFilePath implements WriteableBlock
func (c *CompleteBlock) ObjectFilePath() string {
	return c.fullFilename()
}

// Flushed implements WriteableBlock.  Note that it also cleans up the wal file that this is
//  built off of
func (c *CompleteBlock) Flushed() error {
	c.flushedTime.Store(time.Now().Unix())
	return os.Remove(c.walFilename) // now that we are flushed, remove our wal file
}

// BlockMeta implements WriteableBlock
func (c *CompleteBlock) BlockMeta() *BlockMeta {
	return c.meta
}

// BloomFilter implements WriteableBlock
func (c *CompleteBlock) BloomFilter() *bloom.ShardedBloomFilter {
	return c.bloom
}

// Find searches the for the provided trace id.  A CompleteBlock should never
//  have multiples of a single id so not sure why this uses a DedupingFinder.
func (c *CompleteBlock) Find(id ID, combiner ObjectCombiner) ([]byte, error) {
	if !c.bloom.Test(id) {
		return nil, nil
	}

	file, err := c.file()
	if err != nil {
		return nil, err
	}

	finder := NewDedupingFinder(c.records, file, combiner)

	return finder.Find(id)
}

// Clear removes the backing file.
func (c *CompleteBlock) Clear() error {
	if c.readFile != nil {
		_ = c.readFile.Close()
	}

	name := c.fullFilename()
	return os.Remove(name)
}

// FlushedTime returns the time the block was flushed.  Will return 0
//  if the block was never flushed
func (c *CompleteBlock) FlushedTime() time.Time {
	unixTime := c.flushedTime.Load()
	if unixTime == 0 {
		return time.Time{} // return 0 time.  0 unix time is jan 1, 1970
	}
	return time.Unix(unixTime, 0)
}

func (c *CompleteBlock) fullFilename() string {
	return fmt.Sprintf("%s/%v:%v", c.filepath, c.meta.BlockID, c.meta.TenantID)
}

func (c *CompleteBlock) file() (*os.File, error) {
	var err error
	c.once.Do(func() {
		if c.readFile == nil {
			name := c.fullFilename()

			c.readFile, err = os.OpenFile(name, os.O_RDONLY, 0644)
		}
	})

	return c.readFile, err
}
