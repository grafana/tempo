package vparquet

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/metrics"
)

func NewCompactor() common.Compactor {
	return &Compactor{}
}

type Compactor struct {
}

func (c *Compactor) Compact(ctx context.Context, l log.Logger, r backend.Reader, writerCallback func(*backend.BlockMeta, time.Time) backend.Writer, inputs []*backend.BlockMeta, opts common.CompactionOptions) (newCompactedBlocks []*backend.BlockMeta, err error) {

	bookmarks := make([]*bookmark, 0, len(inputs))

	var compactionLevel uint8
	var totalRecords int
	for _, blockMeta := range inputs {
		totalRecords += blockMeta.TotalObjects

		if blockMeta.CompactionLevel > compactionLevel {
			compactionLevel = blockMeta.CompactionLevel
		}

		block, err := NewBackendBlock(blockMeta, r)
		if err != nil {
			return nil, err
		}

		iter, err := block.Iterator(ctx)
		if err != nil {
			return nil, err
		}

		bookmarks = append(bookmarks, newBookmark(iter))
	}

	nextCompactionLevel := compactionLevel + 1

	recordsPerBlock := (totalRecords / int(opts.OutputBlocks))

	var currentBlock *streamingBlock
	m := NewMultiblockPrefetchIterator(ctx, bookmarks, opts.PrefetchTraceCount)
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

			currentBlock, err = NewStreamingBlock(ctx, &opts.BlockConfig, newMeta, r, w, tempo_io.NewBufferedWriterWithQueue)
			if err != nil {
				return nil, errors.Wrap(err, "error making new compacted block")
			}
			currentBlock.meta.CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.meta)
		}

		err = currentBlock.Add(lowestObject)
		if err != nil {
			return nil, err
		}

		// write partial block
		//if currentBlock.CurrentBufferLength() >= int(opts.FlushSizeBytes) {
		if currentBlock.CurrentBufferedObjects() > 10000 {
			runtime.GC()
			err = appendBlock(currentBlock)
			if err != nil {
				return nil, errors.Wrap(err, "error writing partial block")
			}
		}

		// ship block to backend if done
		if currentBlock.meta.TotalObjects >= recordsPerBlock {
			currenBlockPtrCopy := currentBlock
			go finishBlock(currenBlockPtrCopy, l)
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		go finishBlock(currentBlock, l)
	}

	return newCompactedBlocks, nil
}

func appendBlock(block *streamingBlock) error {
	compactionLevelLabel := strconv.Itoa(int(block.meta.CompactionLevel - 1))
	metrics.MetricCompactionObjectsWritten.WithLabelValues(compactionLevelLabel).Add(float64(block.CurrentBufferedObjects()))

	bytesFlushed, err := block.Flush()
	if err != nil {
		return err
	}
	metrics.MetricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

	return nil
}

func finishBlock(block *streamingBlock, l log.Logger) {
	bytesFlushed, err := block.Complete()
	if err != nil {
		level.Error(l).Log("msg", "error shipping block to backend", "blockID", block.meta.BlockID.String(), "err", err)
		metrics.MetricCompactionErrors.Inc()
	}

	level.Info(l).Log("msg", "wrote compacted block", "meta", fmt.Sprintf("%+v", block.meta))
	compactionLevelLabel := strconv.Itoa(int(block.meta.CompactionLevel) - 1)
	metrics.MetricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))
}
