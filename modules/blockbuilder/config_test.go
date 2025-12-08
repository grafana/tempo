package blockbuilder

import (
	"errors"
	"flag"
	"testing"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	v2 "github.com/grafana/tempo/tempodb/encoding/v2"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/tempodb/wal"
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
				return cfg
			}(),
			expectedErr: nil,
		},
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
			expectedErr: nil,
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
			expectedErr: errors.New("block config validation failed: positive index downsample required"),
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
