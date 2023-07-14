package tempodb

import (
	"context"
	"time"

	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/tempodb/backend"
)

// retentionLoop watches a timer to clean up blocks that are past retention.
func (db *tempoDB) retentionLoop(ctx context.Context) {
	ticker := time.NewTicker(db.cfg.BlocklistPoll)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ticker.C:
			db.doRetention(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (db *tempoDB) doRetention(ctx context.Context) {
	tenants := db.blocklist.Tenants()

	bg := boundedwaitgroup.New(db.compactorCfg.RetentionConcurrency)

	for _, tenantID := range tenants {
		bg.Add(1)
		go func(t string) {
			defer bg.Done()
			db.retainTenant(ctx, t)
		}(tenantID)
	}

	bg.Wait()
}

func (db *tempoDB) retainTenant(ctx context.Context, tenantID string) {
	start := time.Now()
	defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

	// Check for overrides
	retention := db.compactorCfg.BlockRetention // Default
	if r := db.compactorOverrides.BlockRetentionForTenant(tenantID); r != 0 {
		retention = r
	}
	level.Debug(db.logger).Log("msg", "Performing block retention", "tenantID", tenantID, "retention", retention)

	// iterate through block list.  make compacted anything that is past retention.
	cutoff := time.Now().Add(-retention)
	blocklist := db.blocklist.Metas(tenantID)
	for _, b := range blocklist {
		select {
		case <-ctx.Done():
			return
		default:
			if b.EndTime.Before(cutoff) && db.compactorSharder.Owns(b.BlockID.String()) {
				level.Info(db.logger).Log("msg", "marking block for deletion", "blockID", b.BlockID, "tenantID", tenantID)
				err := db.c.MarkBlockCompacted(b.BlockID, tenantID)
				if err != nil {
					level.Error(db.logger).Log("msg", "failed to mark block compacted during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricMarkedForDeletion.Inc()

					db.blocklist.Update(tenantID, nil, []*backend.BlockMeta{b}, []*backend.CompactedBlockMeta{
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
	cutoff = time.Now().Add(-db.compactorCfg.CompactedBlockRetention)
	compactedBlocklist := db.blocklist.CompactedMetas(tenantID)
	for _, b := range compactedBlocklist {
		select {
		case <-ctx.Done():
			return
		default:
			level.Debug(db.logger).Log("owns", db.compactorSharder.Owns(b.BlockID.String()), "blockID", b.BlockID, "tenantID", tenantID)
			if b.CompactedTime.Before(cutoff) && db.compactorSharder.Owns(b.BlockID.String()) {
				level.Info(db.logger).Log("msg", "deleting block", "blockID", b.BlockID, "tenantID", tenantID)
				err := db.c.ClearBlock(b.BlockID, tenantID)
				if err != nil {
					level.Error(db.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
					metricRetentionErrors.Inc()
				} else {
					metricDeleted.Inc()

					db.blocklist.Update(tenantID, nil, nil, nil, []*backend.CompactedBlockMeta{b})
				}
			}
		}
	}
}
