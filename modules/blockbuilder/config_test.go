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
