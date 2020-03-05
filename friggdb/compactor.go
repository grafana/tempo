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
	"github.com/grafana/frigg/friggdb/wal"
	"github.com/grafana/frigg/pkg/util"
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
	metricRangeOfCompaction = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "friggdb",
		Name:      "compaction_id_range",
		Help:      "Total range of IDs compacted into a single block. (The smaller the better)",
	})
)

const (
	inputBlocks  = 4
	outputBlocks = 2
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

		if rw.blockSelector.IsRunning(tenantID) {
			level.Warn(rw.logger).Log("msg", "Previous cycle has not finished. Skipping maintenance cycle.")
			return nil
		}
	L:
		for {
			select {
			case <-stopCh:
				return warning
			default:
				var blocks []*backend.BlockMeta

				blocklist := rw.blocklist(tenantID)
				blocks = rw.blockSelector.BlocksToCompact(blocklist) // todo: pass a context with a deadline?
				if blocks == nil {
					break L
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

	var totalRecords int
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects

		iter, err := backend.NewBackendIterator(tenantID, blockMeta.BlockID, rw.compactorCfg.ChunkSizeBytes, rw.r)
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
	var currentBlock *wal.CompactorBlock

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
			currentBlock, err = rw.wal.NewCompactorBlock(uuid.New(), tenantID, blockMetas, recordsPerBlock)
			if err != nil {
				return err
			}
		}

		// writing to the current block will cause the id is going to escape the iterator so we need to make a copy of it
		// lowestObject is going to be written to disk so we don't need to make a copy
		writeID := append([]byte(nil), lowestID...)
		err = currentBlock.Write(writeID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clear()

		// ship block to backend if done
		if currentBlock.Length() >= recordsPerBlock {
			currentBlock.Complete()
			level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", currentBlock.BlockMeta()))
			err = rw.WriteBlock(context.TODO(), currentBlock) // todo:  add timeout
			if err != nil {
				return err
			}
			err = currentBlock.Clear()
			if err != nil {
				level.Error(rw.logger).Log("msg", "error cleaning up currentBlock in compaction", "err", err)
			}
			currentBlock = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		// Set the range of IDs as the metricRangeOfCompaction
		metricRangeOfCompaction.Set(
			util.BlockIDRange(currentBlock.BlockMeta().MaxID, currentBlock.BlockMeta().MinID),
		)
		currentBlock.Complete()
		level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", currentBlock.BlockMeta()))
		err = rw.WriteBlock(context.TODO(), currentBlock) // todo:  add timeout
		if err != nil {
			return err
		}
		err = currentBlock.Clear()
		if err != nil {
			level.Error(rw.logger).Log("msg", "error cleaning up currentBlock in compaction", "err", err)
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

func allDone(bookmarks []*bookmark) bool {
	for _, b := range bookmarks {
		if !b.done() {
			return false
		}
	}

	return true
}
