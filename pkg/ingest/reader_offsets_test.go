package ingest_test

import (
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/v3/pkg/ingest"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func offsetsTestConfig() ingest.KafkaConfig {
	cfg := ingest.KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Address = flagext.StringSliceCSV{"localhost:9092"}
	cfg.Topic = "test"
	cfg.ConsumerGroup = "test-group"
	return cfg
}

// TestReaderClientResetOffset checks the reader client's reset and start
// offsets: by default RewindOffset(0), so after data loss the client resumes
// where it was, as on franz-go < 1.22. A caller's reset offset replaces it.
func TestReaderClientResetOffset(t *testing.T) {
	tests := []struct {
		name      string
		opts      []kgo.Opt
		wantReset kgo.Offset
		wantStart kgo.Offset
	}{
		{
			name:      "default",
			wantReset: kgo.RewindOffset(0),
			wantStart: kgo.NewOffset().AtStart(),
		},
		{
			name:      "caller replaces the reset offset",
			opts:      []kgo.Opt{kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd())},
			wantReset: kgo.NewOffset().AtEnd(),
			wantStart: kgo.NewOffset().AtEnd(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := ingest.NewReaderClient(offsetsTestConfig(), nil, log.NewNopLogger(), tt.opts...)
			require.NoError(t, err)
			t.Cleanup(client.Close)

			require.Equal(t, tt.wantReset, client.OptValue(kgo.ConsumeResetOffset))
			require.Equal(t, tt.wantStart, client.OptValue(kgo.ConsumeStartOffset))
		})
	}
}

// TestGroupReaderClientOptions checks that the metrics-generator's group client
// starts new partitions at the beginning, after data loss resumes where it was
// instead of re-reading the whole partition, and gets the decompress limit.
func TestGroupReaderClientOptions(t *testing.T) {
	client, err := ingest.NewGroupReaderClient(offsetsTestConfig(), mustRebalanceRing(t, 1), nil, log.NewNopLogger())
	require.NoError(t, err)
	t.Cleanup(client.Close)

	require.Equal(t, kgo.RewindOffset(0), client.OptValue(kgo.ConsumeResetOffset))
	require.Equal(t, kgo.NewOffset().AtStart(), client.OptValue(kgo.ConsumeStartOffset))
	require.Equal(t, ingest.DefaultMaxDecompressBatchBytes, client.OptValue(kgo.MaxDecompressBatchBytes))
}
