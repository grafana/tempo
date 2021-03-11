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

	filepath string
	readFile *os.File
	once     sync.Once

	cfg *BlockConfig
}

// NewCompleteBlock creates a new block and takes _ALL_ the parameters necessary to build the ordered, deduped file on disk
func NewCompleteBlock(cfg *BlockConfig, originatingMeta *backend.BlockMeta, iterator Iterator, estimatedObjects int, filepath string) (*CompleteBlock, error) {
	c := &CompleteBlock{
		encoding: latestEncoding(),
		meta:     backend.NewBlockMeta(originatingMeta.TenantID, uuid.New(), currentVersion, cfg.Encoding),
		bloom:    common.NewWithEstimates(uint(estimatedObjects), cfg.BloomFP),
		records:  make([]*common.Record, 0),
		filepath: filepath,
		cfg:      cfg,
	}

	appendFile, err := os.OpenFile(c.fullFilename(), os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer appendFile.Close()

	dataWriter, err := c.encoding.newDataWriter(appendFile, cfg.Encoding)
	if err != nil {
		return nil, err
	}

	appender, err := NewBufferedAppender(dataWriter, cfg.IndexDownsampleBytes, estimatedObjects)
	if err != nil {
		return nil, err
	}

	// todo: add a timeout?  propagage context from completing?
	ctx := context.Background()
	for {
		bytesID, bytesObject, err := iterator.Next(ctx)
		if bytesID == nil {
			break
		}
		if err != nil {
			_ = os.Remove(c.fullFilename())
			return nil, err
		}

		c.meta.ObjectAdded(bytesID)
		c.bloom.Add(bytesID)
		// obj gets written to disk immediately but the id escapes the iterator and needs to be copied
		writeID := append([]byte(nil), bytesID...)
		err = appender.Append(writeID, bytesObject)
		if err != nil {
			_ = os.Remove(c.fullFilename())
			return nil, err
		}
	}
	err = appender.Complete()
	if err != nil {
		return nil, err
	}
	c.records = appender.Records()
	c.meta.Size = appender.DataLength() // Must be after Complete()
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

	err = writeBlockData(ctx, w, c.meta, src, fileStat.Size())
	if err != nil {
		return err
	}

	indexWriter := c.encoding.newIndexWriter(c.cfg.IndexPageSizeBytes)
	indexBytes, err := indexWriter.Write(c.records)
	if err != nil {
		return err
	}

	c.meta.TotalRecords = uint32(len(c.records))
	c.meta.IndexPageSize = uint32(c.cfg.IndexPageSizeBytes)

	err = writeBlockMeta(ctx, w, c.meta, indexBytes, c.bloom)
	if err != nil {
		return err
	}

	// book keeping
	c.flushedTime.Store(time.Now().Unix())
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

	dataReader, err := c.encoding.newDataReader(backend.NewContextReaderWithAllReader(file), c.meta.Encoding)
	if err != nil {
		return nil, err
	}
	defer dataReader.Close()

	finder := NewPagedFinder(common.Records(c.records), dataReader, combiner)
	return finder.Find(context.Background(), id)
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
