package tempodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/grafana/tempo/v3/pkg/dataquality"
	"github.com/grafana/tempo/v3/pkg/util/tracing"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/blockselector"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

const outputBlocks = 1

var tracer = otel.Tracer("tempodb/compactor")

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
	metricCompactionBlocksMissing = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_blocks_missing_total",
		Help:      "Total number of blocks skipped during compaction because their meta no longer exists.",
	}, []string{"tenant"})
	metricCompactionObjectsCombined = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_objects_combined_total",
		Help:      "Total number of objects combined during compaction.",
	}, []string{"level"})
	metricCompactionOutstandingBlocks = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "compaction_outstanding_blocks",
		Help:      "Number of blocks remaining to be compacted before next maintenance cycle",
	}, []string{"tenant"})
	metricDedupedSpans = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempodb",
		Name:      "compaction_spans_deduped_total",
		Help:      "Total number of spans that are deduped per replication factor.",
	}, []string{"replication_factor"})
	metricCompactionOutputBlockSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       "tempodb",
		Name:                            "compaction_output_block_size_bytes",
		Help:                            "Size in bytes of blocks produced by compaction.",
		Buckets:                         prometheus.ExponentialBuckets(1024*1024, 2, 10),
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: 1 * time.Hour,
	})
)

func (rw *readerWriter) CompactWithConfig(ctx context.Context, blockMetas []*backend.BlockMeta, tenantID string, compactorCfg *CompactorConfig, compactorSharder CompactorSharder, compactorOverrides CompactorOverrides) ([]*backend.BlockMeta, error) {
	level.Debug(rw.logger).Log("msg", "beginning compaction", "num blocks compacting", len(blockMetas))

	// todo - add timeout?
	ctx, span := tracer.Start(ctx, "rw.compact")
	defer span.End()

	traceID, _ := tracing.ExtractTraceID(ctx)
	if traceID != "" {
		level.Info(rw.logger).Log("msg", "beginning compaction", "traceID", traceID)
	}

	if len(blockMetas) == 0 {
		return nil, nil
	}

	var err error
	startTime := time.Now()

	// Drop blocks whose meta has disappeared since the block list was built. Another
	// compaction may already have compacted them, which is an expected race and must
	// not discard the blocks in the same batch that are still there.
	existing := make([]*backend.BlockMeta, 0, len(blockMetas))
	for _, blockMeta := range blockMetas {
		_, err = rw.r.BlockMeta(ctx, uuid.UUID(blockMeta.BlockID), tenantID)
		if err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) {
				metricCompactionBlocksMissing.WithLabelValues(tenantID).Inc()
				level.Warn(rw.logger).Log(
					"msg", "skipping block with no meta during compaction",
					"tenantID", tenantID,
					"blockID", blockMeta.BlockID.String(),
				)
				continue
			}
			return nil, err
		}
		existing = append(existing, blockMeta)
	}

	if len(existing) < len(blockMetas) {
		level.Warn(rw.logger).Log(
			"msg", "compacting remaining blocks after skipping missing metas",
			"tenantID", tenantID,
			"missing", len(blockMetas)-len(existing),
			"remaining", len(existing),
		)
		span.SetAttributes(
			attribute.Int("missing_blocks", len(blockMetas)-len(existing)),
			attribute.Int("remaining_blocks", len(existing)),
		)
	}

	if len(existing) < len(blockMetas) && len(existing) < 2 {
		return nil, nil
	}
	blockMetas = existing

	var totalRecords int
	for _, blockMeta := range blockMetas {
		level.Info(rw.logger).Log(
			"msg", "compacting block",
			"version", blockMeta.Version,
			"tenantID", blockMeta.TenantID,
			"blockID", blockMeta.BlockID.String(),
			"startTime", blockMeta.StartTime.String(),
			"endTime", blockMeta.EndTime.String(),
			"totalObjects", blockMeta.TotalObjects,
			"size", blockMeta.Size_,
			"compactionLevel", blockMeta.CompactionLevel,
			"totalRecords", blockMeta.TotalObjects,
			"bloomShardCount", blockMeta.BloomShardCount,
			"footerSize", blockMeta.FooterSize,
			"replicationFactor", blockMeta.ReplicationFactor,
		)
		totalRecords += int(blockMeta.TotalObjects)
	}

	enc, err := encoding.FromVersion(blockMetas[0].Version)
	if err != nil {
		return nil, err
	}

	if !enc.CompactionSupported() {
		return nil, fmt.Errorf("compaction not supported for block version %s", blockMetas[0].Version)
	}

	compactionLevel := CompactionLevelForBlocks(blockMetas)
	compactionLevelLabel := strconv.Itoa(int(compactionLevel))

	opts := common.CompactionOptions{
		BlockConfig:      *rw.cfg.Block,
		OutputBlocks:     outputBlocks,
		MaxBytesPerTrace: compactorOverrides.MaxBytesPerTraceForTenant(tenantID),
		BytesWritten: func(compactionLevel, bytes int) {
			metricCompactionBytesWritten.WithLabelValues(strconv.Itoa(compactionLevel)).Add(float64(bytes))
		},
		ObjectsCombined: func(compactionLevel, objs int) {
			metricCompactionObjectsCombined.WithLabelValues(strconv.Itoa(compactionLevel)).Add(float64(objs))
		},
		ObjectsWritten: func(compactionLevel, objs int) {
			metricCompactionObjectsWritten.WithLabelValues(strconv.Itoa(compactionLevel)).Add(float64(objs))
		},
		SpansDiscarded: func(traceId, rootSpanName, rootServiceName string, spans int) {
			compactorSharder.RecordDiscardedSpans(spans, tenantID, traceId, rootSpanName, rootServiceName)
		},
		DisconnectedTrace: func() {
			dataquality.WarnDisconnectedTrace(tenantID, dataquality.PhaseTraceCompactorCombine)
		},
		RootlessTrace: func() {
			dataquality.WarnRootlessTrace(tenantID, dataquality.PhaseTraceCompactorCombine)
		},
		DedupedSpans: func(replFactor, dedupedSpans int) {
			metricDedupedSpans.WithLabelValues(strconv.Itoa(replFactor)).Add(float64(dedupedSpans))
		},
	}

	compactor := enc.NewCompactor(opts)

	// Compact selected blocks into a larger one
	newCompactedBlocks, err := compactor.Compact(ctx, rw.logger, rw.r, rw.w, blockMetas)
	if err != nil {
		return nil, err
	}

	// mark old blocks compacted, so they don't show up in polling
	if err := markCompacted(rw, tenantID, blockMetas, newCompactedBlocks); err != nil {
		return nil, err
	}

	metricCompactionBlocks.WithLabelValues(compactionLevelLabel).Add(float64(len(blockMetas)))
	for _, meta := range newCompactedBlocks {
		metricCompactionOutputBlockSize.Observe(float64(meta.Size_))
	}

	logArgs := []interface{}{
		"msg",
		"compaction complete",
		"elapsed",
		time.Since(startTime),
	}
	for _, meta := range newCompactedBlocks {
		logArgs = append(logArgs, "blockID", meta.BlockID.String())
	}
	level.Info(rw.logger).Log(logArgs...)

	return newCompactedBlocks, nil
}

