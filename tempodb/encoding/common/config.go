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

const DeprecatedError = "%s is no longer supported, please use %s or later"

// BlockConfig holds configuration options for newly created blocks
type BlockConfig struct {
	BloomFP             float64 `yaml:"bloom_filter_false_positive"`
	BloomShardSizeBytes int     `yaml:"bloom_filter_shard_size_bytes"`
	Version             string  `yaml:"version"`

	// parquet fields
	RowGroupSizeBytes int `yaml:"parquet_row_group_size_bytes"`

	// vParquet3 fields
	DedicatedColumns backend.DedicatedColumns `yaml:"parquet_dedicated_columns"`

	// used internally. If true, the block will be created by default with the nocompact flag set.
	CreateWithNoCompactFlag bool `yaml:"-"`
}

func (cfg *BlockConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.Float64Var(&cfg.BloomFP, util.PrefixConfig(prefix, "trace.block.v2-bloom-filter-false-positive"), DefaultBloomFP, "Bloom Filter False Positive.")
	f.IntVar(&cfg.BloomShardSizeBytes, util.PrefixConfig(prefix, "trace.block.v2-bloom-filter-shard-size-bytes"), DefaultBloomShardSizeBytes, "Bloom Filter Shard Size in bytes.")

	cfg.RowGroupSizeBytes = 100_000_000 // 100 MB
	cfg.DedicatedColumns = backend.DefaultDedicatedColumns()
}

// ValidateConfig returns true if the config is valid
func ValidateConfig(b *BlockConfig) error {
	if b.BloomFP <= 0.0 || b.BloomFP >= 1.0 {
		return fmt.Errorf("invalid bloom filter fp rate %v", b.BloomFP)
	}

	if b.BloomShardSizeBytes <= 0 {
		return fmt.Errorf("positive value required for bloom-filter shard size")
	}

	// Check for deprecated version,
	// TODO - Cyclic dependency makes this awkward to improve by using the
	// deprecation information in the encoding itself, in the versioned logic
	// in the parent folder. So we are checking raw strings here.
	/*if b.Version == "vParquet2" {
		return fmt.Errorf(DeprecatedError, "vParquet2", "vParquet3")
	}*/

	// TODO - log or pass warnings up the chain?
	_, err := b.DedicatedColumns.Validate()
	return err
}
