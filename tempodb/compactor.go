package tempodb

import (
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
	"github.com/grafana/tempo/tempodb/encoding/common"
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
		Name:      "compaction_objects_written_total",
		Help:      "Total number of objects written to backend during compaction.",
	}, []string{"level"})
	metricCompactionBytesWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_bytes_written_total",
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

	DefaultFlushSizeBytes uint32 = 30 * 1024 * 1024 // 30 MiB

	DefaultIteratorBufferSize = 1000
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

	blockSelector := newTimeWindowBlockSelector(blocklist,
		rw.compactorCfg.MaxCompactionRange,
		rw.compactorCfg.MaxCompactionObjects,
		rw.compactorCfg.MaxBlockBytes,
		defaultMinInputBlocks,
		defaultMaxInputBlocks)

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

		if err == backend.ErrDoesNotExist {
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

func (rw *readerWriter) compact(blockMetas []*backend.BlockMeta, tenantID string) error {
	level.Debug(rw.logger).Log("msg", "beginning compaction", "num blocks compacting", len(blockMetas))

	// todo - add timeout?
	ctx := context.Background()

	if len(blockMetas) == 0 {
		return nil
	}

	compactionLevel := compactionLevelForBlocks(blockMetas)
	compactionLevelLabel := strconv.Itoa(int(compactionLevel))
	nextCompactionLevel := compactionLevel + 1

	var err error
	iters := make([]encoding.Iterator, 0, len(blockMetas))

	// cleanup compaction
	defer func() {
		level.Info(rw.logger).Log("msg", "compaction complete")
		for _, iter := range iters {
			iter.Close()
		}
	}()

	var totalRecords int
	var dataEncoding string
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects
		dataEncoding = blockMeta.DataEncoding // blocks chosen for compaction always have the same data encoding

		// Make sure block still exists
		_, err = rw.r.BlockMeta(ctx, blockMeta.BlockID, tenantID)
		if err != nil {
			return err
		}

		// Open iterator
		block, err := encoding.NewBackendBlock(blockMeta, rw.r)
		if err != nil {
			return err
		}

		iter, err := block.Iterator(rw.compactorCfg.ChunkSizeBytes)
		if err != nil {
			return err
		}

		iters = append(iters, iter)
	}

	recordsPerBlock := (totalRecords / outputBlocks)
	var newCompactedBlocks []*backend.BlockMeta
	var currentBlock *encoding.StreamingBlock
	var tracker backend.AppendTracker

	combiner := instrumentedObjectCombiner{
		inner:                rw.compactorSharder,
		compactionLevelLabel: compactionLevelLabel,
	}

	iter := encoding.NewMultiblockIterator(ctx, iters, rw.compactorCfg.IteratorBufferSize, combiner, dataEncoding)
	defer iter.Close()

	for {

		id, body, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "error iterating input blocks")
		}

		// make a new block if necessary
		if currentBlock == nil {
			currentBlock, err = encoding.NewStreamingBlock(rw.cfg.Block, uuid.New(), tenantID, blockMetas, recordsPerBlock)
			if err != nil {
				return errors.Wrap(err, "error making new compacted block")
			}
			currentBlock.BlockMeta().CompactionLevel = nextCompactionLevel
			newCompactedBlocks = append(newCompactedBlocks, currentBlock.BlockMeta())
		}

		err = currentBlock.AddObject(id, body)
		if err != nil {
			return err
		}

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

func appendBlock(rw *readerWriter, tracker backend.AppendTracker, block *encoding.StreamingBlock) (backend.AppendTracker, error) {
	compactionLevelLabel := strconv.Itoa(int(block.BlockMeta().CompactionLevel - 1))
	metricCompactionObjectsWritten.WithLabelValues(compactionLevelLabel).Add(float64(block.CurrentBufferedObjects()))

	tracker, bytesFlushed, err := block.FlushBuffer(context.TODO(), tracker, rw.w)
	if err != nil {
		return nil, err
	}
	metricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

	return tracker, nil
}

func finishBlock(rw *readerWriter, tracker backend.AppendTracker, block *encoding.StreamingBlock) error {
	level.Info(rw.logger).Log("msg", "writing compacted block", "block", fmt.Sprintf("%+v", block.BlockMeta()))

	w := rw.getWriterForBlock(block.BlockMeta(), time.Now())

	bytesFlushed, err := block.Complete(context.TODO(), tracker, w)
	if err != nil {
		return err
	}
	compactionLevelLabel := strconv.Itoa(int(block.BlockMeta().CompactionLevel - 1))
	metricCompactionBytesWritten.WithLabelValues(compactionLevelLabel).Add(float64(bytesFlushed))

	return nil
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

type instrumentedObjectCombiner struct {
	compactionLevelLabel string
	inner                common.ObjectCombiner
}

// Combine wraps the inner combiner with combined metrics
func (i instrumentedObjectCombiner) Combine(objA []byte, objB []byte, dataEncoding string) ([]byte, bool) {
	b, wasCombined := i.inner.Combine(objA, objB, dataEncoding)
	if wasCombined {
		metricCompactionObjectsCombined.WithLabelValues(i.compactionLevelLabel).Inc()
	}
	return b, wasCombined
}
