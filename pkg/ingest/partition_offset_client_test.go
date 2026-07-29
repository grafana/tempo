// SPDX-License-Identifier: AGPL-3.0-only

package ingest

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/plugin/kprom"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/util/test"
)

const topicName = "test"

func TestPartitionOffsetClient_FetchPartitionsLastProducedOffsets(t *testing.T) {
	const numPartitions = 3

	var (
		ctx             = context.Background()
		allPartitionIDs = []int32{0, 1, 2}
	)

	t.Run("should return the last produced offsets, or -1 if the partition is empty", func(t *testing.T) {
		t.Parallel()

		var (
			_, clusterAddr = testkafka.CreateCluster(t, numPartitions, topicName)
			kafkaCfg       = createTestKafkaConfig(clusterAddr)
			client         = createTestKafkaClient(t, kafkaCfg)
			reader         = NewPartitionOffsetClient(client, topicName)
		)

		offsets, err := reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.NoError(t, err)
		assert.Equal(t, map[int32]int64{0: 0, 1: 0, 2: 0}, getPartitionsOffsets(offsets))

		// Write some records.
		produceRecord(ctx, t, client, 0, []byte("message 1"))
		produceRecord(ctx, t, client, 0, []byte("message 2"))
		produceRecord(ctx, t, client, 1, []byte("message 3"))

		offsets, err = reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.NoError(t, err)
		assert.Equal(t, map[int32]int64{0: 2, 1: 1, 2: 0}, getPartitionsOffsets(offsets))

		// Write more records.
		produceRecord(ctx, t, client, 0, []byte("message 4"))
		produceRecord(ctx, t, client, 1, []byte("message 5"))
		produceRecord(ctx, t, client, 2, []byte("message 6"))

		offsets, err = reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.NoError(t, err)
		assert.Equal(t, map[int32]int64{0: 3, 1: 2, 2: 1}, getPartitionsOffsets(offsets))

		// Fetch offsets for a subset of partitions.
		offsets, err = reader.FetchPartitionsLastProducedOffsets(ctx, []int32{0, 2})
		require.NoError(t, err)
		assert.Equal(t, map[int32]int64{0: 3, 2: 1}, getPartitionsOffsets(offsets))
	})

	t.Run("should return error if response contains an unexpected number of topics", func(t *testing.T) {
		t.Parallel()

		cluster, clusterAddr := testkafka.CreateCluster(t, numPartitions, topicName)

		// Configure a short retry timeout.
		kafkaCfg := createTestKafkaConfig(clusterAddr)
		kafkaCfg.LastProducedOffsetRetryTimeout = time.Second

		client := createTestKafkaClient(t, kafkaCfg)
		reader := NewPartitionOffsetClient(client, topicName)

		cluster.ControlKey(kmsg.ListOffsets, func(kreq kmsg.Request) (kmsg.Response, error, bool) {
			cluster.KeepControl()

			req := kreq.(*kmsg.ListOffsetsRequest)
			res := req.ResponseKind().(*kmsg.ListOffsetsResponse)
			res.Default()
			res.Topics = []kmsg.ListOffsetsResponseTopic{
				{Topic: topicName},
				{Topic: "another-unknown-topic"},
			}

			return res, nil, true
		})

		_, err := reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.Error(t, err)
		require.ErrorContains(t, err, "unexpected number of topics in the response")
	})

	t.Run("should return error if response contains a 1 topic but its not the expected one", func(t *testing.T) {
		t.Parallel()

		cluster, clusterAddr := testkafka.CreateCluster(t, numPartitions, topicName)

		// Configure a short retry timeout.
		kafkaCfg := createTestKafkaConfig(clusterAddr)
		kafkaCfg.LastProducedOffsetRetryTimeout = time.Second

		client := createTestKafkaClient(t, kafkaCfg)
		reader := NewPartitionOffsetClient(client, topicName)

		cluster.ControlKey(kmsg.ListOffsets, func(kreq kmsg.Request) (kmsg.Response, error, bool) {
			cluster.KeepControl()

			req := kreq.(*kmsg.ListOffsetsRequest)
			res := req.ResponseKind().(*kmsg.ListOffsetsResponse)
			res.Default()
			res.Topics = []kmsg.ListOffsetsResponseTopic{
				{Topic: "another-unknown-topic"},
			}

			return res, nil, true
		})

		_, err := reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.Error(t, err)
		require.ErrorContains(t, err, "unexpected topic in the response")
	})

	t.Run("should return error if response contains an error for a partition", func(t *testing.T) {
		t.Parallel()

		cluster, clusterAddr := testkafka.CreateCluster(t, numPartitions, topicName)

		// Configure a short retry timeout.
		kafkaCfg := createTestKafkaConfig(clusterAddr)
		kafkaCfg.LastProducedOffsetRetryTimeout = time.Second

		client := createTestKafkaClient(t, kafkaCfg)
		reader := NewPartitionOffsetClient(client, topicName)

		cluster.ControlKey(kmsg.ListOffsets, func(kreq kmsg.Request) (kmsg.Response, error, bool) {
			cluster.KeepControl()

			req := kreq.(*kmsg.ListOffsetsRequest)
			res := req.ResponseKind().(*kmsg.ListOffsetsResponse)
			res.Default()
			res.Topics = []kmsg.ListOffsetsResponseTopic{
				{
					Topic: topicName,
					Partitions: []kmsg.ListOffsetsResponseTopicPartition{
						{
							Partition: 0,
							Offset:    1,
						}, {
							Partition: 0,
							ErrorCode: kerr.NotLeaderForPartition.Code,
						},
					},
				},
			}

			return res, nil, true
		})

		_, err := reader.FetchPartitionsLastProducedOffsets(ctx, allPartitionIDs)
		require.ErrorIs(t, err, kerr.NotLeaderForPartition)
	})
}