// MarkCompacted marks the old blocks as compacted and adds the new blocks to the blocklist.  No backend changes are made.
func (rw *readerWriter) MarkBlocklistCompacted(tenantID string, oldBlocks, newBlocks []*backend.BlockMeta) error {
	// Converted outgoing blocks into compacted entries.
	newCompactions := make([]*backend.CompactedBlockMeta, 0, len(oldBlocks))
	for _, newBlock := range oldBlocks {
		newCompactions = append(newCompactions, &backend.CompactedBlockMeta{
			BlockMeta:     *newBlock,
			CompactedTime: time.Now(),
		})
	}

	rw.blocklist.Update(tenantID, newBlocks, oldBlocks, newCompactions, nil)

	return nil
}

func markCompacted(rw *readerWriter, tenantID string, oldBlocks, newBlocks []*backend.BlockMeta) error {
	// Check if we have any errors, but continue marking the blocks as compacted
	var errCount int
	for _, meta := range oldBlocks {
		// Mark in the backend
		if err := rw.c.MarkBlockCompacted(uuid.UUID(meta.BlockID), tenantID); err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) {
				// A concurrent compaction or retention pass already retired this
				// block. The desired end state (input block retired) is already
				// true, so this isn't a real failure.
				continue
			}
			errCount++
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
	rw.blocklist.Update(tenantID, newBlocks, oldBlocks, newCompactions, nil)

	if errCount > 0 {
		return fmt.Errorf("unable to mark %d blocks compacted", errCount)
	}

	return nil
}

func MeasureOutstandingBlocks(tenantID string, blockSelector blockselector.CompactionBlockSelector, owned func(hash string) bool) {
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
	metricCompactionOutstandingBlocks.WithLabelValues(tenantID).Set(float64(totalOutstandingBlocks))
}

func CompactionLevelForBlocks(blockMetas []*backend.BlockMeta) uint8 {
	level := uint8(0)

	for _, m := range blockMetas {
		if m.CompactionLevel > uint32(level) {
			level = uint8(m.CompactionLevel)
		}
	}

	return level
}
