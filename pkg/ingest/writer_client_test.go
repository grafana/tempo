package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/plugin/kprom"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/v3/pkg/util/test"
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
	cfg := KafkaConfig{Address: flagext.StringSliceCSV{"localhost:9092"}, Topic: "test", ClientRack: "us-east-1a"}

	opts, err := commonKafkaClientOptions(cfg, metrics, test.NewTestingLogger(t))
	require.NoError(t, err)

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)
	t.Cleanup(client.Close)
}

func TestCommonKafkaClientOptions_EmptyClientRack(t *testing.T) {
	// An empty ClientRack must not add the Rack option, so rack-aware fetching
	// stays disabled by default.
	metrics := kprom.NewMetrics("", kprom.Registerer(prometheus.NewPedanticRegistry()))
	cfg := KafkaConfig{Address: flagext.StringSliceCSV{"localhost:9092"}, Topic: "test"}

	opts, err := commonKafkaClientOptions(cfg, metrics, test.NewTestingLogger(t))
	require.NoError(t, err)

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)
	t.Cleanup(client.Close)
}

// TestProducerBufferedProduceBytesMetrics guards against the kprom client metrics and the Producer metrics
// colliding. It wires them up the same way the distributor does, on a shared prefixed registerer.
func TestProducerBufferedProduceBytesMetrics(t *testing.T) {
	cluster, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.AllowAutoTopicCreation())
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	reg := prometheus.NewPedanticRegistry()
	cfg := KafkaConfig{Address: cluster.ListenAddrs()[0], Topic: "test", WriteTimeout: 10 * time.Second}

	client, err := NewWriterClient(cfg, 10, test.NewTestingLogger(t), prometheus.WrapRegistererWithPrefix("tempo_distributor_", reg))
	require.NoError(t, err)

	require.NotPanics(t, func() {
		producer := NewProducer(client, 1024, reg)
		t.Cleanup(producer.Close)
	})

	mfs, err := reg.Gather()
	require.NoError(t, err)

	counts := map[string]int{}
	for _, mf := range mfs {
		counts[mf.GetName()]++
	}
	require.Equal(t, 1, counts["tempo_distributor_buffered_produce_bytes"])
	require.Equal(t, 1, counts["tempo_distributor_buffered_produce_bytes_distribution"])
	require.Equal(t, 1, counts["tempo_distributor_buffered_produce_bytes_limit"])
}

// TestProducerBufferedProduceBytesValues checks that the exported gauge and distribution summary track the bytes buffered in the Kafka
// client: it rises while a produce request is held by the broker and falls back to zero once it completes.
func TestProducerBufferedProduceBytesValues(t *testing.T) {
	const (
		valueSize        = 10_000
		maxBufferedBytes = 1 << 20
	)

	cluster, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, "test"))
	require.NoError(t, err)
	t.Cleanup(cluster.Close)

	// Hold produce requests at the broker so the record stays buffered in the client.
	release := make(chan struct{})
	cluster.ControlKey(kmsg.Produce.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		cluster.KeepControl()
		<-release
		return nil, nil, false
	})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)

	reg := prometheus.NewPedanticRegistry()
	cfg := KafkaConfig{Address: cluster.ListenAddrs()[0], Topic: "test", WriteTimeout: 10 * time.Second}

	client, err := NewWriterClient(cfg, 10, test.NewTestingLogger(t), prometheus.WrapRegistererWithPrefix("tempo_distributor_", reg))
	require.NoError(t, err)
	producer := NewProducer(client, maxBufferedBytes, reg)
	t.Cleanup(producer.Close)

	gauge := func(name string) float64 {
		mfs, err := reg.Gather()
		require.NoError(t, err)
		for _, mf := range mfs {
			if mf.GetName() == name {
				return mf.GetMetric()[0].GetGauge().GetValue()
			}
		}
		require.Failf(t, "metric not found", "%s", name)
		return 0
	}

	// peak returns the max (quantile 1) of the distribution summary.
	peak := func() float64 {
		mfs, err := reg.Gather()
		require.NoError(t, err)
		for _, mf := range mfs {
			if mf.GetName() != "tempo_distributor_buffered_produce_bytes_distribution" {
				continue
			}
			for _, q := range mf.GetMetric()[0].GetSummary().GetQuantile() {
				if q.GetQuantile() == 1 {
					return q.GetValue()
				}
			}
		}
		require.Fail(t, "distribution summary not found")
		return 0
	}

	require.Equal(t, 0.0, gauge("tempo_distributor_buffered_produce_bytes"))
	require.Equal(t, float64(maxBufferedBytes), gauge("tempo_distributor_buffered_produce_bytes_limit"))

	done := make(chan kgo.ProduceResults, 1)
	go func() {
		done <- producer.ProduceSync(context.Background(), []*kgo.Record{{Value: make([]byte, valueSize), Partition: 0}})
	}()

	require.Eventually(t, func() bool {
		return gauge("tempo_distributor_buffered_produce_bytes") >= valueSize
	}, 5*time.Second, 10*time.Millisecond)
	require.Less(t, gauge("tempo_distributor_buffered_produce_bytes"), float64(valueSize+1024))

	// The summary samples every 250ms, so its max should catch the same buffered value.
	require.Eventually(t, func() bool { return peak() >= valueSize }, 5*time.Second, 10*time.Millisecond)
	require.Less(t, peak(), float64(valueSize+1024))

	releaseOnce()
	res := <-done
	require.NoError(t, res.FirstErr())

	require.Eventually(t, func() bool {
		return gauge("tempo_distributor_buffered_produce_bytes") == 0
	}, 5*time.Second, 10*time.Millisecond)

	// The peak is retained over the 60s window even after the buffer drains.
	require.GreaterOrEqual(t, peak(), float64(valueSize))
}