func getPartitionsOffsets(offsets kadm.ListedOffsets) map[int32]int64 {
	partitionOffsets := make(map[int32]int64)
	offsets.Each(func(offset kadm.ListedOffset) {
		partitionOffsets[offset.Partition] = offset.Offset
	})
	return partitionOffsets
}

func createTestKafkaConfig(clusterAddr string) KafkaConfig {
	cfg := KafkaConfig{}
	flagext.DefaultValues(&cfg)

	fastFetchBackoffConfig := backoff.Config{
		MinBackoff: 10 * time.Millisecond,
		MaxBackoff: 10 * time.Millisecond,
		MaxRetries: 0,
	}

	cfg.Address = clusterAddr
	cfg.Topic = topicName
	cfg.WriteTimeout = 5 * time.Second
	cfg.concurrentFetchersFetchBackoffConfig = fastFetchBackoffConfig

	return cfg
}

func createTestKafkaClient(t *testing.T, cfg KafkaConfig) *kgo.Client {
	metrics := kprom.NewMetrics("", kprom.Registerer(prometheus.NewPedanticRegistry()))
	opts := commonKafkaClientOptions(cfg, metrics, test.NewTestingLogger(t))

	// Use the manual partitioner because produceRecord() utility explicitly specifies
	// the partition to write to in the kgo.Record itself.
	opts = append(opts, kgo.RecordPartitioner(kgo.ManualPartitioner()))

	client, err := kgo.NewClient(opts...)
	require.NoError(t, err)

	// Automatically close it at the end of the test.
	t.Cleanup(client.Close)

	return client
}

func produceRecord(ctx context.Context, t *testing.T, writeClient *kgo.Client, partitionID int32, content []byte) {
	_ = produceRecordWithVersion(ctx, t, writeClient, partitionID, content, 1)
}

func produceRecordWithVersion(ctx context.Context, t *testing.T, writeClient *kgo.Client, partitionID int32, content []byte, version int) int64 {
	rec := &kgo.Record{
		Value:     content,
		Topic:     topicName,
		Partition: partitionID,
	}
	if version == 0 {
		rec.Headers = nil
	} else {
		rec.Headers = []kgo.RecordHeader{RecordVersionHeader(version)}
	}

	produceResult := writeClient.ProduceSync(ctx, rec)
	require.NoError(t, produceResult.FirstErr())

	return rec.Offset
}

func RecordVersionHeader(version int) kgo.RecordHeader {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(version))
	return kgo.RecordHeader{
		Key:   "Version",
		Value: b[:],
	}
}

func produceRecordAt(ctx context.Context, t *testing.T, writeClient *kgo.Client, partitionID int32, ts time.Time, content []byte) {
	rec := &kgo.Record{
		Value:     content,
		Topic:     topicName,
		Partition: partitionID,
		Timestamp: ts,
	}
	require.NoError(t, writeClient.ProduceSync(ctx, rec).FirstErr())
}

// TestPartitionOffsetClient_FetchPartitionsOffsetsAfterMilli verifies the
// timestamp lookup: it returns the first offset at/after the given millisecond
// timestamp, offset 0 when the timestamp precedes all records, and -1 when no
// record is at/after the timestamp.
func TestPartitionOffsetClient_FetchPartitionsOffsetsAfterMilli(t *testing.T) {
	ctx := context.Background()
	_, clusterAddr := testkafka.CreateCluster(t, 1, topicName)
	client := createTestKafkaClient(t, createTestKafkaConfig(clusterAddr))
	reader := NewPartitionOffsetClient(client, topicName)

	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	produceRecordAt(ctx, t, client, 0, base, []byte("r0"))
	produceRecordAt(ctx, t, client, 0, base.Add(10*time.Minute), []byte("r1"))
	produceRecordAt(ctx, t, client, 0, base.Add(20*time.Minute), []byte("r2"))

	// Before all records -> first offset (0).
	offsets, err := reader.FetchPartitionsOffsetsAfterMilli(ctx, base.Add(-time.Minute).UnixMilli(), []int32{0})
	require.NoError(t, err)
	assert.Equal(t, int64(0), getPartitionsOffsets(offsets)[0])

	// At/after the second record's timestamp -> offset 1.
	offsets, err = reader.FetchPartitionsOffsetsAfterMilli(ctx, base.Add(10*time.Minute).UnixMilli(), []int32{0})
	require.NoError(t, err)
	assert.Equal(t, int64(1), getPartitionsOffsets(offsets)[0])

	// After all records -> -1 (no record at/after the timestamp).
	offsets, err = reader.FetchPartitionsOffsetsAfterMilli(ctx, base.Add(time.Hour).UnixMilli(), []int32{0})
	require.NoError(t, err)
	assert.Equal(t, int64(-1), getPartitionsOffsets(offsets)[0])

	// A negative timestamp is clamped to 0 (earliest) rather than colliding with
	// Kafka's reserved ListOffsets values (-1 = end, -2 = start).
	offsets, err = reader.FetchPartitionsOffsetsAfterMilli(ctx, -1, []int32{0})
	require.NoError(t, err)
	assert.Equal(t, int64(0), getPartitionsOffsets(offsets)[0])
}
