package friggdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricCompactionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "compaction_duration_seconds",
		Help:      "Records the amount of time to compact a set of blocks.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	metricCompactionStopDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "friggdb",
		Name:      "compaction_duration_stop_seconds",
		Help:      "Records the amount of time waiting on compaction jobs to stop.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	metricCompactionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "friggdb",
		Name:      "compaction_errors_total",
		Help:      "Total number of errors occurring during compaction.",
	})
)

const (
	inputBlocks  = 4
	outputBlocks = 2

	cursorDone = -1
)

func (rw *readerWriter) doCompaction() {
	// stop any existing compaction jobs
	if rw.jobStopper != nil {
		start := time.Now()
		err := rw.jobStopper.Stop()
		if err != nil {
			level.Warn(rw.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}
		metricCompactionStopDuration.Observe(time.Since(start).Seconds())
	}

	// start crazy jobs to do compaction with new list
	tenants := rw.blocklistTenants()

	var err error
	rw.jobStopper, err = rw.pool.RunStoppableJobs(tenants, func(payload interface{}, stopCh <-chan struct{}) error {
		var warning error
		tenantID := payload.(string)

		cursor := 0
	L:
		for {
			select {
			case <-stopCh:
				return warning
			default:
				var blocks []*backend.BlockMeta
				blocks, cursor = rw.blocksToCompact(tenantID, cursor) // todo: pass a context with a deadline?
				if cursor == cursorDone {
					break L
				}
				if blocks == nil {
					continue
				}
				err := rw.compact(blocks, tenantID)
				if err != nil {
					warning = err
					metricCompactionErrors.Inc()
				}
			}
		}

		return warning
	})

	if err != nil {
		level.Error(rw.logger).Log("msg", "failed to start compaction.  compaction broken until next maintenance cycle.", "err", err)
		metricCompactionErrors.Inc()
	}
}

// todo: metric to determine "effectiveness" of compaction.  i.e. total key overlap of blocks that is being eliminated?
//       switch to iterator pattern?
func (rw *readerWriter) blocksToCompact(tenantID string, cursor int) ([]*backend.BlockMeta, int) {
	// loop through blocks starting at cursor for the given tenant, blocks are sorted by start date so candidates for compaction should be near each other
	//   - consider candidateBlocks at a time.
	//   - find the blocks with the fewest records that are within the compaction range
	rw.blockListsMtx.Lock() // todo: there's lots of contention on this mutex.  keep an eye on this
	defer rw.blockListsMtx.Unlock()

	blocklist := rw.blockLists[tenantID]
	if inputBlocks > len(blocklist) {
		return nil, cursorDone
	}

	if cursor < 0 {
		return nil, cursorDone
	}

	cursorEnd := cursor + inputBlocks - 1
	for {
		if cursorEnd >= len(blocklist) {
			break
		}

		blockStart := blocklist[cursor]
		blockEnd := blocklist[cursorEnd]

		if blockEnd.EndTime.Sub(blockStart.StartTime) < rw.compactorCfg.MaxCompactionRange {
			return blocklist[cursor : cursorEnd+1], cursorEnd + 1
		}

		cursor++
		cursorEnd = cursor + inputBlocks - 1
	}

	return nil, cursorDone
}

// todo : this method is brittle and has weird failure conditions.  if it fails after it has written a new block then it will not clean up the old
//   in these cases it's possible that the compact method actually will start making more blocks.
func (rw *readerWriter) compact(blockMetas []*backend.BlockMeta, tenantID string) error {
	start := time.Now()
	defer func() {
		level.Info(rw.logger).Log("msg", "compaction complete")
		metricCompactionDuration.Observe(time.Since(start).Seconds())
	}()

	level.Info(rw.logger).Log("msg", "beginning compaction")

	var err error
	bookmarks := make([]*bookmark, 0, len(blockMetas))

	var totalRecords uint32
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects

		iter, err := backend.NewLazyIterator(tenantID, blockMeta.BlockID, rw.compactorCfg.ChunkSizeBytes, rw.r)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, newBookmark(iter))

		_, err = rw.r.BlockMeta(blockMeta.BlockID, tenantID)
		if os.IsNotExist(err) {
			// if meta doesn't exist right now it probably means this block was compacted.  warn and bail
			level.Warn(rw.logger).Log("msg", "unable to find meta during compaction", "blockID", blockMeta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
			return nil
		} else if err != nil {
			return err
		}
	}

	recordsPerBlock := (totalRecords / outputBlocks) + 1
	var currentBlock *compactorBlock

	for !allDone(bookmarks) {
		var lowestID []byte
		var lowestObject []byte
		var lowestBookmark *bookmark

		// find lowest ID of the new object
		for _, b := range bookmarks {
			currentID, currentObject, err := b.current()
			if err == io.EOF {
				continue
			} else if err != nil {
				return err
			}

			// todo:  right now if we run into equal ids we take the larger object in the hopes that it's a more complete trace.
			//   in the future add a callback or something that allows the owning application to make a more intelligent choice
			//   such as combining traces if they're both incomplete
			if bytes.Equal(currentID, lowestID) {
				if len(currentObject) > len(lowestObject) {
					lowestBookmark.clear()

					lowestID = currentID
					lowestObject = currentObject
					lowestBookmark = b
				} else {
					b.clear()
				}
			} else if len(lowestID) == 0 || bytes.Compare(currentID, lowestID) == -1 {
				lowestID = currentID
				lowestObject = currentObject
				lowestBookmark = b
			}
		}

		if len(lowestID) == 0 || len(lowestObject) == 0 || lowestBookmark == nil {
			return fmt.Errorf("failed to find a lowest object in compaction")
		}

		// make a new block if necessary
		if currentBlock == nil {
			h, err := rw.wal.NewWorkingBlock(uuid.New(), tenantID)
			if err != nil {
				return err
			}

			currentBlock, err = newCompactorBlock(h, rw.cfg.WAL.BloomFP, rw.cfg.WAL.IndexDownsample, blockMetas)
			if err != nil {
				return err
			}
		}

		// writing to the current block will cause the id is going to escape the iterator so we need to make a copy of it
		// lowestObject is going to be written to disk so we don't need to make a copy
		writeID := append([]byte(nil), lowestID...)
		err = currentBlock.write(writeID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clear()

		// ship block to backend if done
		if uint32(currentBlock.length()) >= recordsPerBlock {
			err = rw.writeCompactedBlock(currentBlock, tenantID)
			if err != nil {
				return err
			}
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		err = rw.writeCompactedBlock(currentBlock, tenantID)
		if err != nil {
			return err
		}
	}

	// mark old blocks compacted so they don't show up in polling
	for _, meta := range blockMetas {
		if err := rw.c.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			level.Error(rw.logger).Log("msg", "unable to mark block compacted", "blockID", meta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
		}
	}

	return nil
}

func (rw *readerWriter) writeCompactedBlock(b *compactorBlock, tenantID string) error {
	currentMeta := b.meta()
	level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", currentMeta))

	currentIndex, err := b.index()
	if err != nil {
		return err
	}

	currentBloom, err := b.bloom()
	if err != nil {
		return err
	}

	err = rw.w.Write(context.TODO(), b.id(), tenantID, currentMeta, currentBloom, currentIndex, b.objectFilePath())
	if err != nil {
		return err
	}

	err = b.clear()
	if err != nil {
		level.Warn(rw.logger).Log("msg", "failed to clear compacted bloc", "blockID", currentMeta.BlockID, "tenantID", tenantID, "err", err)
	}

	return nil
}

func allDone(bookmarks []*bookmark) bool {
	for _, b := range bookmarks {
		if !b.done() {
			return false
		}
	}

	return true
}
