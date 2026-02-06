package common

import (
	"flag"
	"fmt"

	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/gzip"
	"github.com/parquet-go/parquet-go/compress/lz4"
	"github.com/parquet-go/parquet-go/compress/snappy"
	"github.com/parquet-go/parquet-go/compress/uncompressed"
	"github.com/parquet-go/parquet-go/compress/zstd"
)

const (
	DefaultBloomFP              = .01
	DefaultBloomShardSizeBytes  = 100 * 1024
	DefaultIndexDownSampleBytes = 1024 * 1024
	DefaultIndexPageSizeBytes   = 250 * 1024
)

// ParquetCompression defines the compression algorithm for parquet columns.
type ParquetCompression string

const (
	ParquetCompressionSnappy ParquetCompression = "snappy"
	ParquetCompressionLZ4    ParquetCompression = "lz4_raw"
	ParquetCompressionZstd   ParquetCompression = "zstd"
	ParquetCompressionGzip   ParquetCompression = "gzip"
	ParquetCompressionNone   ParquetCompression = "none"
)

const DeprecatedError = "%s is no longer supported, please use %s or later"

// BlockConfig holds configuration options for newly created blocks
type BlockConfig struct {
	BloomFP             float64 `yaml:"bloom_filter_false_positive"`
	BloomShardSizeBytes int     `yaml:"bloom_filter_shard_size_bytes"`
	Version             string  `yaml:"version"`

	// parquet fields
	RowGroupSizeBytes  int                `yaml:"parquet_row_group_size_bytes"`
	ParquetCompression ParquetCompression `yaml:"parquet_compression"`

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
	cfg.ParquetCompression = ParquetCompressionSnappy
}

// GetCompressionOption returns a parquet.WriterOption that sets the compression codec
// for all columns in the parquet file.
func (cfg *BlockConfig) GetCompressionOption() parquet.WriterOption {
	switch cfg.ParquetCompression {
	case ParquetCompressionLZ4:
		return parquet.Compression(&lz4.Codec{})
	case ParquetCompressionZstd:
		return parquet.Compression(&zstd.Codec{})
	case ParquetCompressionGzip:
		return parquet.Compression(&gzip.Codec{})
	case ParquetCompressionNone:
		return parquet.Compression(&uncompressed.Codec{})
	default: // snappy or empty
		return parquet.Compression(&snappy.Codec{})
	}
}

// ValidateConfig returns true if the config is valid
func ValidateConfig(b *BlockConfig) error {
	if b.BloomFP <= 0.0 || b.BloomFP >= 1.0 {
		return fmt.Errorf("invalid bloom filter fp rate %v", b.BloomFP)
	}

	if b.BloomShardSizeBytes <= 0 {
		return fmt.Errorf("positive value required for bloom-filter shard size")
	}

	// Validate parquet compression
	switch b.ParquetCompression {
	case ParquetCompressionSnappy, ParquetCompressionLZ4, ParquetCompressionZstd,
		ParquetCompressionGzip, ParquetCompressionNone, "":
		// valid
	default:
		return fmt.Errorf("invalid parquet_compression: %s", b.ParquetCompression)
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
