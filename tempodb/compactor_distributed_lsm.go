package tempodb

import (
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	// Number of blocks at a level L = blockNumberMultiplier^L
	// blockNumberMultiplier = 10

	// Number of levels in the levelled compaction strategy
	maxNumLevels = 2
)

// For now it just adds blocks prefixed by tenantID to redis
func (rw *readerWriter) doLevelledCompaction() {
	// stop any existing compaction jobs
	// TODO(@annanay25): ideally would want to wait for existing jobs to finish
	if rw.jobStopper != nil {
		start := time.Now()
		err := rw.jobStopper.Stop()
		if err != nil {
			level.Warn(rw.logger).Log("msg", "error during compaction cycle", "err", err)
			metricCompactionErrors.Inc()
		}
		metricCompactionStopDuration.Observe(time.Since(start).Seconds())
	}

	// start crazy jobs to do compaction with new list
	tenants := rw.blocklistTenants()

	var err error
	rw.jobStopper, err = rw.pool.RunStoppableJobs(tenants, func(payload interface{}, stopCh <-chan struct{}) error {
		tenantID := payload.(string)
		blocklist := rw.blocklist(tenantID)
		blocksPerLevel := make([][]*backend.BlockMeta, maxNumLevels)
		for k := 0; k < maxNumLevels; k++ {
			blocksPerLevel[k] = make([]*backend.BlockMeta, 0)
		}

		for _, block := range blocklist {
			blocksPerLevel[block.CompactionLevel] = append(blocksPerLevel[block.CompactionLevel], block)
		}

		rw.blockSelector.ResetCursor()

		// Right now run this loop only for level 0
		for l := 0; l < maxNumLevels-1; l++ {
			// We don't compact anything at the top level
			for {
				// Pop inputBlocks elements from the current queue
				pos := rw.blockSelector.BlocksToCompactInSameLevel(blocksPerLevel[l])
				if pos == -1 {
					// If none are suitable, bail
					break
				}
				toBeCompacted := blocksPerLevel[l][pos : pos+inputBlocks-1]
				rw.compact(toBeCompacted, tenantID)
			}
		}

		return nil
	})

	if err != nil {
		level.Error(rw.logger).Log("msg", "failed to start compaction.  compaction broken until next maintenance cycle.", "err", err)
		metricCompactionErrors.Inc()
	}

}
