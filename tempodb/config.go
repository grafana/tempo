package tempodb

import (
	"errors"
	"fmt"
	"time"

	cortex_cache "github.com/cortexproject/cortex/pkg/chunk/cache"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	DefaultBlocklistPoll            = 5 * time.Minute
	DefaultBlocklistPollConcurrency = uint(50)
	DefaultRetentionConcurrency     = uint(10)
	DefaultTenantIndexBuilders      = 2
)

// Config holds the entirety of tempodb configuration
// Defaults are in modules/storage/config.go
type Config struct {
	Pool  *pool.Config          `yaml:"pool,omitempty"`
	WAL   *wal.Config           `yaml:"wal"`
	Block *encoding.BlockConfig `yaml:"block"`

	BlocklistPoll                    time.Duration `yaml:"blocklist_poll"`
	BlocklistPollConcurrency         uint          `yaml:"blocklist_poll_concurrency"`
	BlocklistPollFallback            bool          `yaml:"blocklist_poll_fallback"`
	BlocklistPollTenantIndexBuilders int           `yaml:"blocklist_poll_tenant_index_builders"`

	// backends
	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Azure   *azure.Config `yaml:"azure"`

	// caches
	Cache                   string                         `yaml:"cache"`
	CacheMinCompactionLevel uint8                          `yaml:"cache_min_compaction_level"`
	CacheMaxBlockAge        time.Duration                  `yaml:"cache_max_block_age"`
	BackgroundCache         *cortex_cache.BackgroundConfig `yaml:"background_cache"`
	Memcached               *memcached.Config              `yaml:"memcached"`
	Redis                   *redis.Config                  `yaml:"redis"`
}

// CompactorConfig contains compaction configuration options
type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunk_size_bytes"` // todo: do we need this?
	FlushSizeBytes          uint32        `yaml:"flush_size_bytes"`
	MaxCompactionRange      time.Duration `yaml:"compaction_window"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects"`
	MaxBlockBytes           uint64        `yaml:"max_block_bytes"`
	BlockRetention          time.Duration `yaml:"block_retention"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention"`
	RetentionConcurrency    uint          `yaml:"retention_concurrency"`
	IteratorBufferSize      int           `yaml:"iterator_buffer_size"`
}

func validateConfig(cfg *Config) error {
	if cfg.WAL == nil {
		return errors.New("wal config should be non-nil")
	}

	if cfg.Block == nil {
		return errors.New("block config should be non-nil")
	}

	err := encoding.ValidateConfig(cfg.Block)
	if err != nil {
		return fmt.Errorf("block config validation failed: %w", err)
	}

	return nil
}
