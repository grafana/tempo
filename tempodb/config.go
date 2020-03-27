package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type Config struct {
	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	Cache   *cache.Config `yaml:"cache"`
	Pool    *pool.Config  `yaml:"query_pool,omitempty"`
	WAL     *wal.Config   `yaml:"wal"`

	MaintenanceCycle time.Duration `yaml:"maintenanceCycle"`
}

type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunkSizeBytes"`
	MaxCompactionRange      time.Duration `yaml:"maxCompactionRange"`
	MaxCompactionObjects    int           `yaml:"maxCompactionObjects"`
	BlockRetention          time.Duration `yaml:"blockRetention"`
	CompactedBlockRetention time.Duration `yaml:"compactedBlockRetention"`
}
