package wal

import (
	"context"
	"time"

	"go.uber.org/atomic"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

const nameFlushed = "flushed"

// LocalBlock is a block stored in a local storage.  It can be searched and flushed to a remote backend, and
// permanently tracks the flushed time with a special file in the block
type LocalBlock struct {
	encoding.BackendBlock
	local *local.Backend

	flushedTime atomic.Int64 // protecting flushedTime b/c it's accessed from the store on flush and from the ingester instance checking flush time
}

func NewLocalBlock(ctx context.Context, existingBlock *encoding.BackendBlock, l *local.Backend) (*LocalBlock, error) {

	c := &LocalBlock{
		BackendBlock: *existingBlock,
		local:        l,
	}

	flushedBytes, err := l.Read(ctx, nameFlushed, backend.KeyPathForBlock(c.BlockMeta().BlockID, c.BlockMeta().TenantID))
	if err == nil {
		flushedTime := time.Time{}
		err = flushedTime.UnmarshalText(flushedBytes)
		if err == nil {
			c.flushedTime.Store(flushedTime.Unix())
		}
	}

	return c, nil
}

func (c *LocalBlock) Find(ctx context.Context, id common.ID) ([]byte, error) {
	return c.BackendBlock.Find(ctx, id)
}

// FlushedTime returns the time the block was flushed.  Will return 0
//  if the block was never flushed
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
		return errors.Wrap(err, "error marshalling flush time to text")
	}

	err = c.local.Write(ctx, nameFlushed, backend.KeyPathForBlock(c.BlockMeta().BlockID, c.BlockMeta().TenantID), flushedBytes)
	if err != nil {
		return errors.Wrap(err, "error writing ingester block flushed file")
	}

	c.flushedTime.Store(flushedTime.Unix())
	return nil
}

func (c *LocalBlock) Write(ctx context.Context, w backend.Writer) error {
	err := encoding.CopyBlock(ctx, c.BlockMeta(), c.local, w)
	if err != nil {
		return errors.Wrap(err, "error copying block from local to remote backend")
	}

	err = c.SetFlushed(ctx)
	return err
}
