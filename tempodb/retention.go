package tempodb

import (
	"context"
	"time"

	"github.com/go-kit/kit/log/level"
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
	tenants := rw.blocklistTenants()

	// todo: continued abuse of runJobs.  need a runAllJobs() method or something
	_, err := rw.pool.RunJobs(context.TODO(), tenants, func(_ context.Context, payload interface{}) ([]byte, error) {

		start := time.Now()
		defer func() { metricRetentionDuration.Observe(time.Since(start).Seconds()) }()

		tenantID := payload.(string)

		// iterate through block list.  make compacted anything that is past retention.
		cutoff := time.Now().Add(-rw.compactorCfg.BlockRetention)
		blocklist := rw.blocklist(tenantID)
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
		compactedBlocklist := rw.compactedBlocklist(tenantID)
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

		return nil, nil
	})

	if err != nil {
		level.Error(rw.logger).Log("msg", "failure to start retention.  retention disabled until the next maintenance cycle", "err", err)
		metricRetentionErrors.Inc()
	}
}
