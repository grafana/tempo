package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
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
						Backend:       backend.S3,
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
					Backend:                  backend.Local,
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
				cfg.StorageConfig.Trace.Block.Version = vparquet2.VersionString
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
		{
			name: "trace storage conflicts with overrides storage - local",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.Local
				cfg.StorageConfig.Trace.Local.Path = "/var/tempo"
				cfg.LimitsConfig.UserConfigurableOverrides.Client.Backend = backend.Local
				cfg.LimitsConfig.UserConfigurableOverrides.Client.Local.Path = "/var/tempo"
				return cfg
			}(),
			expect: []ConfigWarning{warnTracesAndUserConfigurableOverridesStorageConflict},
		},
		{
			name: "trace storage conflicts with overrides storage - gcs",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.GCS
				cfg.StorageConfig.Trace.GCS.BucketName = "bucketname"
				cfg.StorageConfig.Trace.GCS.Prefix = "tempo"
				cfg.LimitsConfig.UserConfigurableOverrides.Client.Backend = backend.GCS
				cfg.LimitsConfig.UserConfigurableOverrides.Client.GCS.BucketName = "bucketname"
				cfg.LimitsConfig.UserConfigurableOverrides.Client.GCS.Prefix = "tempo"
				return cfg
			}(),
			expect: []ConfigWarning{warnTracesAndUserConfigurableOverridesStorageConflict},
		},
		{
			name: "trace storage conflicts with overrides storage - different backends",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.GCS
				cfg.StorageConfig.Trace.GCS.BucketName = "my-bucket"
				cfg.LimitsConfig.UserConfigurableOverrides.Client.Backend = backend.S3
				cfg.LimitsConfig.UserConfigurableOverrides.Client.S3.Bucket = "my-bucket"
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
