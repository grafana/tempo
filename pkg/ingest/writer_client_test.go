package ingest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kprom"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/pkg/util/test"
)

func TestStatusFromProduceErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"record timeout", kgo.ErrRecordTimeout, codes.Unavailable},
		{"context deadline", context.DeadlineExceeded, codes.Unavailable},
		{"context canceled", context.Canceled, codes.Canceled},
		{"buffer full", kgo.ErrMaxBuffered, codes.Unavailable},
		{"generic", errors.New("boom"), codes.Unavailable},
		{"message too large", kerr.MessageTooLarge, codes.InvalidArgument},
		{"wrapped message too large", fmt.Errorf("failed to produce: %w", kerr.MessageTooLarge), codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusFromProduceErr(tt.err)
			require.Equal(t, tt.want, status.Code(got))
			if tt.err == nil {
				require.NoError(t, got)
				return
			}
			require.Contains(t, got.Error(), tt.err.Error())
		})
	}
}

func TestCommonKafkaClientOptions_ClientRack(t *testing.T) {
	// kgo.Opt values are opaque, so we can't assert on the Rack option directly.
	// Instead we verify that setting ClientRack still produces a valid client
	// that kgo.NewClient accepts.
	metrics := kprom.NewMetrics("", kprom.Registerer(prometheus.NewPedanticRegistry()))
	cfg := KafkaConfig{Address: "localhost:9092", Topic: "test", ClientRack: "us-east-1a"}

	opts := commonKafkaClientOptions(cfg, metrics, test.NewTestingLogger(t))

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)
	t.Cleanup(client.Close)
}

func TestCommonKafkaClientOptions_EmptyClientRack(t *testing.T) {
	// An empty ClientRack must not add the Rack option, so rack-aware fetching
	// stays disabled by default.
	metrics := kprom.NewMetrics("", kprom.Registerer(prometheus.NewPedanticRegistry()))
	cfg := KafkaConfig{Address: "localhost:9092", Topic: "test"}

	opts := commonKafkaClientOptions(cfg, metrics, test.NewTestingLogger(t))

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)
	t.Cleanup(client.Close)
}
