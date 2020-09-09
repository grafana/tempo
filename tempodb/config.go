package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

type Config struct {
	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Cache   *cache.Config `yaml:"cache"`
	Pool    *pool.Config  `yaml:"pool,omitempty"`
	WAL     *wal.Config   `yaml:"wal"`

	MaintenanceCycle time.Duration `yaml:"maintenanceCycle"`
}

type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunk_size_bytes"` // todo: do we need this?
	MaxCompactionRange      time.Duration `yaml:"compaction_window"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects"`
	BlockRetention          time.Duration `yaml:"block_retention"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention"`
}
