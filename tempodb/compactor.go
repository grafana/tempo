package tempodb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"

	"github.com/grafana/tempo/pkg/dataquality"
	"github.com/grafana/tempo/pkg/util/tracing"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	inputBlocks  = 2
	outputBlocks = 1

	DefaultCompactionCycle = 30 * time.Second

	DefaultChunkSizeBytes            = 5 * 1024 * 1024  // 5 MiB
	DefaultFlushSizeBytes     uint32 = 20 * 1024 * 1024 // 20 MiB
	DefaultIteratorBufferSize        = 1000
)

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
	metricDedupedSpans = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempodb",
		Name:      "compaction_spans_combined_total",
		Help:      "Number of spans that are deduped per replication factor.",
	}, []string{"replication_factor"})

	errCompactionJobNoLongerOwned = fmt.Errorf("compaction job no longer owned")
)

func (rw *readerWriter) compactionLoop(ctx context.Context) {
	compactionCycle := DefaultCompactionCycle
	if rw.compactorCfg.CompactionCycle > 0 {
		compactionCycle = rw.compactorCfg.CompactionCycle
	}

	for {
		// if the context is cancelled, we're shutting down and need to stop compacting
		if ctx.Err() != nil {
			break
		}

		doForAtLeast(ctx, compactionCycle, func() {
			rw.compactOneTenant(ctx)
		})
	}
}

// compactOneTenant runs a compaction cycle every 30s
func (rw *readerWriter) compactOneTenant(ctx context.Context) {
	// List of all tenants in the block list
	// The block list is updated by constant polling the storage for tenant indexes and/or tenant blocks (and building the index)
	tenants := rw.blocklist.Tenants()
	if len(tenants) == 0 {
		return
	}

	// Iterate through tenants each cycle
	// Sort tenants for stability (since original map does not guarantee order)
	sort.Slice(tenants, func(i, j int) bool { return tenants[i] < tenants[j] })
	rw.compactorTenantOffset = (rw.compactorTenantOffset + 1) % uint(len(tenants))

	// Select the next tenant to run compaction for
	tenantID := tenants[rw.compactorTenantOffset]

	// Skip compaction for tenants which have it disabled.
	if rw.compactorOverrides.CompactionDisabledForTenant(tenantID) {
		return
	}

	// Get the meta file of all non-compacted blocks for the given tenant
	blocklist := rw.blocklist.Metas(tenantID)

	window := rw.compactorOverrides.MaxCompactionRangeForTenant(tenantID)
	if window == 0 {
		window = rw.compactorCfg.MaxCompactionRange
	}

	// Select which blocks to compact.
	//
	// Blocks are firstly divided by the active compaction window (default: most recent 24h)
	//  1. If blocks are inside the active window, they're grouped by compaction level (how many times they've been compacted).
	//   Favoring lower compaction levels, and compacting blocks only from the same tenant.
	//  2. If blocks are outside the active window, they're grouped only by windows, ignoring compaction level.
	//   It picks more recent windows first, and compacting blocks only from the same tenant.
	blockSelector := newTimeWindowBlockSelector(blocklist,
		window,
		rw.compactorCfg.MaxCompactionObjects,
		rw.compactorCfg.MaxBlockBytes,
		defaultMinInputBlocks,
		defaultMaxInputBlocks)

	start := time.Now()

	level.Info(rw.logger).Log("msg", "starting compaction cycle", "tenantID", tenantID, "offset", rw.compactorTenantOffset)
	for {
		// this context is controlled by the service manager. it being cancelled means that the process is shutting down
		if ctx.Err() != nil {
			level.Info(rw.logger).Log("msg", "caught context cancelled at the top of the compaction loop. bailing.", "err", ctx.Err(), "cause", context.Cause(ctx))
			return
		}

		// Pick up to defaultMaxInputBlocks (4) blocks to compact into a single one
		toBeCompacted, hashString := blockSelector.BlocksToCompact()
		if len(toBeCompacted) == 0 {
			measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

			level.Info(rw.logger).Log("msg", "compaction cycle complete. No more blocks to compact", "tenantID", tenantID)
			return
		}

		owns := func() bool {
			return rw.compactorSharder.Owns(hashString)
		}
		if !owns() {
			// continue on this tenant until we find something we own
			continue
		}

		level.Info(rw.logger).Log("msg", "Compacting hash", "hashString", hashString)
		err := rw.compactWhileOwns(ctx, toBeCompacted, tenantID, owns)

		if errors.Is(err, backend.ErrDoesNotExist) {
			level.Warn(rw.logger).Log("msg", "unable to find meta during compaction. trying again on this block list", "err", err)
		} else if err != nil {
			level.Error(rw.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}

		// after a maintenance cycle bail out
		if start.Add(rw.compactorCfg.MaxTimePerTenant).Before(time.Now()) {
			measureOutstandingBlocks(tenantID, blockSelector, rw.compactorSharder.Owns)

			level.Info(rw.logger).Log("msg", "compacted blocks for a maintenance cycle, bailing out", "tenantID", tenantID)
			return
		}
	}
}

func (rw *readerWriter) compactWhileOwns(ctx context.Context, blockMetas []*backend.BlockMeta, tenantID string, owns func() bool) error {
	ownsCtx, cancel := context.WithCancelCause(ctx)

	done := make(chan struct{})
	defer close(done)

	// every second test if we still own the job. if we don't then cancel the context with a cause
	// that we can then test for
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			if !owns() {
				cancel(errCompactionJobNoLongerOwned)
				return
			}

			select {
			case <-ticker.C:
			case <-done:
				return
			case <-ownsCtx.Done():
				return
			}
		}
	}()

	err := rw.Compact(ownsCtx, blockMetas, tenantID)
	if errors.Is(err, context.Canceled) && errors.Is(context.Cause(ownsCtx), errCompactionJobNoLongerOwned) {
		level.Warn(rw.logger).Log("msg", "lost ownership of this job. abandoning job and trying again on this block list", "err", err)
		return nil
	}

	// test to see if we still own this job. it would be exceptional to log this message, but would be nice to know. a more likely bad case is that
	// job ownership changes but that change has not yet propagated to this compactor, so it duplicated data w/o realizing it.
	if !owns() {
		// format a string with all input metas
		sb := &strings.Builder{}
		for _, meta := range blockMetas {
			sb.WriteString(meta.BlockID.String())
			sb.WriteString(", ")
		}

		level.Error(rw.logger).Log("msg", "lost ownership of this job after compaction. possible data duplication", "tenant", tenantID, "input_blocks", sb.String())
	}

	return err
}

