package blockbuilder

import (
	"errors"
	"flag"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectedErr error
	}{
		{
			name: "Default",
			cfg: func() Config {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
				cfg.PartitionsPerInstance = 2
				return cfg
			}(),
			expectedErr: nil,
		},
		{
			name: "ValidConfig",
			cfg: Config{
				BlockConfig: BlockConfig{
					BlockCfg: common.BlockConfig{
						Version:             encoding.LatestEncoding().Version(),
						BloomFP:             0.1,
						BloomShardSizeBytes: 1,
						DedicatedColumns: backend.DedicatedColumns{
							{Scope: backend.DedicatedColumnScopeResource, Name: "foo", Type: backend.DedicatedColumnTypeString},
						},
					},
				},
				WAL: wal.Config{
					Version: encoding.LatestEncoding().Version(),
				},
				PartitionsPerInstance: 5,
			},
			expectedErr: nil,
		},
		{
			name: "InvalidBlockVersion",
			cfg: Config{
				BlockConfig: BlockConfig{
					BlockCfg: common.BlockConfig{
						// This parses for reads but not for writes
						Version: "vParquet5-preview1",
					},
				},
			},
			expectedErr: errors.New("block version validation failed: vParquet5-preview1 is not a valid block version for creating blocks"),
		},
		{
			name: "InvalidPartitionAssignment",
			cfg: Config{
				BlockConfig: BlockConfig{
					BlockCfg: common.BlockConfig{
						Version:             encoding.LatestEncoding().Version(),
						BloomFP:             0.1,
						BloomShardSizeBytes: 1,
						DedicatedColumns: backend.DedicatedColumns{
							{Scope: backend.DedicatedColumnScopeResource, Name: "foo", Type: backend.DedicatedColumnTypeString},
						},
					},
				},
				WAL: wal.Config{
					Version: encoding.LatestEncoding().Version(),
				},
			},
			expectedErr: errors.New("at least one of AssignedPartitionsMap or PartitionsPerInstance must be set"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tc.expectedErr.Error(), err.Error())
			}
		})
	}
}

func TestCoalesceBlockVersion(t *testing.T) {
	defaultVer := encoding.DefaultEncoding().Version()

	tests := []struct {
		name            string
		modifyConfig    func(*Config)
		expectedVersion string
		expectedErr     string
	}{
		{
			name:            "uses default when all version fields are empty",
			modifyConfig:    func(_ *Config) {},
			expectedVersion: defaultVer,
		},
		{
			name: "fallback to GlobalBlockConfig when block_config version is empty",
			modifyConfig: func(cfg *Config) {
				cfg.GlobalBlockConfig = &common.BlockConfig{Version: encoding.LatestEncoding().Version()}
			},
			expectedVersion: encoding.LatestEncoding().Version(),
		},
		{
			name: "block_config version overrides GlobalBlockConfig",
			modifyConfig: func(cfg *Config) {
				cfg.GlobalBlockConfig = &common.BlockConfig{Version: "vParquet4"}
				cfg.BlockConfig.BlockCfg.Version = encoding.LatestEncoding().Version()
			},
			expectedVersion: encoding.LatestEncoding().Version(),
		},
		{
			name: "WAL version follows block version",
			modifyConfig: func(cfg *Config) {
				cfg.GlobalBlockConfig = &common.BlockConfig{Version: encoding.LatestEncoding().Version()}
			},
			expectedVersion: encoding.LatestEncoding().Version(),
		},
		{
			name: "unsupported block version returns error",
			modifyConfig: func(cfg *Config) {
				cfg.BlockConfig.BlockCfg.Version = "preview"
			},
			expectedErr: "preview",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.PanicOnError))
			tt.modifyConfig(cfg)

			enc, err := coalesceBlockVersion(cfg)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVersion, enc.Version())
			assert.Equal(t, tt.expectedVersion, cfg.BlockConfig.BlockCfg.Version)
			assert.Equal(t, tt.expectedVersion, cfg.WAL.Version)
		})
	}
}

func TestConfig_partitionAssignment(t *testing.T) {
	instanceID := "block-builder-42"
	for _, tc := range []struct {
		name               string
		cfg                Config
		expectedPartitions []int32
	}{
		{
			name: "assigned_partitions",
			cfg: Config{
				InstanceID: instanceID,
				AssignedPartitionsMap: map[string][]int32{
					instanceID: {1, 2, 56},
				},
			},
			expectedPartitions: []int32{1, 2, 56},
		},
		{
			name: "partitions_per_instance",
			cfg: Config{
				InstanceID:            instanceID,
				PartitionsPerInstance: 2,
			},
			expectedPartitions: []int32{84, 85},
		},
		{
			name: "assigned_partitions takes precedence",
			cfg: Config{
				InstanceID:            instanceID,
				PartitionsPerInstance: 2,
				AssignedPartitionsMap: map[string][]int32{
					instanceID: {1, 2, 56},
				},
			},
			expectedPartitions: []int32{1, 2, 56},
		},
		{
			name: "falls back to assigned_partitions if ID doesn't have an index",
			cfg: Config{
				InstanceID: "block-builder",
				AssignedPartitionsMap: map[string][]int32{
					"block-builder": {1, 2, 56},
				},
				PartitionsPerInstance: 2,
			},
			expectedPartitions: []int32{1, 2, 56},
		},
		{
			name: "returns nil if no instance ID",
			cfg: Config{
				InstanceID: "",
				AssignedPartitionsMap: map[string][]int32{
					instanceID: {1, 2, 56},
				},
				PartitionsPerInstance: 2,
			},
			expectedPartitions: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedPartitions, tc.cfg.AssignedPartitions())
		})
	}
}
