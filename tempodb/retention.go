package tempodb

import (
	"time"

	"github.com/go-kit/kit/log/level"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
)

// todo: pass a context/chan in to cancel this cleanly
//  once a maintenance cycle cleanup any blocks
func (rw *readerWriter) retentionLoop() {
	ticker := time.NewTicker(rw.cfg.BlocklistPoll)
	for range ticker.C {
		rw.doRetention()
	}
}

func (rw *readerWriter) doRetention() {
	tenants := rw.blocklist.Tenants()

	bg := boundedwaitgroup.New(rw.compactorCfg.RetentionConcurrency)

	for _, tenantID := range tenants {
		bg.Add(1)
		go func(t string) {
			defer bg.Done()
			rw.retainTenant(t)
		}(tenantID.(string))
	}

	bg.Wait()
}

func (rw *readerWriter) retainTenant(tenantID string) {
	start := time.Now()
	defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

	// Check for overrides
	retention := rw.compactorCfg.BlockRetention // Default
	if r := rw.compactorOverrides.BlockRetentionForTenant(tenantID); r != 0 {
		retention = r
	}
	level.Debug(rw.logger).Log("msg", "Performing block retention", "tenantID", tenantID, "retention", retention)

	// iterate through block list.  make compacted anything that is past retention.
	cutoff := time.Now().Add(-retention)
	blocklist := rw.blocklist.Metas(tenantID)
	for _, b := range blocklist {
		if b.EndTime.Before(cutoff) && rw.compactorSharder.Owns(b.BlockID.String()) {
			level.Info(rw.logger).Log("msg", "marking block for deletion", "blockID", b.BlockID, "tenantID", tenantID)
			err := rw.c.MarkBlockCompacted(b.BlockID, tenantID)
			if err != nil {
				level.Error(rw.logger).Log("msg", "failed to mark block compacted during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
				metricRetentionErrors.Inc()
			} else {
				metricMarkedForDeletion.Inc()
			}
		}
	}

	// iterate through compacted list looking for blocks ready to be cleared
	cutoff = time.Now().Add(-rw.compactorCfg.CompactedBlockRetention)
	compactedBlocklist := rw.blocklist.CompactedMetas(tenantID)
	for _, b := range compactedBlocklist {
		if b.CompactedTime.Before(cutoff) && rw.compactorSharder.Owns(b.BlockID.String()) {
			level.Info(rw.logger).Log("msg", "deleting block", "blockID", b.BlockID, "tenantID", tenantID)
			err := rw.c.ClearBlock(b.BlockID, tenantID)
			if err != nil {
				level.Error(rw.logger).Log("msg", "failed to clear compacted block during retention", "blockID", b.BlockID, "tenantID", tenantID, "err", err)
				metricRetentionErrors.Inc()
			} else {
				metricDeleted.Inc()
			}
		}
	}
}
