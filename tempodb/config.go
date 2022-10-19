package tempodb

import (
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	DefaultBlocklistPoll            = 5 * time.Minute
	DefaultMaxTimePerTenant         = 5 * time.Minute
	DefaultBlocklistPollConcurrency = uint(50)
	DefaultRetentionConcurrency     = uint(10)
	DefaultTenantIndexBuilders      = 2

	DefaultPrefetchTraceCount   = 1000
	DefaultSearchChunkSizeBytes = 1_000_000
	DefaultReadBufferCount      = 8
	DefaultReadBufferSize       = 4 * 1024 * 1024
)

// Config holds the entirety of tempodb configuration
// Defaults are in modules/storage/config.go
type Config struct {
	Pool   *pool.Config        `yaml:"pool,omitempty"`
	WAL    *wal.Config         `yaml:"wal,omitempty"`
	Block  *common.BlockConfig `yaml:"block,omitempty"`
	Search SearchConfig        `yaml:"search,omitempty"`

	BlocklistPoll                    time.Duration `yaml:"blocklist_poll,omitempty"`
	BlocklistPollConcurrency         uint          `yaml:"blocklist_poll_concurrency,omitempty"`
	BlocklistPollFallback            bool          `yaml:"blocklist_poll_fallback,omitempty"`
	BlocklistPollTenantIndexBuilders int           `yaml:"blocklist_poll_tenant_index_builders,omitempty"`
	BlocklistPollStaleTenantIndex    time.Duration `yaml:"blocklist_poll_stale_tenant_index,omitempty"`
	BlocklistPollJitterMs            int           `yaml:"blocklist_poll_jitter_ms,omitempty"`

	// backends
	Backend string        `yaml:"backend,omitempty"`
	Local   *local.Config `yaml:"local,omitempty"`
	GCS     *gcs.Config   `yaml:"gcs,omitempty"`
	S3      *s3.Config    `yaml:"s3,omitempty"`
	Azure   *azure.Config `yaml:"azure,omitempty"`

	// caches
	Cache                   string                  `yaml:"cache,omitempty"`
	CacheMinCompactionLevel uint8                   `yaml:"cache_min_compaction_level,omitempty"`
	CacheMaxBlockAge        time.Duration           `yaml:"cache_max_block_age,omitempty"`
	BackgroundCache         *cache.BackgroundConfig `yaml:"background_cache,omitempty"`
	Memcached               *memcached.Config       `yaml:"memcached,omitempty"`
	Redis                   *redis.Config           `yaml:"redis,omitempty"`
}

type SearchConfig struct {
	// v2 blocks
	ChunkSizeBytes     uint32 `yaml:"chunk_size_bytes,omitempty"`
	PrefetchTraceCount int    `yaml:"prefetch_trace_count,omitempty"`

	// vParquet blocks
	ReadBufferCount     int `yaml:"read_buffer_count,omitempty"`
	ReadBufferSizeBytes int `yaml:"read_buffer_size_bytes,omitempty"`
	CacheControl        struct {
		Footer      bool `yaml:"footer,omitempty"`
		ColumnIndex bool `yaml:"column_index,omitempty"`
		OffsetIndex bool `yaml:"offset_index,omitempty"`
	} `yaml:"cache_control"`
}

func (c SearchConfig) ApplyToOptions(o *common.SearchOptions) {
	o.ChunkSizeBytes = c.ChunkSizeBytes
	o.PrefetchTraceCount = c.PrefetchTraceCount
	o.ReadBufferCount = c.ReadBufferCount
	o.ReadBufferSize = c.ReadBufferSizeBytes

	if o.ChunkSizeBytes == 0 {
		o.ChunkSizeBytes = DefaultSearchChunkSizeBytes
	}
	if o.PrefetchTraceCount <= 0 {
		o.PrefetchTraceCount = DefaultPrefetchTraceCount
	}
	if o.ReadBufferSize <= 0 {
		o.ReadBufferSize = DefaultReadBufferSize
	}
	if o.ReadBufferCount <= 0 {
		o.ReadBufferCount = DefaultReadBufferCount
	}

	o.CacheControl.Footer = c.CacheControl.Footer
	o.CacheControl.ColumnIndex = c.CacheControl.ColumnIndex
	o.CacheControl.OffsetIndex = c.CacheControl.OffsetIndex
}

// CompactorConfig contains compaction configuration options
type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunk_size_bytes,omitempty"`
	FlushSizeBytes          uint32        `yaml:"flush_size_bytes,omitempty"`
	MaxCompactionRange      time.Duration `yaml:"compaction_window,omitempty"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects,omitempty"`
	MaxBlockBytes           uint64        `yaml:"max_block_bytes,omitempty"`
	BlockRetention          time.Duration `yaml:"block_retention,omitempty"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention,omitempty"`
	RetentionConcurrency    uint          `yaml:"retention_concurrency,omitempty"`
	IteratorBufferSize      int           `yaml:"iterator_buffer_size,omitempty"`
	MaxTimePerTenant        time.Duration `yaml:"max_time_per_tenant,omitempty"`
	CompactionCycle         time.Duration `yaml:"compaction_cycle,omitempty"`
}

func validateConfig(cfg *Config) error {
	err := common.ValidateConfig(cfg.Block)
	if err != nil {
		return fmt.Errorf("block config validation failed: %w", err)
	}

	_, err = encoding.FromVersion(cfg.Block.Version)
	if err != nil {
		return fmt.Errorf("block version validation failed: %w", err)
	}

	return nil
}
