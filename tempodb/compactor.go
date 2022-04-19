package tempodb

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/metrics"
)

const (
	inputBlocks  = 2
	outputBlocks = 1

	DefaultCompactionCycle = 30 * time.Second

	DefaultFlushSizeBytes uint32 = 30 * 1024 * 1024 // 30 MiB

	DefaultIteratorBufferSize = 1000
)

// todo: pass a context/chan in to cancel this cleanly
func (rw *readerWriter) compactionLoop() {
	compactionCycle := DefaultCompactionCycle
	if rw.compactorCfg.CompactionCycle > 0 {
		compactionCycle = rw.compactorCfg.CompactionCycle
	}

	ticker := time.NewTicker(compactionCycle)
	for range ticker.C {
		rw.doCompaction()
	}
}

func (rw *readerWriter) doCompaction() {
	tenants := rw.blocklist.Tenants()
	if len(tenants) == 0 {
		return
	}

	// Iterate through tenants each cycle
	// Sort tenants for stability (since original map does not guarantee order)
	sort.Slice(tenants, func(i, j int) bool { return tenants[i] < tenants[j] })
	rw.compactorTenantOffset = (rw.compactorTenantOffset + 1) % uint(len(tenants))

	tenantID := tenants[rw.compactorTenantOffset]
	blocklist := rw.blocklist.Metas(tenantID)

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
			measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

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
			metrics.MetricCompactionErrors.Inc()
		}

		// after a maintenance cycle bail out
		if start.Add(rw.compactorCfg.MaxTimePerTenant).Before(time.Now()) {
			measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

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

	var err error

	defer func() {
		level.Info(rw.logger).Log("msg", "compaction complete")
	}()

	var totalRecords int
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log("msg", "compacting block", "block", fmt.Sprintf("%+v", blockMeta))
		totalRecords += blockMeta.TotalObjects

		// Make sure block still exists
		_, err = rw.r.BlockMeta(ctx, blockMeta.BlockID, tenantID)
		if err != nil {
			return err
		}
	}

	enc, err := encoding.FromVersion(blockMetas[0].Version)
	if err != nil {
		return err
	}

	/*combiner := instrumentedObjectCombiner{
		tenant:               tenantID,
		inner:                rw.compactorSharder,
		compactionLevelLabel: compactionLevelLabel,
	}*/

	compactor := enc.NewCompactor()
	opts := common.DefaultCompactionOptions()
	opts.BlockConfig = *rw.cfg.Block
	opts.ChunkSizeBytes = rw.compactorCfg.ChunkSizeBytes
	opts.FlushSizeBytes = rw.compactorCfg.FlushSizeBytes
	opts.OutputBlocks = outputBlocks
	newCompactedBlocks, err := compactor.Compact(ctx, rw.logger, rw.r, rw.getWriterForBlock, blockMetas, opts)
	if err != nil {
		return err
	}

	// mark old blocks compacted so they don't show up in polling
	markCompacted(rw, tenantID, blockMetas, newCompactedBlocks)

	compactionLabel := strconv.Itoa(int(newCompactedBlocks[0].CompactionLevel - 1))
	metrics.MetricCompactionBlocks.WithLabelValues(compactionLabel).Add(float64(len(blockMetas)))

	return nil
}

func markCompacted(rw *readerWriter, tenantID string, oldBlocks []*backend.BlockMeta, newBlocks []*backend.BlockMeta) {
	for _, meta := range oldBlocks {
		// Mark in the backend
		if err := rw.c.MarkBlockCompacted(meta.BlockID, tenantID); err != nil {
			level.Error(rw.logger).Log("msg", "unable to mark block compacted", "blockID", meta.BlockID, "tenantID", tenantID, "err", err)
			metrics.MetricCompactionErrors.Inc()
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
	rw.blocklist.Update(tenantID, newBlocks, oldBlocks, newCompactions, nil)
}

func measureOutstandingBlocks(tenantID string, blockSelector CompactionBlockSelector, owned func(hash string) bool) {
	// count number of per-tenant outstanding blocks before next maintenance cycle
	var totalOutstandingBlocks int
	for {
		leftToBeCompacted, hashString := blockSelector.BlocksToCompact()
		if len(leftToBeCompacted) == 0 {
			break
		}
		if !owned(hashString) {
			// continue on this tenant until we find something we own
			continue
		}
		totalOutstandingBlocks += len(leftToBeCompacted)
	}
	metrics.MetricCompactionOutstandingBlocks.WithLabelValues(tenantID).Set(float64(totalOutstandingBlocks))
}

/*type instrumentedObjectCombiner struct {
	tenant               string
	compactionLevelLabel string
	inner                CompactorSharder
}

// Combine wraps the inner combiner with combined metrics
func (i instrumentedObjectCombiner) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	b, wasCombined, err := i.inner.Combine(dataEncoding, i.tenant, objs...)
	if wasCombined {
		metricCompactionObjectsCombined.WithLabelValues(i.compactionLevelLabel).Inc()
	}
	return b, wasCombined, err
}*/
