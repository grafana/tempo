package vparquet

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/metrics"
	"github.com/pkg/errors"
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

	allDone := func() bool {
		for _, b := range bookmarks {
			if !b.done() {
				return false
			}
		}
		return true
	}

	nextCompactionLevel := compactionLevel + 1

	recordsPerBlock := (totalRecords / int(opts.OutputBlocks))

	var currentBlock *streamingBlock

	for !allDone() {
		var lowestID string
		var lowestObjects []*Trace
		var lowestBookmarks []*bookmark

		// find lowest ID of the new object
		for _, b := range bookmarks {
			currentObject, err := b.current()
			if err != nil {
				return nil, err
			}
			if currentObject == nil {
				continue
			}

			comparison := strings.Compare(currentObject.TraceID, lowestID)

			if comparison == 0 {
				lowestObjects = append(lowestObjects, currentObject)
				lowestBookmarks = append(lowestBookmarks, b)
			} else if len(lowestID) == 0 || comparison == -1 {
				lowestID = currentObject.TraceID
				lowestObjects = []*Trace{currentObject}
				lowestBookmarks = []*bookmark{b}
			}
		}

		lowestObject := CombineTraces(lowestObjects...)
		for _, b := range lowestBookmarks {
			b.clear()
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

			currentBlock, err = NewStreamingBlock(ctx, &opts.BlockConfig, newMeta, w)
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
			err = finishBlock(currentBlock, l)
			if err != nil {
				return nil, errors.Wrap(err, "error shipping block to backend")
			}
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		err = finishBlock(currentBlock, l)
		if err != nil {
			return nil, errors.Wrap(err, "error shipping block to backend")
		}
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

func finishBlock(block *streamingBlock, l log.Logger) error {

	bytesFlushed, err := block.Complete()
	if err != nil {
		return err
	}

	level.Info(l).Log("msg", "wrote compacted block", "meta", fmt.Sprintf("%+v", block.meta))
	compactionLevelLabel := strconv.Itoa(int(block.meta.CompactionLevel) - 1)
	metrics.MetricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

	return nil
}

type bookmark struct {
	iter *iterator

	currentObject *Trace
	currentErr    error
}

func newBookmark(iter *iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current() (*Trace, error) {
	if b.currentErr != nil {
		return nil, b.currentErr
	}

	if b.currentObject != nil {
		return b.currentObject, nil
	}

	b.currentObject, b.currentErr = b.iter.Next()
	return b.currentObject, b.currentErr
}

func (b *bookmark) done() bool {
	obj, err := b.current()

	return obj == nil || err != nil
}

func (b *bookmark) clear() {
	b.currentObject = nil
}
