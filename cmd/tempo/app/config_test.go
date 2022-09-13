package app

import (
	"testing"
	"time"

	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/encoding/common"

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
						Block:         &common.BlockConfig{},
					},
				},
				Distributor: distributor.Config{
					LogReceivedTraces: true,
				},
			},
			expect: []ConfigWarning{
				warnMetricsGenerator,
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
					Block:                    &common.BlockConfig{},
				}
				cfg.Target = "something"
				return cfg
			}(),
			expect: []ConfigWarning{warnStorageTraceBackendLocal},
		},
		{
			name: "warn ingester search",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.StorageConfig.Trace.Block.Version = "v2"
				return cfg
			}(),
			expect: []ConfigWarning{warnIngesterSearchWillNotWork},
		},
		{
			name: "warn flatbuffers not necessary",
			config: func() *Config {
				cfg := newDefaultConfig()
				cfg.Ingester.UseFlatbufferSearch = true
				return cfg
			}(),
			expect: []ConfigWarning{warnFlatBuffersNotNecessary},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			warnings := tc.config.CheckConfig()
			assert.Equal(t, tc.expect, warnings)
		})
	}
}
