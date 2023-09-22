package v2

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/pkg/errors"
)

type Compactor struct {
	opts common.CompactionOptions
}

var _ common.Compactor = (*Compactor)(nil)

func NewCompactor(opts common.CompactionOptions) *Compactor {
	return &Compactor{opts}
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta) (newCompactedBlocks []*backend.BlockMeta, err error) {
	tenantID := inputs[0].TenantID
	dataEncoding := inputs[0].DataEncoding // blocks chosen for compaction always have the same data encoding

	iters := make([]BytesIterator, 0, len(inputs))

	// cleanup compaction
	defer func() {
		for _, iter := range iters {
			iter.Close()
		}
	}()

	var compactionLevel uint8
	var totalRecords int
	for _, blockMeta := range inputs {
		totalRecords += blockMeta.TotalObjects

		if blockMeta.CompactionLevel > compactionLevel {
			compactionLevel = blockMeta.CompactionLevel
		}

		// Open iterator
		block, err := NewBackendBlock(blockMeta, r)
		if err != nil {
			return nil, err
		}

		iter, err := block.Iterator(c.opts.ChunkSizeBytes)
		if err != nil {
			return nil, err
		}

		iters = append(iters, iter)
	}

	nextCompactionLevel := compactionLevel + 1

	recordsPerBlock := (totalRecords / int(c.opts.OutputBlocks))

	combiner := c.opts.Combiner
	if combiner == nil {
		combiner = model.StaticCombiner
	}

	var currentBlock *StreamingBlock
	var tracker backend.AppendTracker

	iter := NewMultiblockIterator(ctx, iters, c.opts.IteratorBufferSize, combiner, dataEncoding, l)
	defer iter.Close()

	for {

		id, body, err := iter.NextBytes(ctx)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, "error iterating input blocks")
		}

		// make a new block if necessary
		if currentBlock == nil {
			currentBlock, err = NewStreamingBlock(&c.opts.BlockConfig, uuid.New(), tenantID, inputs, recordsPerBlock)
			if err != nil {
				return nil, errors.Wrap(err, "error making new compacted block")
			}
			currentBlock.BlockMeta().CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.BlockMeta())
		}

		err = currentBlock.AddObject(id, body)
		if err != nil {
			return nil, err
		}

		// write partial block
		if currentBlock.CurrentBufferLength() >= int(c.opts.FlushSizeBytes) {
			runtime.GC()
			tracker, err = c.appendBlock(ctx, writerCallback, tracker, currentBlock)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		// ship block to backend if done
		if currentBlock.Length() >= recordsPerBlock {
			err = c.finishBlock(ctx, writerCallback, tracker, currentBlock, l)
			if err != nil {
				return nil, errors.Wrap(err, "error shipping block to backend")
			}
			currentBlock = nil
			tracker = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		err = c.finishBlock(ctx, writerCallback, tracker, currentBlock, l)
		if err != nil {
			return nil, errors.Wrap(err, "error shipping block to backend")
		}
	}

	return newCompactedBlocks, nil
}

func (c *Compactor) appendBlock(ctx context.Context, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, tracker backend.AppendTracker, block *StreamingBlock) (backend.AppendTracker, error) {
	compactionLevel := int(block.BlockMeta().CompactionLevel - 1)

	if c.opts.ObjectsWritten != nil {
		c.opts.ObjectsWritten(compactionLevel, block.CurrentBufferedObjects())
	}

	tracker, bytesFlushed, err := block.FlushBuffer(ctx, tracker, writerCallback(block.BlockMeta(), time.Now()))
	if err != nil {
		return nil, err
	}

	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}

	return tracker, nil
}

func (c *Compactor) finishBlock(ctx context.Context, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, tracker backend.AppendTracker, block *StreamingBlock, l log.Logger) error {
	level.Info(l).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", block.BlockMeta()))

	bytesFlushed, err := block.Complete(ctx, tracker, writerCallback(block.BlockMeta(), time.Now()))
	if err != nil {
		return err
	}

	if c.opts.BytesWritten != nil {
		compactionLevel := int(block.BlockMeta().CompactionLevel - 1)
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}

	return nil
}