func (rw *readerWriter) Compact(ctx context.Context, blockMetas []*backend.BlockMeta, tenantID string) error {
	level.Debug(rw.logger).Log("msg", "beginning compaction", "num blocks compacting", len(blockMetas))

	// todo - add timeout?
	ctx, span := tracer.Start(ctx, "rw.compact")
	defer span.End()

	traceID, _ := tracing.ExtractTraceID(ctx)
	if traceID != "" {
		level.Info(rw.logger).Log("msg", "beginning compaction", "traceID", traceID)
	}

	if len(blockMetas) == 0 {
		return nil
	}

	var err error
	startTime := time.Now()

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
			"encoding", blockMeta.Encoding.String(),
			"totalRecords", blockMeta.TotalObjects,
			"bloomShardCount", blockMeta.BloomShardCount,
			"footerSize", blockMeta.FooterSize,
			"replicationFactor", blockMeta.ReplicationFactor,
		)
		totalRecords += int(blockMeta.TotalObjects)

		// Make sure block still exists
		_, err = rw.r.BlockMeta(ctx, (uuid.UUID)(blockMeta.BlockID), tenantID)
		if err != nil {
			return err
		}
	}

	enc, err := encoding.FromVersion(blockMetas[0].Version)
	if err != nil {
		return err
	}

	compactionLevel := compactionLevelForBlocks(blockMetas)
	compactionLevelLabel := strconv.Itoa(int(compactionLevel))

	combiner := instrumentedObjectCombiner{
		tenant:               tenantID,
		inner:                rw.compactorSharder,
		compactionLevelLabel: compactionLevelLabel,
	}

	opts := common.CompactionOptions{
		BlockConfig:        *rw.cfg.Block,
		ChunkSizeBytes:     rw.compactorCfg.ChunkSizeBytes,
		FlushSizeBytes:     rw.compactorCfg.FlushSizeBytes,
		IteratorBufferSize: rw.compactorCfg.IteratorBufferSize,
		OutputBlocks:       outputBlocks,
		Combiner:           combiner,
		MaxBytesPerTrace:   rw.compactorOverrides.MaxBytesPerTraceForTenant(tenantID),
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
			rw.compactorSharder.RecordDiscardedSpans(spans, tenantID, traceId, rootSpanName, rootServiceName)
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
		return err
	}

	// mark old blocks compacted, so they don't show up in polling
	if err := markCompacted(rw, tenantID, blockMetas, newCompactedBlocks); err != nil {
		return err
	}

	metricCompactionBlocks.WithLabelValues(compactionLevelLabel).Add(float64(len(blockMetas)))

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

	return nil
}

func markCompacted(rw *readerWriter, tenantID string, oldBlocks, newBlocks []*backend.BlockMeta) error {
	// Check if we have any errors, but continue marking the blocks as compacted
	var errCount int
	for _, meta := range oldBlocks {
		// Mark in the backend
		if err := rw.c.MarkBlockCompacted((uuid.UUID)(meta.BlockID), tenantID); err != nil {
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
	metricCompactionOutstandingBlocks.WithLabelValues(tenantID).Set(float64(totalOutstandingBlocks))
}

func compactionLevelForBlocks(blockMetas []*backend.BlockMeta) uint8 {
	level := uint8(0)

	for _, m := range blockMetas {
		if m.CompactionLevel > uint32(level) {
			level = uint8(m.CompactionLevel)
		}
	}

	return level
}

type instrumentedObjectCombiner struct {
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
}

// doForAtLeast executes the function f. It blocks for at least the passed duration but can go longer. if context is cancelled after
// the function is done we will bail immediately. in the current use case this means that the process is shutting down
// we don't force f() to cancel, we assume it also responds to the cancelled context
func doForAtLeast(ctx context.Context, dur time.Duration, f func()) {
	startTime := time.Now()
	f()
	elapsed := time.Since(startTime)

	if elapsed < dur {
		ticker := time.NewTicker(dur - elapsed)
		defer ticker.Stop()

		select {
		case <-ticker.C:
		case <-ctx.Done():
		}
	}
}
