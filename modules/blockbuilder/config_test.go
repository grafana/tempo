package blockbuilder

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectedErr string
	}{
		{
			name: "Default",
			cfg: func() Config {
				cfg := Config{}
				cfg.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
				cfg.PartitionsPerInstance = 2
				return cfg
			}(),
		},
		{
			name: "ValidConfig",
			cfg:  Config{PartitionsPerInstance: 5},
		},
		{
			name:        "InvalidPartitionAssignment",
			cfg:         Config{},
			expectedErr: "at least one of AssignedPartitionsMap or PartitionsPerInstance must be set",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
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
