package tempodb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricCompactionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "tempodb",
		Name:      "compaction_duration_seconds",
		Help:      "Records the amount of time to compact a set of blocks.",
		Buckets:   prometheus.ExponentialBuckets(30, 2, 10),
	}, []string{"level"})
	metricCompactionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_errors_total",
		Help:      "Total number of errors occurring during compaction.",
	})
)

const (
	inputBlocks  = 2
	outputBlocks = 1

	recordsPerBatch = 1000
	compactionCycle = 30 * time.Second
)

// todo: pass a context/chan in to cancel this cleanly
func (rw *readerWriter) compactionLoop() {
	ticker := time.NewTicker(compactionCycle)
	for range ticker.C {
		rw.doCompaction()
		rw.doRetention()
	}
}

func (rw *readerWriter) doCompaction() {
	tenants := rw.blocklistTenants()
	if len(tenants) == 0 {
		return
	}

	// pick a random tenant and find some blocks to compact
	rand.Seed(time.Now().Unix())
	tenantID := tenants[rand.Intn(len(tenants))].(string)
	blocklist := rw.blocklist(tenantID)
	blockSelector := newTimeWindowBlockSelector(blocklist, rw.compactorCfg.MaxCompactionRange, rw.compactorCfg.MaxCompactionObjects)

	for {
		toBeCompacted, hashString := blockSelector.BlocksToCompact()
		if len(toBeCompacted) == 0 {
			break
		}
		if !rw.compactorSharder.Owns(hashString) {
			// continue on this tenant until we find something we own
			continue
		}
		level.Info(rw.logger).Log("msg", "Compacting hash", "hashString", hashString)
		if err := rw.compact(toBeCompacted, tenantID); err != nil {
			level.Error(rw.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}
		// after one compaction break to give other tenants an opportunity
		break
	}
}

// todo : this method is brittle and has weird failure conditions.  if it fails after it has written a new block then it will not clean up the old
//   in these cases it's possible that the compact method actually will start making more blocks.
func (rw *readerWriter) compact(blockMetas []*encoding.BlockMeta, tenantID string) error {
	level.Debug(rw.logger).Log("msg", "beginning compaction", "num blocks compacting", len(blockMetas))

	if len(blockMetas) == 0 {
		return nil
	}

	compactionLevel := compactionLevelForBlocks(blockMetas)
	nextCompactionLevel := compactionLevel + 1

	start := time.Now()
	defer func() {
		level.Info(rw.logger).Log("msg", "compaction complete")
		metricCompactionDuration.WithLabelValues(strconv.Itoa(int(compactionLevel))).Observe(time.Since(start).Seconds())
	}()

	var err error
	bookmarks := make([]*bookmark, 0, len(blockMetas))

	var totalRecords int
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects

		iter, err := encoding.NewBackendIterator(tenantID, blockMeta.BlockID, rw.compactorCfg.ChunkSizeBytes, rw.r)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, newBookmark(iter))

		_, err = rw.r.BlockMeta(blockMeta.BlockID, tenantID)
		if err == backend.ErrMetaDoesNotExist {
			// if meta doesn't exist right now it probably means this block was compacted.  warn and bail
			level.Warn(rw.logger).Log("msg", "unable to find meta during compaction", "blockID", blockMeta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
			return nil
		} else if err != nil {
			return err
		}
	}

	recordsPerBlock := (totalRecords / outputBlocks)
	var currentBlock *wal.CompactorBlock
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
				return errors.Wrap(err, "error making new compacted block")
			}
			currentBlock.BlockMeta().CompactionLevel = nextCompactionLevel
		}

		// writing to the current block will cause the id to escape the iterator so we need to make a copy of it
		writeID := append([]byte(nil), lowestID...)
		err = currentBlock.Write(writeID, lowestObject)
		if err != nil {
			return err
		}
		lowestBookmark.clear()

		// write partial block
		if currentBlock.Length()%recordsPerBatch == 0 {
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
	for _, meta := range blockMetas {
		if err := rw.c.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			level.Error(rw.logger).Log("msg", "unable to mark block compacted", "blockID", meta.BlockID, "tenantID", tenantID, "err", err)
			metricCompactionErrors.Inc()
		}
	}

	return nil
}

func appendBlock(rw *readerWriter, tracker backend.AppendTracker, block *wal.CompactorBlock) (backend.AppendTracker, error) {
	tracker, err := rw.w.AppendObject(context.TODO(), tracker, block.BlockMeta(), block.CurrentBuffer())
	if err != nil {
		return nil, err
	}
	block.ResetBuffer()

	return tracker, nil
}

func finishBlock(rw *readerWriter, tracker backend.AppendTracker, block *wal.CompactorBlock) error {
	level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", block.BlockMeta()))

	tracker, err := appendBlock(rw, tracker, block)
	if err != nil {
		return err
	}
	block.Complete()

	err = rw.WriteBlockMeta(context.TODO(), tracker, block) // todo:  add timeout
	if err != nil {
		return err
	}
	err = block.Clear()
	if err != nil {
		level.Error(rw.logger).Log("msg", "error cleaning up currentBlock in compaction", "err", err)
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

func compactionLevelForBlocks(blockMetas []*encoding.BlockMeta) uint8 {
	level := uint8(0)

	for _, m := range blockMetas {
		if m.CompactionLevel > level {
			level = m.CompactionLevel
		}
	}

	return level
}
