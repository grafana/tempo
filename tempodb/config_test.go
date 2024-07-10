package tempodb

import (
	"errors"
	"testing"

	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyToOptions(t *testing.T) {
	opts := common.DefaultSearchOptions()
	cfg := SearchConfig{}

	// test defaults
	cfg.ApplyToOptions(&opts)
	require.Equal(t, opts.PrefetchTraceCount, DefaultPrefetchTraceCount)
	require.Equal(t, opts.ChunkSizeBytes, uint32(DefaultSearchChunkSizeBytes))
	require.Equal(t, opts.ReadBufferCount, DefaultReadBufferCount)
	require.Equal(t, opts.ReadBufferSize, DefaultReadBufferSize)

	// test parameter fields are left alone
	opts.StartPage = 1
	opts.TotalPages = 2
	opts.MaxBytes = 3
	cfg.ApplyToOptions(&opts)
	require.Equal(t, opts.StartPage, 1)
	require.Equal(t, opts.TotalPages, 2)
	require.Equal(t, opts.MaxBytes, 3)

	// test non defaults
	cfg.ChunkSizeBytes = 4
	cfg.PrefetchTraceCount = 5
	cfg.ReadBufferCount = 6
	cfg.ReadBufferSizeBytes = 7
	cfg.ApplyToOptions(&opts)
	require.Equal(t, cfg.ChunkSizeBytes, uint32(4))
	require.Equal(t, cfg.PrefetchTraceCount, 5)
	require.Equal(t, cfg.ReadBufferCount, 6)
	require.Equal(t, cfg.ReadBufferSizeBytes, 7)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		cfg            *Config
		expectedConfig *Config
		err            error
	}{
		// nil config fails
		{
			err: errors.New("config should be non-nil"),
		},
		// nil wal fails
		{
			cfg: &Config{},
			err: errors.New("wal config should be non-nil"),
		},
		// nil block fails
		{
			cfg: &Config{
				WAL: &wal.Config{},
			},
			err: errors.New("block config should be non-nil"),
		},
		// block version copied to wal if empty
		{
			cfg: &Config{
				WAL: &wal.Config{},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "v2",
				},
			},
			expectedConfig: &Config{
				WAL: &wal.Config{
					Version: "v2",
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "v2",
				},
			},
		},
		// block version not copied to wal if populated
		{
			cfg: &Config{
				WAL: &wal.Config{
					Version: "vParquet2",
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "v2",
				},
			},
			expectedConfig: &Config{
				WAL: &wal.Config{
					Version: "vParquet2",
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "v2",
				},
			},
		},
	}

	for _, test := range tests {
		err := validateConfig(test.cfg)
		require.Equal(t, test.err, err)

		if test.expectedConfig != nil {
			require.Equal(t, test.expectedConfig, test.cfg)
		}
	}
}

func TestDeprecatedVersions(t *testing.T) {
	tests := []struct {
		cfg            *Config
		expectedConfig *Config
		err            string
	}{
		// block version not copied to wal if populated
		{
			cfg: &Config{
				WAL: &wal.Config{
					Version: "vParquet2",
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "vParquet4",
				},
			},
			expectedConfig: &Config{
				WAL: &wal.Config{
					Version: "vParquet2",
				},
				Block: &common.BlockConfig{
					IndexDownsampleBytes: 1,
					IndexPageSizeBytes:   1,
					BloomFP:              0.01,
					BloomShardSizeBytes:  1,
					Version:              "vParquet4",
				},
			},
		},
	}

	for _, test := range tests {
		err := validateConfig(test.cfg)
		if test.err == "" {
			require.Equal(t, nil, err)
		} else {
			assert.Contains(t, err.Error(), test.err)
		}

		if test.expectedConfig != nil {
			require.Equal(t, test.expectedConfig, test.cfg)
		}
	}
}

func TestValidateCompactorConfig(t *testing.T) {
	compactorConfig := CompactorConfig{
		MaxCompactionRange: 0,
	}

	expected := errors.New("Compaction window can't be 0")
	actual := compactorConfig.validate()

	require.Equal(t, expected, actual)
}
