package encoding

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// CompleteBlock represent a block that has been "cut", is ready to be flushed and is not appendable.
// A CompleteBlock also knows the filepath of the append wal file it was cut from.  It is responsible for
// cleaning this block up once it has been flushed to the backend.
type CompleteBlock struct {
	encoding versionedEncoding

	meta    *backend.BlockMeta
	bloom   *common.ShardedBloomFilter
	records []*common.Record

	flushedTime atomic.Int64 // protecting flushedTime b/c it's accessed from the store on flush and from the ingester instance checking flush time
	walFilename string

	filepath string
	readFile *os.File
	once     sync.Once
}

// NewCompleteBlock creates a new block and takes _ALL_ the parameters necessary to build the ordered, deduped file on disk
func NewCompleteBlock(cfg *BlockConfig, originatingMeta *backend.BlockMeta, iterator common.Iterator, estimatedObjects int, filepath string, walFilename string) (*CompleteBlock, error) {
	c := &CompleteBlock{
		encoding:    latestEncoding(),
		meta:        backend.NewBlockMeta(originatingMeta.TenantID, uuid.New(), currentVersion, cfg.Encoding),
		bloom:       common.NewWithEstimates(uint(estimatedObjects), cfg.BloomFP),
		records:     make([]*common.Record, 0),
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

	appender, err := c.encoding.newBufferedAppender(appendFile, cfg.Encoding, cfg.IndexDownsample, estimatedObjects)
	if err != nil {
		return nil, err
	}
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
	err = appender.Complete()
	if err != nil {
		return nil, err
	}
	appendFile.Close()
	c.records = appender.Records()
	c.meta.StartTime = originatingMeta.StartTime
	c.meta.EndTime = originatingMeta.EndTime

	return c, nil
}

// BlockMeta returns a pointer to this blocks meta
func (c *CompleteBlock) BlockMeta() *backend.BlockMeta {
	return c.meta
}

// Write implements WriteableBlock
func (c *CompleteBlock) Write(ctx context.Context, w backend.Writer) error {
	// write object file
	src, err := os.Open(c.fullFilename())
	if err != nil {
		return err
	}
	defer src.Close()

	fileStat, err := src.Stat()
	if err != nil {
		return err
	}

	err = c.encoding.writeBlockData(ctx, w, c.meta, src, fileStat.Size())
	if err != nil {
		return err
	}

	err = c.encoding.writeBlockMeta(ctx, w, c.meta, c.records, c.bloom)
	if err != nil {
		return err
	}

	// book keeping
	c.flushedTime.Store(time.Now().Unix())
	err = os.Remove(c.walFilename) // now that we are flushed, remove our wal file
	if err != nil {
		return err
	}

	return nil
}

// Find searches the for the provided trace id.  A CompleteBlock should never
//  have multiples of a single id so not sure why this uses a DedupingFinder.
func (c *CompleteBlock) Find(id common.ID, combiner common.ObjectCombiner) ([]byte, error) {
	if !c.bloom.Test(id) {
		return nil, nil
	}

	file, err := c.file()
	if err != nil {
		return nil, err
	}

	pageReader, err := c.encoding.newPageReader(file, c.meta.Encoding)
	if err != nil {
		return nil, err
	}

	finder := c.encoding.newPagedFinder(common.Records(c.records), pageReader, combiner)
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
