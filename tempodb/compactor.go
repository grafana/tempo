package tempodb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricCompactionBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_blocks_total",
		Help:      "Total number of blocks compacted.",
	}, []string{"level"})
	metricCompactionObjectsWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_objects_written",
		Help:      "Total number of objects written to backend during compaction.",
	}, []string{"level"})
	metricCompactionBytesWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_bytes_written",
		Help:      "Total number of bytes written to backend during compaction.",
	}, []string{"level"})
	metricCompactionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_errors_total",
		Help:      "Total number of errors occurring during compaction.",
	})
	metricCompactionObjectsCombined = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_objects_combined_total",
		Help:      "Total number of objects combined during compaction.",
	}, []string{"level"})
)

const (
	inputBlocks  = 2
	outputBlocks = 1

	compactionCycle = 30 * time.Second
)

// todo: pass a context/chan in to cancel this cleanly
func (rw *readerWriter) compactionLoop() {
	ticker := time.NewTicker(compactionCycle)
	for range ticker.C {
		rw.doCompaction()
	}
}

func (rw *readerWriter) doCompaction() {
	tenants := rw.blocklistTenants()
	if len(tenants) == 0 {
		return
	}

	// Iterate through tenants each cycle
	// Sort tenants for stability (since original map does not guarantee order)
	sort.Slice(tenants, func(i, j int) bool { return tenants[i].(string) < tenants[j].(string) })
	rw.compactorTenantOffset = (rw.compactorTenantOffset + 1) % uint(len(tenants))

	tenantID := tenants[rw.compactorTenantOffset].(string)
	blocklist := rw.blocklist(tenantID)

	blockSelector := newTimeWindowBlockSelector(blocklist, rw.compactorCfg.MaxCompactionRange, rw.compactorCfg.MaxCompactionObjects, defaultMinInputBlocks, defaultMaxInputBlocks)

	start := time.Now()

	level.Info(rw.logger).Log("msg", "starting compaction cycle", "tenantID", tenantID, "offset", rw.compactorTenantOffset)
	for {
		toBeCompacted, hashString := blockSelector.BlocksToCompact()
		if len(toBeCompacted) == 0 {
			level.Info(rw.logger).Log("msg", "compaction cycle complete. No more blocks to compact", "tenantID", tenantID)
			break
		}
		if !rw.compactorSharder.Owns(hashString) {
			// continue on this tenant until we find something we own
			continue
		}
		level.Info(rw.logger).Log("msg", "Compacting hash", "hashString", hashString)
		err := rw.compact(toBeCompacted, tenantID)

		if err == backend.ErrMetaDoesNotExist {
			level.Warn(rw.logger).Log("msg", "unable to find meta during compaction.  trying again on this block list", "err", err)
		} else if err != nil {
			level.Error(rw.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}

		// after a maintenance cycle bail out
		if start.Add(rw.cfg.BlocklistPoll).Before(time.Now()) {
			level.Info(rw.logger).Log("msg", "compacted blocks for a maintenance cycle, bailing out", "tenantID", tenantID)
			break
		}
	}
}

// todo : this method is brittle and has weird failure conditions.  if it fails after it has written a new block then it will not clean up the old
//   in these cases it's possible that the compact method actually will start making more blocks.
func (rw *readerWriter) compact(blockMetas []*backend.BlockMeta, tenantID string) error {
	level.Debug(rw.logger).Log("msg", "beginning compaction", "num blocks compacting", len(blockMetas))

	if len(blockMetas) == 0 {
		return nil
	}

	compactionLevel := compactionLevelForBlocks(blockMetas)
	compactionLevelLabel := strconv.Itoa(int(compactionLevel))
	nextCompactionLevel := compactionLevel + 1

	defer func() {
		level.Info(rw.logger).Log("msg", "compaction complete")
	}()

	var err error
	bookmarks := make([]*bookmark, 0, len(blockMetas))

	var totalRecords int
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects

		block, err := encoding.NewBackendBlock(blockMeta, rw.r)
		if err != nil {
			return err
		}

		iter, err := block.Iterator(rw.compactorCfg.ChunkSizeBytes)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, newBookmark(iter))

		_, err = rw.r.BlockMeta(context.TODO(), blockMeta.BlockID, tenantID)
		if err != nil {
			return err
		}
	}

	recordsPerBlock := (totalRecords / outputBlocks)
	var newCompactedBlocks []*backend.BlockMeta
	var currentBlock *encoding.CompactorBlock
	var tracker backend.AppendTracker

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

			if bytes.Equal(currentID, lowestID) {
				lowestObject = rw.compactorSharder.Combine(currentObject, lowestObject)
				b.clear()
				metricCompactionObjectsCombined.WithLabelValues(compactionLevelLabel).Inc()
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
			currentBlock, err = encoding.NewCompactorBlock(rw.cfg.Block, uuid.New(), tenantID, blockMetas, recordsPerBlock)
			if err != nil {
				return errors.Wrap(err, "error making new compacted block")
			}
			currentBlock.BlockMeta().CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.BlockMeta())
		}

		// writing to the current block will cause the id to escape the iterator so we need to make a copy of it
		writeID := append([]byte(nil), lowestID...)
		err = currentBlock.AddObject(writeID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clear()

		// write partial block
		if currentBlock.CurrentBufferLength() >= int(rw.compactorCfg.FlushSizeBytes) {
			tracker, err = appendBlock(rw, tracker, currentBlock)
			if err != nil {
				return errors.Wrap(err, "error writing partial block")
			}
		}

		// ship block to backend if done
		if currentBlock.Length() >= recordsPerBlock {
			err = finishBlock(rw, tracker, currentBlock)
			if err != nil {
				return errors.Wrap(err, "error shipping block to backend")
			}
			currentBlock = nil
			tracker = nil
		}
	}

	// ship final block to backend
	if currentBlock != nil {
		err = finishBlock(rw, tracker, currentBlock)
		if err != nil {
			return errors.Wrap(err, "error shipping block to backend")
		}
	}

	// mark old blocks compacted so they don't show up in polling
	markCompacted(rw, tenantID, blockMetas, newCompactedBlocks)

	metricCompactionBlocks.WithLabelValues(compactionLevelLabel).Add(float64(len(blockMetas)))

	return nil
}

