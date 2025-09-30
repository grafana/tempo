package livestore

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.uber.org/atomic"
)

const (
	testTopic         = "test-topic"
	testConsumerGroup = "test-consumer-group"
	testPartition     = int32(0)
)

func TestPartitionReaderCommits(t *testing.T) {
	t.Run("sync commits", func(t *testing.T) {
		k, address := testkafka.CreateCluster(t, 1, testTopic)

		kafkaCommits := atomic.NewInt32(0)
		k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
			kafkaCommits.Inc()
			return nil, nil, false
		})

		client := testkafka.NewKafkaClient(t, address, testTopic)
		testkafka.SendReq(t.Context(), t, client, ingest.Encode, testTenantID)

		consumeFn := func(_ context.Context, rs recordIter, _ time.Time) (*kadm.Offset, error) {
			var lastRecord *kgo.Record
			for !rs.Done() {
				lastRecord = rs.Next()
			}
			offset := kadm.NewOffsetFromRecord(lastRecord)
			return &offset, nil
		}

		// commitInterval=0 commits synchronously
		r := defaultPartitionReaderWithCommitInterval(t, address, 0, consumeFn)

		assert.Eventually(t, func() bool { return kafkaCommits.Load() >= 1 }, time.Second*2, 10*time.Millisecond)

		t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), r)) })
	})

	t.Run("async commits", func(t *testing.T) {
		var asyncCommits atomic.Int32

		commitInterval := 5 * time.Second

		k, address := testkafka.CreateCluster(t, 1, testTopic)
		k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
			asyncCommits.Inc()
			return nil, nil, false
		})

		client := testkafka.NewKafkaClient(t, address, testTopic)
		testkafka.SendReq(t.Context(), t, client, ingest.Encode, testTenantID)

		consumed := make(chan struct{})
		consumeFn := func(_ context.Context, rs recordIter, _ time.Time) (*kadm.Offset, error) {
			defer close(consumed)
			var lastRecord *kgo.Record
			for !rs.Done() {
				lastRecord = rs.Next()
			}
			offset := kadm.NewOffsetFromRecord(lastRecord)
			return &offset, nil
		}

		r := defaultPartitionReaderWithCommitInterval(t, address, commitInterval, consumeFn)

		<-consumed                                     // Assert that record has been consumed
		assert.Equal(t, asyncCommits.Load(), int32(0)) // Nothing committed

		// Waiting up to commitInterval, a commit will have happened by then
		assert.Eventually(t, func() bool { return asyncCommits.Load() >= 1 }, commitInterval*2, 10*time.Millisecond)

		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), r))
	})
}

func defaultPartitionReaderWithCommitInterval(t *testing.T, address string, commitInterval time.Duration, consume consumeFn) *PartitionReader {
	l := &testLogger{t}

	cfg := ingest.KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Address = address
	cfg.Topic = testTopic
	cfg.ConsumerGroup = testConsumerGroup

	client, err := ingest.NewReaderClient(
		cfg,
		ingest.NewReaderClientMetrics(liveStoreServiceName, prometheus.NewRegistry()),
		l,
	)
	require.NoError(t, err)

	r, err := newPartitionReader(client, 0, cfg, commitInterval, time.Hour, consume, l, newPartitionReaderMetrics(testPartition, prometheus.NewRegistry()))
	require.NoError(t, err)

	err = services.StartAndAwaitRunning(t.Context(), r)
	require.NoError(t, err)

	return r
}
