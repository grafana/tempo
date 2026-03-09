package distributor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/ingest"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "kafka write path enabled with valid kafka config",
			cfg: Config{
				KafkaWritePathEnabled:    true,
				IngesterWritePathEnabled: false,
				KafkaConfig: ingest.KafkaConfig{
					Address:                    "localhost:9092",
					Topic:                      "tempo-traces",
					ProducerMaxRecordSizeBytes: 15983616, // max allowed default
				},
			},
			wantErr: false,
		},
		{
			name: "ingester write path enabled",
			cfg: Config{
				IngesterWritePathEnabled: true,
				KafkaWritePathEnabled:    false,
			},
			wantErr: false,
		},
		{
			name: "both write paths enabled with valid kafka config",
			cfg: Config{
				IngesterWritePathEnabled: true,
				KafkaWritePathEnabled:    true,
				KafkaConfig: ingest.KafkaConfig{
					Address:                    "localhost:9092",
					Topic:                      "tempo-traces",
					ProducerMaxRecordSizeBytes: 15983616,
				},
			},
			wantErr: false,
		},
		{
			name: "both write paths disabled - no write path error",
			cfg: Config{
				IngesterWritePathEnabled: false,
				KafkaWritePathEnabled:    false,
			},
			wantErr: true,
			errMsg:  "distributor has no write path configured",
		},
		{
			name: "kafka write path enabled but missing kafka address",
			cfg: Config{
				KafkaWritePathEnabled:    true,
				IngesterWritePathEnabled: false,
				KafkaConfig: ingest.KafkaConfig{
					Topic: "tempo-traces",
					// Address intentionally missing
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
