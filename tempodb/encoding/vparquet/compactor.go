package vparquet

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func NewCompactor(opts common.CompactionOptions) *Compactor {
	return &Compactor{opts: opts}
}

type Compactor struct {
	opts common.CompactionOptions
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta) (newCompactedBlocks []*backend.BlockMeta, err error) {

	var (
		compactionLevel uint8
		totalRecords    int
		minBlockStart   time.Time
		maxBlockEnd     time.Time
		bookmarks       = make([]*bookmark, 0, len(inputs))
		pool            = newRowPool(int(c.opts.FlushSizeBytes))
	)
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

		span, derivedCtx := opentracing.StartSpanFromContext(ctx, "vparquet.compactor.iterator")
		defer span.Finish()

		iter, err := block.RawIterator(derivedCtx, pool)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, newBookmark(iter))
	}

	var (
		nextCompactionLevel = compactionLevel + 1
		sch                 = parquet.SchemaOf(new(Trace))
	)

	// Dedupe rows and also call the metrics callback.
	combine := func(rows []parquet.Row) (parquet.Row, error) {
		if len(rows) == 0 {
			return nil, nil
		}

		if len(rows) == 1 {
			return rows[0], nil
		}

		// Time to combine.
		cmb := NewCombiner()
		for i, row := range rows {
			tr := new(Trace)
			err := sch.Reconstruct(tr, row)
			if err != nil {
				return nil, err
			}
			cmb.ConsumeWithFinal(tr, i == len(rows)-1)
			pool.Put(row)
		}
		tr, _ := cmb.Result()

		c.opts.ObjectsCombined(int(compactionLevel), 1)
		return sch.Deconstruct(pool.Get(), tr), nil
	}

	var (
		m               = newMultiblockIterator(bookmarks, combine)
		recordsPerBlock = (totalRecords / int(c.opts.OutputBlocks))
		currentBlock    *streamingBlock
	)
	defer m.Close()

	for {
		lowestID, lowestObject, err := m.Next(ctx)
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

		// Flush existing block data if the next trace can't fit
		// Here we repurpose FlushSizeBytes as number of raw column values.
		// This is a fairly close approximation.
		if currentBlock.CurrentBufferedValues() > 0 && currentBlock.CurrentBufferedValues()+len(lowestObject) > int(c.opts.FlushSizeBytes) {
			runtime.GC()
			err = c.appendBlock(ctx, currentBlock, l)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		// Write trace.
		// Note - not specifying trace start/end here, we set the overall block start/stop
		// times from the input metas.
		err = currentBlock.AddRaw(lowestID, lowestObject, 0, 0)
		if err != nil {
			return nil, err
		}

		// Flush again if block is already full.
		if currentBlock.CurrentBufferedValues() > int(c.opts.FlushSizeBytes) {
			runtime.GC()
			err = c.appendBlock(ctx, currentBlock, l)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		pool.Put(lowestObject)

		// ship block to backend if done
		if currentBlock.meta.TotalObjects >= recordsPerBlock {
			currentBlockPtrCopy := currentBlock
			currentBlockPtrCopy.meta.StartTime = minBlockStart
			currentBlockPtrCopy.meta.EndTime = maxBlockEnd
			err := c.finishBlock(ctx, currentBlockPtrCopy, l)
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
		err := c.finishBlock(ctx, currentBlock, l)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("error shipping block to backend, blockID %s", currentBlock.meta.BlockID.String()))
		}
	}

	return newCompactedBlocks, nil
}

func (c *Compactor) appendBlock(ctx context.Context, block *streamingBlock, l log.Logger) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "vparquet.compactor.appendBlock")
	defer span.Finish()

	var (
		objs            = block.CurrentBufferedObjects()
		vals            = block.CurrentBufferedValues()
		compactionLevel = int(block.meta.CompactionLevel - 1)
	)

	if c.opts.ObjectsWritten != nil {
		c.opts.ObjectsWritten(compactionLevel, objs)
	}

	bytesFlushed, err := block.Flush()
	if err != nil {
		return err
	}

	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}

	level.Info(l).Log("msg", "flushed to block", "bytes", bytesFlushed, "objects", objs, "values", vals)

	return nil
}

func (c *Compactor) finishBlock(ctx context.Context, block *streamingBlock, l log.Logger) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "vparquet.compactor.finishBlock")
	defer span.Finish()

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
	iter RawIterator

	currentID     common.ID
	currentObject parquet.Row
	currentErr    error
}

func newBookmark(iter RawIterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) ([]byte, parquet.Row, error) {
	if b.currentErr != nil {
		return nil, nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentID, b.currentObject, nil
	}

	b.currentID, b.currentObject, b.currentErr = b.iter.Next(ctx)
	return b.currentID, b.currentObject, b.currentErr
}

func (b *bookmark) done(ctx context.Context) bool {
	_, obj, err := b.current(ctx)

	return obj == nil || err != nil
}

func (b *bookmark) clear() {
	b.currentID = nil
	b.currentObject = nil
}

func (b *bookmark) close() {
	b.iter.Close()
}

type rowPool struct {
	pool    sync.Pool
	maxSize int
}

func newRowPool(maxRowSize int) *rowPool {
	return &rowPool{
		maxSize: maxRowSize,
		pool: sync.Pool{
			New: func() any {
				return make(parquet.Row, 0, maxRowSize)
			},
		},
	}
}

func (r *rowPool) Get() parquet.Row {
	return r.pool.Get().(parquet.Row)
}

func (r *rowPool) Put(row parquet.Row) {
	// Clear before putting into the pool.
	// This is important so that pool entries don't hang
	// onto the underlying buffers.
	for i := range row {
		row[i] = parquet.Value{}
	}
	r.pool.Put(row[:0])
}
