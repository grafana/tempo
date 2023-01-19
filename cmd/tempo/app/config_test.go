package app

import (
	"testing"
	"time"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/stretchr/testify/assert"
)

func TestConfig_CheckConfig(t *testing.T) {
	tt := []struct {
		name   string
		config *Config
		expect []ConfigWarning
	}{
		{
			name:   "check default cfg and expect no warnings",
			config: newDefaultConfig(),
			expect: nil,
		},
		{
			name: "hit all except local backend warnings",
			config: &Config{
				Target: MetricsGenerator,
				StorageConfig: storage.Config{
					Trace: tempodb.Config{
						Backend:       "s3",
						BlocklistPoll: time.Minute,
						Block: &common.BlockConfig{
							Version: "v2",
						},
					},
				},
				Distributor: distributor.Config{
					LogReceivedTraces: true,
				},
			},
			expect: []ConfigWarning{
				warnCompleteBlockTimeout,
				warnBlockRetention,
				warnRetentionConcurrency,
				warnStorageTraceBackendS3,
				warnBlocklistPollConcurrency,
				warnLogReceivedTraces,
			},
		},
		{
			name: "hit local backend warnings",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace = tempodb.Config{
					Backend:                  "local",
					BlocklistPollConcurrency: 1,
					Block: &common.BlockConfig{
						Version: "v2",
					},
				}
				cfg.Target = "something"
				return cfg
			}(),
			expect: []ConfigWarning{warnStorageTraceBackendLocal},
		},
		{
			name: "warnings for v2 settings when they drift from default",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Block.Version = vparquet.VersionString
				cfg.StorageConfig.Trace.Block.IndexDownsampleBytes = 1
				cfg.StorageConfig.Trace.Block.IndexPageSizeBytes = 1
				cfg.Compactor.Compactor.ChunkSizeBytes = 1
				cfg.Compactor.Compactor.FlushSizeBytes = 1
				cfg.Compactor.Compactor.IteratorBufferSize = 1
				return cfg
			}(),
			expect: []ConfigWarning{
				newV2Warning("v2_index_downsample_bytes"),
				newV2Warning("v2_index_page_size_bytes"),
				newV2Warning("v2_in_buffer_bytes"),
				newV2Warning("v2_out_buffer_bytes"),
				newV2Warning("v2_prefetch_traces_count"),
			},
		},
		{
			name: "no warnings for v2 settings when they drift from default and v2 is the block version",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Block.Version = v2.VersionString
				cfg.StorageConfig.Trace.Block.IndexDownsampleBytes = 1
				cfg.StorageConfig.Trace.Block.IndexPageSizeBytes = 1
				cfg.Compactor.Compactor.ChunkSizeBytes = 1
				cfg.Compactor.Compactor.FlushSizeBytes = 1
				cfg.Compactor.Compactor.IteratorBufferSize = 1
				return cfg
			}(),
			expect: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			warnings := tc.config.CheckConfig()
			assert.Equal(t, tc.expect, warnings)
		})
	}
}
