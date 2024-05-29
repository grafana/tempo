package ingester

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/atomic"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const nameFlushed = "flushed"

// LocalBlock is a block stored in a local storage.  It can be searched and flushed to a remote backend, and
// permanently tracks the flushed time with a special file in the block
type LocalBlock struct {
	common.BackendBlock
	reader backend.Reader
	writer backend.Writer

	flushedTime atomic.Int64 // protecting flushedTime b/c it's accessed from the store on flush and from the ingester instance checking flush time
}

var _ common.Finder = (*LocalBlock)(nil)

// NewLocalBlock creates a local block
func NewLocalBlock(ctx context.Context, existingBlock common.BackendBlock, l *local.Backend) *LocalBlock {
	c := &LocalBlock{
		BackendBlock: existingBlock,
		reader:       backend.NewReader(l),
		writer:       backend.NewWriter(l),
	}

	flushedBytes, err := c.reader.Read(ctx, nameFlushed, c.BlockMeta().BlockID, c.BlockMeta().TenantID, nil)
	if err == nil {
		flushedTime := time.Time{}
		err = flushedTime.UnmarshalText(flushedBytes)
		if err == nil {
			c.flushedTime.Store(flushedTime.Unix())
		}
	}

	return c
}

func (c *LocalBlock) FindTraceByID(ctx context.Context, id common.ID, opts common.SearchOptions) (*tempopb.Trace, error) {
	ctx, span := tracer.Start(ctx, "LocalBlock.FindTraceByID")
	defer span.End()
	return c.BackendBlock.FindTraceByID(ctx, id, opts)
}

// FlushedTime returns the time the block was flushed.  Will return 0
//
//	if the block was never flushed
func (c *LocalBlock) FlushedTime() time.Time {
	unixTime := c.flushedTime.Load()
	if unixTime == 0 {
		return time.Time{} // return 0 time.  0 unix time is jan 1, 1970
	}
	return time.Unix(unixTime, 0)
}

func (c *LocalBlock) SetFlushed(ctx context.Context) error {
	flushedTime := time.Now()
	flushedBytes, err := flushedTime.MarshalText()
	if err != nil {
		return fmt.Errorf("error marshalling flush time to text: %w", err)
	}

	err = c.writer.Write(ctx, nameFlushed, c.BlockMeta().BlockID, c.BlockMeta().TenantID, flushedBytes, nil)
	if err != nil {
		return fmt.Errorf("error writing ingester block flushed file: %w", err)
	}

	c.flushedTime.Store(flushedTime.Unix())
	return nil
}

func (c *LocalBlock) Write(ctx context.Context, w backend.Writer) error {
	err := encoding.CopyBlock(ctx, c.BlockMeta(), c.reader, w)
	if err != nil {
		return fmt.Errorf("error copying block from local to remote backend: %w", err)
	}

	err = c.SetFlushed(ctx)
	return err
}
