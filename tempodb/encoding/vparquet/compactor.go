package vparquet

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func NewCompactor(opts common.CompactionOptions) common.Compactor {
	return &Compactor{opts: opts}
}

type Compactor struct {
	opts common.CompactionOptions
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta) (newCompactedBlocks []*backend.BlockMeta, err error) {

	var minBlockStart, maxBlockEnd time.Time
	bookmarks := make([]*bookmark, 0, len(inputs))

	var compactionLevel uint8
	var totalRecords int
	for _, blockMeta := range inputs {
		totalRecords += blockMeta.TotalObjects

		if blockMeta.CompactionLevel > compactionLevel {
			compactionLevel = blockMeta.CompactionLevel
		}

		if blockMeta.StartTime.Before(minBlockStart) || minBlockStart.IsZero() {
			minBlockStart = blockMeta.StartTime
		}
		if blockMeta.EndTime.After(maxBlockEnd) {
			maxBlockEnd = blockMeta.EndTime
		}

		block := newBackendBlock(blockMeta, r)

		iter, err := block.Iterator(ctx)
		if err != nil {
			return nil, err
		}

		// wrap bookmark with a prefetch iterator
		bookmarks = append(bookmarks, newBookmark(newPrefetchIterator(ctx, iter, c.opts.IteratorBufferSize/len(inputs))))
	}

	nextCompactionLevel := compactionLevel + 1

	recordsPerBlock := (totalRecords / int(c.opts.OutputBlocks))

	var currentBlock *streamingBlock
	m := newMultiblockIterator(bookmarks)
	defer m.Close()

	for {
		lowestObject, err := m.Next(ctx)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, "error iterating input blocks")
		}

		// make a new block if necessary
		if currentBlock == nil {
			// Start with a copy and then customize
			newMeta := &backend.BlockMeta{
				BlockID:         uuid.New(),
				TenantID:        inputs[0].TenantID,
				CompactionLevel: nextCompactionLevel,
				TotalObjects:    recordsPerBlock, // Just an estimate
			}
			w := writerCallback(newMeta, time.Now())

			currentBlock = newStreamingBlock(ctx, &c.opts.BlockConfig, newMeta, r, w, tempo_io.NewBufferedWriter)
			currentBlock.meta.CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.meta)
		}

		// Write trace.
		// Note - not specifying trace start/end here, we set the overall block start/stop
		// times from the input metas.
		err = currentBlock.Add(lowestObject, 0, 0)
		if err != nil {
			return nil, err
		}

		// write partial block
		//if currentBlock.CurrentBufferLength() >= int(opts.FlushSizeBytes) {
		if currentBlock.CurrentBufferedObjects() > 10000 {
			runtime.GC()
			err = c.appendBlock(currentBlock)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		// ship block to backend if done
		if currentBlock.meta.TotalObjects >= recordsPerBlock {
			currentBlockPtrCopy := currentBlock
			currentBlockPtrCopy.meta.StartTime = minBlockStart
			currentBlockPtrCopy.meta.EndTime = maxBlockEnd
			err := c.finishBlock(currentBlockPtrCopy, l)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("error shipping block to backend, blockID %s", currentBlockPtrCopy.meta.BlockID.String()))
			}
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		currentBlock.meta.StartTime = minBlockStart
		currentBlock.meta.EndTime = maxBlockEnd
		err := c.finishBlock(currentBlock, l)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("error shipping block to backend, blockID %s", currentBlock.meta.BlockID.String()))
		}
	}

	return newCompactedBlocks, nil
}

func (c *Compactor) appendBlock(block *streamingBlock) error {
	compactionLevel := int(block.meta.CompactionLevel - 1)
	if c.opts.ObjectsWritten != nil {
		c.opts.ObjectsWritten(compactionLevel, block.CurrentBufferedObjects())
	}

	bytesFlushed, err := block.Flush()
	if err != nil {
		return err
	}

	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}

	return nil
}

func (c *Compactor) finishBlock(block *streamingBlock, l log.Logger) error {
	bytesFlushed, err := block.Complete()
	if err != nil {
		return errors.Wrap(err, "error completing block")
	}

	level.Info(l).Log("msg", "wrote compacted block", "meta", fmt.Sprintf("%+v", block.meta))
	compactionLevel := int(block.meta.CompactionLevel) - 1
	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}
	return nil
}

type bookmark struct {
	iter Iterator

	currentObject *Trace
	currentErr    error
}

func newBookmark(iter Iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) (*Trace, error) {
	if b.currentErr != nil {
		return nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentObject, nil
	}

	b.currentObject, b.currentErr = b.iter.Next(ctx)
	return b.currentObject, b.currentErr
}

func (b *bookmark) done(ctx context.Context) bool {
	obj, err := b.current(ctx)

	return obj == nil || err != nil
}

func (b *bookmark) clear() {
	b.currentObject = nil
}

func (b *bookmark) close() {
	b.iter.Close()
}
