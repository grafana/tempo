package vparquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/opentracing/opentracing-go"
	"github.com/parquet-go/parquet-go"

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
		bookmarks       = make([]*bookmark[parquet.Row], 0, len(inputs))
		// MaxBytesPerTrace is the largest trace that can be expected, and assumes 1 byte per value on average (same as flushing).
		// Divide by 4 to presumably require 2 slice allocations if we ever see a trace this large
		pool = newRowPool(c.opts.MaxBytesPerTrace / 4)
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

		iter, err := block.rawIter(derivedCtx, pool)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, newBookmark[parquet.Row](iter))
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

		isEqual := true
		for i := 1; i < len(rows) && isEqual; i++ {
			isEqual = rows[0].Equal(rows[i])
		}
		if isEqual {
			for i := 1; i < len(rows); i++ {
				pool.Put(rows[i])
			}
			return rows[0], nil
		}

		// Total
		if c.opts.MaxBytesPerTrace > 0 {
			sum := 0
			for _, row := range rows {
				sum += estimateProtoSizeFromParquetRow(row)
			}
			if sum > c.opts.MaxBytesPerTrace {
				// Trace too large to compact
				for i := 1; i < len(rows); i++ {
					c.opts.SpansDiscarded(countSpans(sch, rows[i]))
					pool.Put(rows[i])
				}
				return rows[0], nil
			}
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
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error iterating input blocks: %w", err)
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
		if currentBlock.EstimatedBufferedBytes() > 0 && currentBlock.EstimatedBufferedBytes()+estimateMarshalledSizeFromParquetRow(lowestObject) > c.opts.BlockConfig.RowGroupSizeBytes {
			runtime.GC()
			err = c.appendBlock(ctx, currentBlock, l)
			if err != nil {
				return nil, fmt.Errorf("error writing partial block: %w", err)
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
		if currentBlock.EstimatedBufferedBytes() > c.opts.BlockConfig.RowGroupSizeBytes {
			runtime.GC()
			err = c.appendBlock(ctx, currentBlock, l)
			if err != nil {
				return nil, fmt.Errorf("error writing partial block: %w", err)
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
				return nil, fmt.Errorf("error shipping block to backend, blockID %s: %w", currentBlockPtrCopy.meta.BlockID.String(), err)
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
			return nil, fmt.Errorf("error shipping block to backend, blockID %s: %w", currentBlock.meta.BlockID.String(), err)
		}
	}

	return newCompactedBlocks, nil
}

func (c *Compactor) appendBlock(ctx context.Context, block *streamingBlock, l log.Logger) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "vparquet.compactor.appendBlock")
	defer span.Finish()

	var (
		objs            = block.CurrentBufferedObjects()
		vals            = block.EstimatedBufferedBytes()
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
		return fmt.Errorf("error completing block: %w", err)
	}

	level.Info(l).Log("msg", "wrote compacted block", "meta", fmt.Sprintf("%+v", block.meta))
	compactionLevel := int(block.meta.CompactionLevel) - 1
	if c.opts.BytesWritten != nil {
		c.opts.BytesWritten(compactionLevel, bytesFlushed)
	}
	return nil
}

type rowPool struct {
	pool sync.Pool
}

func newRowPool(defaultRowSize int) *rowPool {
	return &rowPool{
		pool: sync.Pool{
			New: func() any {
				return make(parquet.Row, 0, defaultRowSize)
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
	r.pool.Put(row[:0]) //nolint:all //SA6002
}

// estimateProtoSizeFromParquetRow estimates the byte-length of the corresponding
// trace in tempopb.Trace format. This method is unreasonably effective.
// Testing on real blocks shows 90-98% accuracy.
func estimateProtoSizeFromParquetRow(row parquet.Row) (size int) {
	for _, v := range row {
		size++ // Field identifier

		switch v.Kind() {
		case parquet.ByteArray:
			size += len(v.ByteArray())

		case parquet.FixedLenByteArray:
			size += len(v.ByteArray())

		default:
			// All other types (ints, bools) approach 1 byte per value
			size++
		}
	}
	return
}

// estimateMarshalledSizeFromParquetRow estimates the byte size as marshalled into parquet.
// this is a very rough estimate and is generally 66%-100% of actual size.
func estimateMarshalledSizeFromParquetRow(row parquet.Row) (size int) {
	return len(row)
}

// countSpans counts the number of spans in the given trace in deconstructed
// parquet row format and returns traceId.
// It simply counts the number of values for span ID, which is always present.
func countSpans(schema *parquet.Schema, row parquet.Row) (traceID, rootSpanName, rootServiceName string, spans int) {
	traceIDColumn, found := schema.Lookup(TraceIDColumnName)
	if !found {
		return "", "", "", 0
	}

	rootSpanNameColumn, found := schema.Lookup(columnPathRootSpanName)
	if !found {
		return "", "", "", 0
	}

	rootServiceNameColumn, found := schema.Lookup(columnPathRootServiceName)
	if !found {
		return "", "", "", 0
	}

	spanID, found := schema.Lookup("rs", "ils", "Spans", "ID")
	if !found {
		return "", "", "", 0
	}

	for _, v := range row {
		if v.Column() == spanID.ColumnIndex {
			spans++
		}

		if v.Column() == traceIDColumn.ColumnIndex {
			traceID = tempoUtil.TraceIDToHexString(v.ByteArray())
		}

		if v.Column() == rootSpanNameColumn.ColumnIndex {
			rootSpanName = v.String()
		}

		if v.Column() == rootServiceNameColumn.ColumnIndex {
			rootServiceName = v.String()
		}
	}

	return
}
