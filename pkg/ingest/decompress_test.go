package ingest

import (
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grafana/tempo/v3/pkg/util/test"
)

// TestReaderClientMaxDecompressBatchBytes checks that the reader client gets
// the configured decompress limit.
func TestReaderClientMaxDecompressBatchBytes(t *testing.T) {
	const franzGoDefault = 1 << 30

	tests := []struct {
		name  string
		value int
		want  int
	}{
		{name: "default", value: DefaultMaxDecompressBatchBytes, want: DefaultMaxDecompressBatchBytes},
		{name: "lower limit", value: 1 << 20, want: 1 << 20},
		{name: "zero uses the franz-go default", value: 0, want: franzGoDefault},
		{name: "too high is capped at the default", value: DefaultMaxDecompressBatchBytes + 1, want: DefaultMaxDecompressBatchBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := KafkaConfig{}
			flagext.DefaultValues(&cfg)
			cfg.Address = flagext.StringSliceCSV{"localhost:9092"}
			cfg.Topic = "test"
			cfg.MaxDecompressBatchBytes = tt.value

			client, err := NewReaderClient(cfg, nil, test.NewTestingLogger(t))
			require.NoError(t, err)
			t.Cleanup(client.Close)

			require.Equal(t, tt.want, client.OptValue(kgo.MaxDecompressBatchBytes))
		})
	}
}

func TestKafkaConfig_Validate_MaxDecompressBatchBytes(t *testing.T) {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Topic = "test"
	require.Equal(t, DefaultMaxDecompressBatchBytes, cfg.MaxDecompressBatchBytes)

	for _, v := range []int{0, 1, DefaultMaxDecompressBatchBytes} {
		cfg.MaxDecompressBatchBytes = v
		require.NoError(t, cfg.Validate(), "value %d", v)
	}
	for _, v := range []int{-1, DefaultMaxDecompressBatchBytes + 1} {
		cfg.MaxDecompressBatchBytes = v
		require.ErrorIs(t, cfg.Validate(), ErrInvalidMaxDecompressBatchBytes, "value %d", v)
	}
}
