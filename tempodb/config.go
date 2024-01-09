package tempodb

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/grafana/tempo/modules/cache/memcached"
	"github.com/grafana/tempo/modules/cache/redis"

	"github.com/grafana/tempo/pkg/cache"
	azure "github.com/grafana/tempo/tempodb/backend/azure/config"
	backend_cache "github.com/grafana/tempo/tempodb/backend/cache"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	DefaultBlocklistPoll             = 5 * time.Minute
	DefaultMaxTimePerTenant          = 5 * time.Minute
	DefaultBlocklistPollConcurrency  = uint(50)
	DefaultRetentionConcurrency      = uint(10)
	DefaultTenantIndexBuilders       = 2
	DefaultTolerateConsecutiveErrors = 1

	DefaultPrefetchTraceCount   = 1000
	DefaultSearchChunkSizeBytes = 1_000_000
	DefaultReadBufferCount      = 32
	DefaultReadBufferSize       = 1 * 1024 * 1024
)

// Config holds the entirety of tempodb configuration
// Defaults are in modules/storage/config.go
type Config struct {
	Pool   *pool.Config        `yaml:"pool,omitempty"`
	WAL    *wal.Config         `yaml:"wal"`
	Block  *common.BlockConfig `yaml:"block"`
	Search *SearchConfig       `yaml:"search"`

	BlocklistPoll                          time.Duration `yaml:"blocklist_poll"`
	BlocklistPollConcurrency               uint          `yaml:"blocklist_poll_concurrency"`
	BlocklistPollFallback                  bool          `yaml:"blocklist_poll_fallback"`
	BlocklistPollTenantIndexBuilders       int           `yaml:"blocklist_poll_tenant_index_builders"`
	BlocklistPollStaleTenantIndex          time.Duration `yaml:"blocklist_poll_stale_tenant_index"`
	BlocklistPollJitterMs                  int           `yaml:"blocklist_poll_jitter_ms"`
	BlocklistPollTolerateConsecutiveErrors int           `yaml:"blocklist_poll_tolerate_consecutive_errors"`

	// backends
	Backend string        `yaml:"backend"`
	Local   *local.Config `yaml:"local"`
	GCS     *gcs.Config   `yaml:"gcs"`
	S3      *s3.Config    `yaml:"s3"`
	Azure   *azure.Config `yaml:"azure"`

	// legacy cache config. this is loaded by tempodb and added to the cache
	// provider on construction
	Cache           string                  `yaml:"cache"`
	BackgroundCache *cache.BackgroundConfig `yaml:"background_cache"`
	Memcached       *memcached.Config       `yaml:"memcached"`
	Redis           *redis.Config           `yaml:"redis"`

	BloomCacheCfg backend_cache.BloomConfig `yaml:",inline"`
}

type CacheControlConfig struct {
	Footer      bool `yaml:"footer"`
	ColumnIndex bool `yaml:"column_index"`
	OffsetIndex bool `yaml:"offset_index"`
}

type SearchConfig struct {
	// v2 blocks
	ChunkSizeBytes     uint32 `yaml:"chunk_size_bytes"`
	PrefetchTraceCount int    `yaml:"prefetch_trace_count"`

	// vParquet blocks
	ReadBufferCount     int `yaml:"read_buffer_count"`
	ReadBufferSizeBytes int `yaml:"read_buffer_size_bytes"`
	// todo: consolidate caching conffig in one spot
	CacheControl CacheControlConfig `yaml:"cache_control"`
}

func (c *SearchConfig) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	c.ChunkSizeBytes = DefaultSearchChunkSizeBytes
	c.PrefetchTraceCount = DefaultPrefetchTraceCount
	c.ReadBufferCount = DefaultReadBufferCount
	c.ReadBufferSizeBytes = DefaultReadBufferSize
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
}

// CompactorConfig contains compaction configuration options
type CompactorConfig struct {
	ChunkSizeBytes          uint32        `yaml:"v2_in_buffer_bytes"`
	FlushSizeBytes          uint32        `yaml:"v2_out_buffer_bytes"`
	IteratorBufferSize      int           `yaml:"v2_prefetch_traces_count"`
	MaxCompactionRange      time.Duration `yaml:"compaction_window"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects"`
	MaxBlockBytes           uint64        `yaml:"max_block_bytes"`
	BlockRetention          time.Duration `yaml:"block_retention"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention"`
	RetentionConcurrency    uint          `yaml:"retention_concurrency"`
	MaxTimePerTenant        time.Duration `yaml:"max_time_per_tenant"`
	CompactionCycle         time.Duration `yaml:"compaction_cycle"`
}

func (compactorConfig CompactorConfig) validate() error {
	if compactorConfig.MaxCompactionRange == 0 {
		return errors.New("Compaction window can't be 0")
	}

	return nil
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config should be non-nil")
	}

	if cfg.WAL == nil {
		return errors.New("wal config should be non-nil")
	}

	if cfg.Block == nil {
		return errors.New("block config should be non-nil")
	}

	// if the wal version is unspecified default to the block version
	if cfg.WAL.Version == "" {
		cfg.WAL.Version = cfg.Block.Version
	}

	err := wal.ValidateConfig(cfg.WAL)
	if err != nil {
		return fmt.Errorf("wal config validation failed: %w", err)
	}

	err = common.ValidateConfig(cfg.Block)
	if err != nil {
		return fmt.Errorf("block config validation failed: %w", err)
	}

	_, err = encoding.FromVersion(cfg.Block.Version)
	if err != nil {
		return fmt.Errorf("block version validation failed: %w", err)
	}

	return nil
}
