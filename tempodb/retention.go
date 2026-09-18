package tempodb

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend"
	backend_cache "github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// retentionLoop watches a timer to clean up blocks that are past retention.
// todo: correctly pass context all the way to the backend so a cancelled context can stop the retention loop.
// see implementation of compactionLoop()
func (rw *readerWriter) retentionLoop(ctx context.Context) {
	ticker := time.NewTicker(rw.cfg.BlocklistPoll)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ticker.C:
			rw.doRetention(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (rw *readerWriter) doRetention(ctx context.Context) {
	rw.RetainWithConfig(ctx, rw.compactorCfg, rw.compactorSharder, rw.compactorOverrides)
}

func (rw *readerWriter) RetainWithConfig(ctx context.Context, compactorCfg *CompactorConfig, compactorSharder CompactorSharder, compactorOverrides CompactorOverrides) {
	tenants := rw.blocklist.Tenants()

	bg := boundedwaitgroup.New(compactorCfg.RetentionConcurrency)

	for _, tenantID := range tenants {
		if ctx.Err() != nil {
			break
		}

		if compactorOverrides.CompactionDisabledForTenant(tenantID) {
			continue
		}

		bg.Add(1)
		go func(t string) {
			defer bg.Done()

			rw.retainTenant(ctx, t, compactorCfg, compactorSharder, compactorOverrides)
		}(tenantID)
	}

	bg.Wait()
}

func (rw *readerWriter) RetainTenantWithConfig(ctx context.Context, tenantID string, compactorCfg *CompactorConfig, compactorSharder CompactorSharder, compactorOverrides CompactorOverrides) {
	rw.retainTenant(ctx, tenantID, compactorCfg, compactorSharder, compactorOverrides)
}

func (rw *readerWriter) retainTenant(ctx context.Context, tenantID string, compactorCfg *CompactorConfig, compactorSharder CompactorSharder, compactorOverrides CompactorOverrides) {
	start := time.Now()
	defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

	// Check for overrides
	retention := compactorCfg.BlockRetention // Default
	if r := compactorOverrides.BlockRetentionForTenant(tenantID); r != 0 {
		retention = r
	}
	level.Debug(rw.logger).Log("msg", "Performing block retention", "tenantID", tenantID, "retention", retention)

	// iterate through block list.  make compacted anything that is past retention.
	cutoff := time.Now().Add(-retention)
	blocklist := rw.blocklist.Metas(tenantID)
	for _, b := range blocklist {
		select {
		case <-ctx.Done():
			return
		default:
			if b.EndTime.Before(cutoff) && compactorSharder.Owns(b.BlockID.String()) {
				level.Info(rw.logger).Log("msg", "marking block for deletion", "blockID", b.BlockID, "tenantID", tenantID)
				err := rw.c.MarkBlockCompacted(uuid.UUID(b.BlockID), tenantID)
				if err != nil {
					level.Error(rw.logger).Log("msg", "failed to mark block compacted during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricMarkedForDeletion.Inc()

					rw.blocklist.Update(tenantID, nil, []*backend.BlockMeta{b}, []*backend.CompactedBlockMeta{
						{
							BlockMeta:     *b,
							CompactedTime: time.Now(),
						},
					}, nil)
				}
			}
		}
	}

	// iterate through compacted list looking for blocks ready to be cleared
	cutoff = time.Now().Add(-compactorCfg.CompactedBlockRetention)
	compactedBlocklist := rw.blocklist.CompactedMetas(tenantID)

	// Clearing a block is a list plus a delete batch against the backend, so the
	// loop is latency bound rather than CPU bound. Run a bounded number of blocks
	// at once: one retention job covers a whole tenant, and a serial loop cannot
	// keep up with compaction when a tenant has a large backlog.
	concurrency := compactorCfg.RetentionBlockConcurrency
	if concurrency == 0 {
		concurrency = DefaultRetentionBlockConcurrency
	}

	bg := boundedwaitgroup.New(concurrency)

	for _, b := range compactedBlocklist {
		if ctx.Err() != nil {
			break
		}

		if !b.CompactedTime.Before(cutoff) || !compactorSharder.Owns(b.BlockID.String()) {
			continue
		}

		bg.Add(1)
		go func(b *backend.CompactedBlockMeta) {
			defer bg.Done()

			level.Info(rw.logger).Log("msg", "deleting block", "blockID", b.BlockID, "tenantID", tenantID)
			err := rw.c.ClearBlock(uuid.UUID(b.BlockID), tenantID)
			if errors.Is(err, backend.ErrDoesNotExist) {
				level.Warn(rw.logger).Log("msg", "compacted block was already gone", "blockID", b.BlockID, "tenantID", tenantID)
				err = nil
			}
			if err != nil {
				level.Error(rw.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
				metricRetentionErrors.Inc()
				return
			}

			metricDeleted.Inc()
			rw.removeCachedBlock(ctx, tenantID, uuid.UUID(b.BlockID), int(b.BloomShardCount))
			rw.blocklist.Update(tenantID, nil, nil, nil, []*backend.CompactedBlockMeta{b})
		}(b)
	}

	bg.Wait()
}

func (rw *readerWriter) removeCachedBlock(ctx context.Context, tenantID string, blockID uuid.UUID, bloomShardCount int) {
	if rw.cacheProvider == nil {
		return
	}

	keyPrefix := backend_cache.BlockKeyPrefix(blockID, tenantID)

	if c := rw.cacheProvider.CacheFor(cache.RoleBloom); c != nil {
		keys := make([]string, 0, common.ValidateShardCount(bloomShardCount))
		for i := 0; i < common.ValidateShardCount(bloomShardCount); i++ {
			keys = append(keys, keyPrefix+common.BloomName(i))
		}
		c.Remove(ctx, keys)
	}

	if c := rw.cacheProvider.CacheFor(cache.RoleTraceIDIdx); c != nil {
		c.Remove(ctx, []string{keyPrefix + common.NameIndex})
	}
}