func appendBlock(rw *readerWriter, tracker backend.AppendTracker, block *encoding.CompactorBlock) (backend.AppendTracker, error) {
	compactionLevelLabel := strconv.Itoa(int(block.BlockMeta().CompactionLevel - 1))
	metricCompactionObjectsWritten.WithLabelValues(compactionLevelLabel).Add(float64(block.CurrentBufferedObjects()))

	tracker, bytesFlushed, err := block.FlushBuffer(context.TODO(), tracker, rw.w)
	if err != nil {
		return nil, err
	}
	metricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

	return tracker, nil
}

func finishBlock(rw *readerWriter, tracker backend.AppendTracker, block *encoding.CompactorBlock) error {
	level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", block.BlockMeta()))

	bytesFlushed, err := block.Complete(context.TODO(), tracker, rw.w)
	if err != nil {
		return err
	}
	compactionLevelLabel := strconv.Itoa(int(block.BlockMeta().CompactionLevel - 1))
	metricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

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

func compactionLevelForBlocks(blockMetas []*backend.BlockMeta) uint8 {
	level := uint8(0)

	for _, m := range blockMetas {
		if m.CompactionLevel > level {
			level = m.CompactionLevel
		}
	}

	return level
}

func markCompacted(rw *readerWriter, tenantID string, oldBlocks []*backend.BlockMeta, newBlocks []*backend.BlockMeta) {
	for _, meta := range oldBlocks {
		// Mark in the backend
		if err := rw.c.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			level.Error(rw.logger).Log("msg", "unable to mark block compacted", "blockID", meta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
		}
	}

	// Converted outgoing blocks into compacted entries.
	newCompactions := make([]*backend.CompactedBlockMeta, 0, len(oldBlocks))
	for _, newBlock := range oldBlocks {
		newCompactions = append(newCompactions, &backend.CompactedBlockMeta{
			BlockMeta:     *newBlock,
			CompactedTime: time.Now(),
		})
	}

	// Update blocklist in memory
	rw.updateBlocklist(tenantID, newBlocks, oldBlocks, newCompactions)
}
