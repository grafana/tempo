package livestore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/util/test"
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
		assert.Equal(t, int64(0), r.lag.Load())

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
		assert.Equal(t, int64(0), r.lag.Load())

		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), r))
	})
}

func TestPartitionReaderLag(t *testing.T) {
	k, address := testkafka.CreateCluster(t, 1, testTopic)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	var counter int
	consumeFn := func(_ context.Context, rs recordIter, _ time.Time) (*kadm.Offset, error) {
		counter++
		if counter <= 1 { // allow to process one record to store lag
			lastRecord := rs.Next()
			offset := kadm.NewOffsetFromRecord(lastRecord)
			return &offset, nil
		}
		return nil, errors.New("error consuming records")
	}

	// commitInterval=0 commits synchronously
	r := defaultPartitionReaderWithCommitInterval(t, address, 0, consumeFn)

	client := testkafka.NewKafkaClient(t, address, testTopic)
	records := 10
	for range records {
		testkafka.SendReq(t.Context(), t, client, ingest.Encode, testTenantID)
	}

	time.Sleep(time.Second)
	assert.Equal(t, int32(1), kafkaCommits.Load(), "only one record should be committed")
	assert.Equal(t, int64(records)-1, r.lag.Load(), "only one record should be committed")

	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), r)) })
}

func TestFetchLastCommittedOffsetForceFromLookback(t *testing.T) {
	lookback := time.Hour

	t.Run("committed offset exists, forceFromLookback=false uses committed offset", func(t *testing.T) {
		_, address := testkafka.CreateCluster(t, 1, testTopic)
		client := testkafka.NewKafkaClient(t, address, testTopic)

		// Produce a record and commit an offset
		testkafka.SendReq(t.Context(), t, client, ingest.Encode, testTenantID)

		adm := kadm.NewClient(client)
		offsets := make(kadm.Offsets)
		offsets.Add(kadm.Offset{
			Topic:     testTopic,
			Partition: testPartition,
			At:        0,
		})
		_, err := adm.CommitOffsets(t.Context(), testConsumerGroup, offsets)
		require.NoError(t, err)

		l := test.NewTestingLogger(t)
		cfg := ingest.KafkaConfig{}
		flagext.DefaultValues(&cfg)
		cfg.Address = address
		cfg.Topic = testTopic
		cfg.ConsumerGroup = testConsumerGroup

		readerClient, err := ingest.NewReaderClient(cfg, ingest.NewReaderClientMetrics(liveStoreServiceName, prometheus.NewRegistry()), l)
		require.NoError(t, err)

		r, err := newPartitionReader(readerClient, testPartition, cfg, 0, lookback, false, nil, l, newPartitionReaderMetrics(testPartition, prometheus.NewRegistry()))
		require.NoError(t, err)

		offset, err := r.fetchLastCommittedOffset(t.Context())
		require.NoError(t, err)

		// Should use committed offset (At(0)), not lookback
		epochOffset := offset.EpochOffset()
		assert.Equal(t, int64(0), epochOffset.Offset)
	})

	t.Run("committed offset exists, forceFromLookback=true ignores committed offset", func(t *testing.T) {
		_, address := testkafka.CreateCluster(t, 1, testTopic)
		client := testkafka.NewKafkaClient(t, address, testTopic)

		// Produce a record and commit an offset
		testkafka.SendReq(t.Context(), t, client, ingest.Encode, testTenantID)

		adm := kadm.NewClient(client)
		offsets := make(kadm.Offsets)
		offsets.Add(kadm.Offset{
			Topic:     testTopic,
			Partition: testPartition,
			At:        0,
		})
		_, err := adm.CommitOffsets(t.Context(), testConsumerGroup, offsets)
		require.NoError(t, err)

		l := test.NewTestingLogger(t)
		cfg := ingest.KafkaConfig{}
		flagext.DefaultValues(&cfg)
		cfg.Address = address
		cfg.Topic = testTopic
		cfg.ConsumerGroup = testConsumerGroup

		readerClient, err := ingest.NewReaderClient(cfg, ingest.NewReaderClientMetrics(liveStoreServiceName, prometheus.NewRegistry()), l)
		require.NoError(t, err)

		r, err := newPartitionReader(readerClient, testPartition, cfg, 0, lookback, true, nil, l, newPartitionReaderMetrics(testPartition, prometheus.NewRegistry()))
		require.NoError(t, err)

		offset, err := r.fetchLastCommittedOffset(t.Context())
		require.NoError(t, err)

		// Should use lookback period (AfterMilli), not the committed offset
		epochOffset := offset.EpochOffset()
		expectedMilli := time.Now().Add(-lookback).UnixMilli()
		assert.InDelta(t, expectedMilli, epochOffset.Offset, 5000, "offset should be near lookback time in millis")
		assert.Equal(t, int32(-1), epochOffset.Epoch, "epoch=-1 indicates AfterMilli offset")
	})

	t.Run("no committed offset, forceFromLookback=true uses lookback period", func(t *testing.T) {
		_, address := testkafka.CreateCluster(t, 1, testTopic)

		l := test.NewTestingLogger(t)
		cfg := ingest.KafkaConfig{}
		flagext.DefaultValues(&cfg)
		cfg.Address = address
		cfg.Topic = testTopic
		cfg.ConsumerGroup = testConsumerGroup

		readerClient, err := ingest.NewReaderClient(cfg, ingest.NewReaderClientMetrics(liveStoreServiceName, prometheus.NewRegistry()), l)
		require.NoError(t, err)

		r, err := newPartitionReader(readerClient, testPartition, cfg, 0, lookback, true, nil, l, newPartitionReaderMetrics(testPartition, prometheus.NewRegistry()))
		require.NoError(t, err)

		offset, err := r.fetchLastCommittedOffset(t.Context())
		require.NoError(t, err)

		// Should use lookback period
		epochOffset := offset.EpochOffset()
		expectedMilli := time.Now().Add(-lookback).UnixMilli()
		assert.InDelta(t, expectedMilli, epochOffset.Offset, 5000, "offset should be near lookback time in millis")
		assert.Equal(t, int32(-1), epochOffset.Epoch, "epoch=-1 indicates AfterMilli offset")
	})
}

func defaultPartitionReaderWithCommitInterval(t *testing.T, address string, commitInterval time.Duration, consume consumeFn) *PartitionReader {
	l := test.NewTestingLogger(t)

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

	r, err := newPartitionReader(client, 0, cfg, commitInterval, time.Hour, false, consume, l, newPartitionReaderMetrics(testPartition, prometheus.NewRegistry()))
	require.NoError(t, err)

	err = services.StartAndAwaitRunning(t.Context(), r)
	require.NoError(t, err)

	return r
}
