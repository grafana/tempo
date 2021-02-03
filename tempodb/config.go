package tempodb

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

const DefaultBlocklistPollConcurrency = uint(50)
const DefaultRetentionConcurrency = uint(10)

type Config struct {
	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Azure   *azure.Config `yaml:"azure"`
	Pool    *pool.Config  `yaml:"pool,omitempty"`
	WAL     *wal.Config   `yaml:"wal"`

	Cache     string            `yaml:"cache"`
	Memcached *memcached.Config `yaml:"memcached"`
	Redis     *redis.Config     `yaml:"redis"`

	BlocklistPoll            time.Duration `yaml:"blocklist_poll"`
	BlocklistPollConcurrency uint          `yaml:"blocklist_poll_concurrency"`
}

type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunk_size_bytes"` // todo: do we need this?
	FlushSizeBytes          uint32        `yaml:"flush_size_bytes"`
	MaxCompactionRange      time.Duration `yaml:"compaction_window"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects"`
	BlockRetention          time.Duration `yaml:"block_retention"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention"`
	RetentionConcurrency    uint          `yaml:"retention_concurrency"`
}
