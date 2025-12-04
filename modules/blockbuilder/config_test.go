package blockbuilder

import (
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
)

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectedErr bool
	}{
		{
			name: "ValidConfig",
			cfg: Config{
				BlockConfig: BlockConfig{
					BlockCfg: common.BlockConfig{
						Version:              encoding.LatestEncoding().Version(),
						IndexDownsampleBytes: 1,
						IndexPageSizeBytes:   1,
						BloomFP:              0.1,
						BloomShardSizeBytes:  1,
						DedicatedColumns: backend.DedicatedColumns{
							{Scope: backend.DedicatedColumnScopeResource, Name: "foo", Type: backend.DedicatedColumnTypeString},
						},
					},
				},
				WAL: wal.Config{
					Version: encoding.LatestEncoding().Version(),
				},
			},
			expectedErr: false,
		},
		{
			name: "InvalidBlockConfig",
			cfg: Config{
				BlockConfig: BlockConfig{
					BlockCfg: common.BlockConfig{
						Version:              vparquet4.VersionString,
						IndexDownsampleBytes: 0,
					},
				},
				WAL: wal.Config{
					Version: v2.VersionString,
				},
			},
			expectedErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			assert.Equal(t, tc.expectedErr, err != nil, "unexpected error: %v", err)
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
			name: "partitions_per_instance takes precedence",
			cfg: Config{
				InstanceID:            instanceID,
				PartitionsPerInstance: 2,
				AssignedPartitionsMap: map[string][]int32{
					instanceID: {1, 2, 56},
				},
			},
			expectedPartitions: []int32{84, 85},
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
