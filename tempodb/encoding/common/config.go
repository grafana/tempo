package common

import (
	"flag"
	"fmt"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	DefaultBloomFP              = .01
	DefaultBloomShardSizeBytes  = 100 * 1024
	DefaultIndexDownSampleBytes = 1024 * 1024
	DefaultIndexPageSizeBytes   = 250 * 1024
)

// BlockConfig holds configuration options for newly created blocks
type BlockConfig struct {
	BloomFP             float64          `yaml:"bloom_filter_false_positive"`
	BloomShardSizeBytes int              `yaml:"bloom_filter_shard_size_bytes"`
	Version             string           `yaml:"version"`
	SearchEncoding      backend.Encoding `yaml:"search_encoding"`
	SearchPageSizeBytes int              `yaml:"search_page_size_bytes"`

	// v2 fields
	IndexDownsampleBytes int              `yaml:"v2_index_downsample_bytes"`
	IndexPageSizeBytes   int              `yaml:"v2_index_page_size_bytes"`
	Encoding             backend.Encoding `yaml:"v2_encoding"`

	// parquet fields
	RowGroupSizeBytes int `yaml:"parquet_row_group_size_bytes"`

	// vParquet3 fields
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns"`
}

func (cfg *BlockConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.Float64Var(&cfg.BloomFP, util.PrefixConfig(prefix, "trace.block.v2-bloom-filter-false-positive"), DefaultBloomFP, "Bloom Filter False Positive.")
	f.IntVar(&cfg.BloomShardSizeBytes, util.PrefixConfig(prefix, "trace.block.v2-bloom-filter-shard-size-bytes"), DefaultBloomShardSizeBytes, "Bloom Filter Shard Size in bytes.")
	f.IntVar(&cfg.IndexDownsampleBytes, util.PrefixConfig(prefix, "trace.block.v2-index-downsample-bytes"), DefaultIndexDownSampleBytes, "Number of bytes (before compression) per index record.")
	f.IntVar(&cfg.IndexPageSizeBytes, util.PrefixConfig(prefix, "trace.block.v2-index-page-size-bytes"), DefaultIndexPageSizeBytes, "Number of bytes per index page.")
	// cfg.Version = encoding.DefaultEncoding().Version() // Cyclic dependency - ugh
	cfg.Encoding = backend.EncZstd
	cfg.SearchEncoding = backend.EncSnappy
	cfg.SearchPageSizeBytes = 1024 * 1024 // 1 MB
	cfg.RowGroupSizeBytes = 100_000_000   // 100 MB
}

// ValidateConfig returns true if the config is valid
func ValidateConfig(b *BlockConfig) error {
	if b.IndexDownsampleBytes <= 0 {
		return fmt.Errorf("positive index downsample required")
	}

	if b.IndexPageSizeBytes <= 0 {
		return fmt.Errorf("positive index page size required")
	}

	if b.BloomFP <= 0.0 {
		return fmt.Errorf("invalid bloom filter fp rate %v", b.BloomFP)
	}

	if b.BloomShardSizeBytes <= 0 {
		return fmt.Errorf("positive value required for bloom-filter shard size")
	}

	if b.Version == "vParquet" {
		return fmt.Errorf("this version of vParquet has been deprecated, please use vParquet2 or higher")
	}

	return b.DedicatedColumns.Validate()
}
