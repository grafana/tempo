package app

import (
	"testing"
	"time"

	"github.com/grafana/tempo/modules/blockbuilder"
	"github.com/grafana/tempo/modules/frontend"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func TestConfig_CheckConfig(t *testing.T) {
	tt := []struct {
		name   string
		config *Config
		expect []ConfigWarning
	}{
		{
			name:   "check default cfg and expect no warnings",
			config: NewDefaultConfig(),
			expect: nil,
		},
		{
			name: "hit all except local backend warnings",
			config: &Config{
				Target: MetricsGenerator,
				StorageConfig: storage.Config{
					Trace: tempodb.Config{
						Cache:         "supercache",
						Backend:       backend.S3,
						BlocklistPoll: time.Minute,
						Block: &common.BlockConfig{
							Version: "v2",
						},
						S3: &s3.Config{
							NativeAWSAuthEnabled: true,
						},
					},
				},
				Distributor: distributor.Config{
					LogReceivedSpans: distributor.LogSpansConfig{
						Enabled: true,
					},
					LogDiscardedSpans: distributor.LogSpansConfig{
						Enabled: true,
					},
				},
				Frontend: frontend.Config{
					MCPServer: frontend.MCPServerConfig{
						Enabled: true,
					},
					TraceByID: frontend.TraceByIDConfig{
						QueryShards:      100,
						ConcurrentShards: 200,
					},
				},
				BlockBuilder: blockbuilder.Config{
					PartitionsPerInstance: 20,
					AssignedPartitionsMap: map[string][]int32{
						"foo-0": {0},
					},
				},
			},
			expect: []ConfigWarning{
				warnCompleteBlockTimeout,
				warnBlockRetention,
				warnRetentionConcurrency,
				warnBlocklistPollConcurrency,
				warnLogReceivedTraces,
				warnLogDiscardedTraces,
				warnMCPServerEnabled,
				warnNativeAWSAuthEnabled,
				warnConfiguredLegacyCache,
				warnTraceByIDConcurrentShards,
				warnBackendSchedulerPruneAgeLessThanBlocklistPoll,
				warnPartitionAssigmentCollision,
			},
		},
		{
			name: "hit local backend warnings",
			config: func() *Config {
				cfg := NewDefaultConfig()
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
			name: "trace storage conflicts with overrides storage - local",
			config: func() *Config {
				cfg := NewDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.Local
				cfg.StorageConfig.Trace.Local.Path = "/var/tempo"
				cfg.Overrides.UserConfigurableOverridesConfig.Client.Backend = backend.Local
				cfg.Overrides.UserConfigurableOverridesConfig.Client.Local.Path = "/var/tempo"
				return cfg
			}(),
			expect: []ConfigWarning{warnTracesAndUserConfigurableOverridesStorageConflict},
		},
		{
			name: "trace storage conflicts with overrides storage - gcs",
			config: func() *Config {
				cfg := NewDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.GCS
				cfg.StorageConfig.Trace.GCS.BucketName = "bucketname"
				cfg.StorageConfig.Trace.GCS.Prefix = "tempo"
				cfg.Overrides.UserConfigurableOverridesConfig.Client.Backend = backend.GCS
				cfg.Overrides.UserConfigurableOverridesConfig.Client.GCS.BucketName = "bucketname"
				cfg.Overrides.UserConfigurableOverridesConfig.Client.GCS.Prefix = "tempo"
				return cfg
			}(),
			expect: []ConfigWarning{warnTracesAndUserConfigurableOverridesStorageConflict},
		},
		{
			name: "trace storage conflicts with overrides storage - different backends",
			config: func() *Config {
				cfg := NewDefaultConfig()
				cfg.StorageConfig.Trace.Backend = backend.GCS
				cfg.StorageConfig.Trace.GCS.BucketName = "my-bucket"
				cfg.Overrides.UserConfigurableOverridesConfig.Client.Backend = backend.S3
				cfg.Overrides.UserConfigurableOverridesConfig.Client.S3.Bucket = "my-bucket"
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
