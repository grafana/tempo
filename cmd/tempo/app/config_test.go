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
		{
			name: "trace storage has too many dedicated columns",
			config: func() *Config {
				cfg := NewDefaultConfig()
				cfg.StorageConfig.Trace.Block.DedicatedColumns = backend.DedicatedColumns{
					{Name: "dedicated.resource.1", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.2", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.3", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.4", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.5", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.6", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.7", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.8", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.9", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.10", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.11", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.12", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.13", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.14", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.15", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.16", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.17", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.18", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.19", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.20", Type: "string", Scope: "resource"},
					{Name: "dedicated.resource.21", Type: "string", Scope: "resource"},
				}
				return cfg
			}(),
			expect: []ConfigWarning{
				{
					Message: (backend.WarnTooManyColumns{Type: "string", Scope: "resource", Count: 21, MaxCount: 20}).Error(),
					Explain: "Dedicated attribute column configuration contains an invalid configuration that will be ignored",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			warnings := tc.config.CheckConfig()
			assert.Equal(t, tc.expect, warnings)
		})
	}
}
