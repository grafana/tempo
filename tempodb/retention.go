package tempodb

import (
	"context"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
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
				err := rw.c.MarkBlockCompacted((uuid.UUID)(b.BlockID), tenantID)
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
	for _, b := range compactedBlocklist {
		select {
		case <-ctx.Done():
			return
		default:
			level.Debug(rw.logger).Log("owns", compactorSharder.Owns(b.BlockID.String()), "blockID", b.BlockID, "tenantID", tenantID)
			if b.CompactedTime.Before(cutoff) && compactorSharder.Owns(b.BlockID.String()) {
				level.Info(rw.logger).Log("msg", "deleting block", "blockID", b.BlockID, "tenantID", tenantID)
				err := rw.c.ClearBlock((uuid.UUID)(b.BlockID), tenantID)
				if err != nil {
					level.Error(rw.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricDeleted.Inc()

					rw.blocklist.Update(tenantID, nil, nil, nil, []*backend.CompactedBlockMeta{b})
				}
			}
		}
	}
}
