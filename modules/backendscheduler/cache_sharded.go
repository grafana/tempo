package backendscheduler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/tempo/modules/backendscheduler/work"
)

// checkForShardFiles checks if any shard files exist
func (s *BackendScheduler) checkForShardFiles() bool {
	for i := range work.ShardCount {
		if _, err := os.Stat(s.filenameForShard(uint8(i))); err == nil {
			return true // Found at least one shard file
		}
	}
	return false
}

func (s *BackendScheduler) filenameForShard(shardID uint8) string {
	return filepath.Join(s.cfg.LocalWorkPath, fmt.Sprintf("shard_%03d.json", shardID))
}
