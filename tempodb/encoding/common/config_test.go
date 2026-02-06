package common

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParquetCompressionValidation(t *testing.T) {
	tests := []struct {
		name        string
		compression ParquetCompression
		wantErr     bool
	}{
		{
			name:        "snappy is valid",
			compression: ParquetCompressionSnappy,
			wantErr:     false,
		},
		{
			name:        "lz4_raw is valid",
			compression: ParquetCompressionLZ4,
			wantErr:     false,
		},
		{
			name:        "zstd is valid",
			compression: ParquetCompressionZstd,
			wantErr:     false,
		},
		{
			name:        "gzip is valid",
			compression: ParquetCompressionGzip,
			wantErr:     false,
		},
		{
			name:        "none is valid",
			compression: ParquetCompressionNone,
			wantErr:     false,
		},
		{
			name:        "empty string is valid (defaults to snappy)",
			compression: "",
			wantErr:     false,
		},
		{
			name:        "invalid compression",
			compression: "invalid",
			wantErr:     true,
		},
		{
			name:        "lz4 without _raw is invalid",
			compression: "lz4",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &BlockConfig{
				BloomFP:             0.01,
				BloomShardSizeBytes: 100 * 1024,
				ParquetCompression:  tc.compression,
			}
			err := ValidateConfig(cfg)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid parquet_compression")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetCompressionOption(t *testing.T) {
	tests := []struct {
		name        string
		compression ParquetCompression
	}{
		{"snappy", ParquetCompressionSnappy},
		{"lz4_raw", ParquetCompressionLZ4},
		{"zstd", ParquetCompressionZstd},
		{"gzip", ParquetCompressionGzip},
		{"none", ParquetCompressionNone},
		{"empty defaults to snappy", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &BlockConfig{
				ParquetCompression: tc.compression,
			}
			opt := cfg.GetCompressionOption()
			require.NotNil(t, opt)
		})
	}
}

func TestBlockConfigDefaultCompression(t *testing.T) {
	cfg := &BlockConfig{}
	f := &flag.FlagSet{}
	cfg.RegisterFlagsAndApplyDefaults("test", f)

	assert.Equal(t, ParquetCompressionSnappy, cfg.ParquetCompression)
}
