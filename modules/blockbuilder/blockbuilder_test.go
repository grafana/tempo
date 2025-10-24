package blockbuilder

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/storage"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.uber.org/atomic"
)

const (
	testTopic         = "test-topic"
	testConsumerGroup = "test-consumer-group"
	testPartition     = int32(0)
)

// When the partition starts with no existing commit,
// the block-builder looks back to consume all available records from the start and ensures they are committed and flushed into a block.
func TestBlockbuilder_lookbackOnNoCommit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})

	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1 && countFlushedTraces(store) == 1
	}, time.Minute, time.Second)

	// Check committed offset
	requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
}

func TestBlockbuilder_without_partitions_assigned_returns_an_error(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{})

	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	_, err = b.consume(ctx)
	require.ErrorIs(t, err, errNoPartitionsAssigned)
}

func TestBlockbuilder_getAssignedPartitions(t *testing.T) {
	cfg := blockbuilderConfig(t, "localhost", []int32{0, 2, 4, 6})
	partitionRing := newPartitionRingReaderWithPartitions(map[int32]ring.PartitionDesc{
		0:  {Id: 0, State: ring.PartitionActive},
		1:  {Id: 1, State: ring.PartitionActive},
		2:  {Id: 2, State: ring.PartitionInactive},
		3:  {Id: 3, State: ring.PartitionActive},
		4:  {Id: 4, State: ring.PartitionPending},
		5:  {Id: 5, State: ring.PartitionDeleted},
		20: {Id: 20, State: ring.PartitionActive},
	})

	b, err := New(cfg, test.NewTestingLogger(t), partitionRing, &mockOverrides{}, nil)
	require.NoError(t, err)
	partitions := b.getAssignedPartitions()
	assert.Equal(t, []int32{0, 2}, partitions)
}

// Starting with a pre-existing commit,
// the block-builder resumes from the last known position, consuming new records,
// and ensures all of them are properly committed and flushed into blocks.
func TestBlockbuilder_startWithCommit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 100, testTopic)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendTracesFor(t, ctx, client, 5*time.Second, 100*time.Millisecond, ingest.Encode) // Send for 5 seconds

	commitedAt := len(producedRecords) / 2
	// Commit half of the records
	offsets := make(kadm.Offsets)
	offsets.Add(kadm.Offset{
		Topic:     testTopic,
		Partition: 0,
		At:        producedRecords[commitedAt].Offset,
	})
	admClient := kadm.NewClient(client)
	require.NoError(t, admClient.CommitAllOffsets(ctx, cfg.IngestStorageConfig.Kafka.ConsumerGroup, offsets))

	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	records := testkafka.SendTracesFor(t, ctx, client, 5*time.Second, 100*time.Millisecond, ingest.Encode) // Send for 5 seconds
	producedRecords = append(producedRecords, records...)

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return countFlushedTraces(store) == len(producedRecords)-commitedAt
	}, time.Minute, time.Second)

	// Check committed offset
	requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
}

// In case a block flush initially fails, the system retries until it succeeds.
func TestBlockbuilder_flushingFails(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, "test-topic")

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	storageWrites := atomic.NewInt32(0)
	store := newStoreWrapper(newStore(ctx, t), func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error {
		// Fail the first block write
		if storageWrites.Inc() == 1 {
			return errors.New("failed to write block")
		}
		return store.WriteBlock(ctx, block)
	})
	cfg := blockbuilderConfig(t, address, []int32{0})
	logger := test.NewTestingLogger(t)

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond, ingest.Encode) // Send for 1 second, <1 consumption cycles

	b, err := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool { return kafkaCommits.Load() >= 1 }, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) >= 1
	}, time.Minute, time.Second)

	// Check committed offset
	requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
}

// Receiving records with older timestamps the block-builder processes them in the current cycle,
// ensuring they're written into a new block despite "belonging" to another cycle.
func TestBlockbuilder_receivesOldRecords(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, "test-topic")

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})

	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() == 1
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1
	}, time.Minute, time.Second)

	// Re-send the same records with an older timestamp
	// They should be processed in the next cycle and written to a new block regardless of the timestamp
	for _, record := range producedRecords {
		record.Timestamp = record.Timestamp.Add(-time.Hour)
	}
	res := client.ProduceSync(ctx, producedRecords...)
	require.NoError(t, res.FirstErr())

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		l := kafkaCommits.Load()
		return l == 2
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 2
	}, time.Minute, time.Second)

	// Check committed offset
	requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
}

// FIXME - Test is unstable and will fail if records cross two consumption cycles,
//
//	because it's asserting that there is exactly two commits, one of which fails.
//	It can be 3 commits if the records cross two consumption cycles.
//
// On encountering a commit failure, the block-builder retries the operation and eventually succeeds.
//
// This would cause two blocks to be written, one for each cycle (one cycle fails at commit, the other succeeds).
// The block-builder deterministically generates the block ID based on the cycle end timestamp,
// so the block ID for the failed cycle is the same from the block ID for the successful cycle,
// and the failed block is overwritten by the successful one.
func TestBlockbuilder_committingFails(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, "test-topic")

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(req kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()

		if kafkaCommits.Load() == 1 { // First commit fails
			res := kmsg.NewOffsetCommitResponse()
			res.Version = req.GetVersion()
			res.Topics = []kmsg.OffsetCommitResponseTopic{
				{
					Topic: testTopic,
					Partitions: []kmsg.OffsetCommitResponseTopicPartition{
						{
							Partition: 0,
							ErrorCode: kerr.RebalanceInProgress.Code,
						},
					},
				},
			}
			return &res, nil, true
		}

		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})
	logger := test.NewTestingLogger(t)

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond, ingest.Encode) // Send for 1 second, <1 consumption cycle

	b, err := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() == 2 // First commit fails, second commit succeeds
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1 // Only one block should have been written
	}, time.Minute, time.Second)

	// Check committed offset
	requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
}

func TestBlockbuilder_retries_on_retriable_commit_error(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, "test-topic")

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(req kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()

		if kafkaCommits.Load() == 1 {
			res := kmsg.NewOffsetCommitResponse()
			res.Version = req.GetVersion()
			res.Topics = []kmsg.OffsetCommitResponseTopic{
				{
					Topic: testTopic,
					Partitions: []kmsg.OffsetCommitResponseTopicPartition{
						{
							Partition: 0,
							ErrorCode: kerr.NotEnoughReplicas.Code, // Retryable error code
						},
					},
				},
			}
			return &res, nil, true
		}

		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})
	logger := test.NewTestingLogger(t)

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)
	lastRecordOffset := producedRecords[len(producedRecords)-1].Offset

	b, err := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)

	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() == 2
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1 // Only one block should have been written
	}, time.Minute, time.Second)

	requireLastCommitEquals(t, ctx, client, lastRecordOffset+1)
}

func TestBlockbuilder_retries_on_commit_error(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, "test-topic")

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(req kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()

		if kafkaCommits.Load() == 1 {
			res := kmsg.NewOffsetCommitResponse()
			res.Version = req.GetVersion()
			res.Topics = []kmsg.OffsetCommitResponseTopic{
				{
					Topic: testTopic,
					Partitions: []kmsg.OffsetCommitResponseTopicPartition{
						{
							Partition: 0,
						},
					},
				},
			}
			return &res, fmt.Errorf("error committing offset"), true
		}

		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})
	logger := test.NewTestingLogger(t)

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)
	lastRecordOffset := producedRecords[len(producedRecords)-1].Offset

	b, err := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)

	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() == 2
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1 // Only one block should have been written
	}, time.Minute, time.Second)

	requireLastCommitEquals(t, ctx, client, lastRecordOffset+1)
}

// TestBlockbuilder_noDoubleConsumption verifies that records are not consumed twice when there are no more records in the partition.
// This test ensures that the BlockBuilder correctly commits the offset as lastRec.Offset + 1 instead of just lastRec.Offset.
func TestBlockbuilder_noDoubleConsumption(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	// Track commits
	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(_ kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})
	// Set a shorter consume cycle duration
	cfg.ConsumeCycleDuration = 500 * time.Millisecond

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)

	// Send a single record
	producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)
	lastRecordOffset := producedRecords[len(producedRecords)-1].Offset

	// Create the block builder
	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// Wait for the record to be consumed and committed
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, 30*time.Second, time.Second)

	// Check that the offset was committed correctly (lastRec.Offset + 1)
	requireLastCommitEquals(t, ctx, client, lastRecordOffset+1)

	// Send another record
	newRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)
	newRecordOffset := newRecords[len(newRecords)-1].Offset

	// Wait for the new record to be consumed and committed
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 1
	}, 30*time.Second, time.Second)

	// Verify that the new offset was committed correctly
	requireLastCommitEquals(t, ctx, client, newRecordOffset+1)

	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 2
	}, 30*time.Second, time.Second)

	// Verify the total number of traces is correct (1 from each batch)
	require.Equal(t, 2, countFlushedTraces(store))
}

func TestBlockBuilder_honor_maxBytesPerCycle(t *testing.T) {
	cases := []struct {
		name             string
		maxBytesPerCycle int
		expectedCommits  int32
		expectedWrites   int32
	}{
		{
			name:             "Limited to 1 bytes per cycle",
			maxBytesPerCycle: 1,
			expectedCommits:  2,
			expectedWrites:   2,
		},
		{
			name:             "Limited to 100_000 bytes per cycle",
			maxBytesPerCycle: 100_000,
			expectedCommits:  1,
			expectedWrites:   1,
		},
		{
			name:             "Unlimited bytes per cycle",
			maxBytesPerCycle: 0,
			expectedCommits:  1,
			expectedWrites:   1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(errors.New("test done")) })

			k, address := testkafka.CreateCluster(t, 1, "test-topic")

			kafkaCommits := atomic.NewInt32(0)
			k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
				kafkaCommits.Inc()
				return nil, nil, false
			})

			storageWrites := atomic.NewInt32(0)
			store := newStoreWrapper(newStore(ctx, t), func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error {
				storageWrites.Inc()
				return store.WriteBlock(ctx, block)
			})

			cfg := blockbuilderConfig(t, address, []int32{0})
			cfg.MaxBytesPerCycle = uint64(tc.maxBytesPerCycle)

			b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
			require.NoError(t, err)
			require.NoError(t, services.StartAndAwaitRunning(ctx, b))
			t.Cleanup(func() {
				require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
			})

			client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
			// We send two records with a size less than 30KB
			testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)
			producedRecords := testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)

			require.Eventually(t, func() bool {
				return kafkaCommits.Load() == tc.expectedCommits
			}, time.Minute, time.Second)

			require.Eventually(t, func() bool {
				return storageWrites.Load() == tc.expectedWrites
			}, 30*time.Second, time.Second)

			requireLastCommitEquals(t, ctx, client, producedRecords[len(producedRecords)-1].Offset+1)
		})
	}
}

func TestBlockbuilder_usesRecordTimestampForBlockStartAndEnd(t *testing.T) {
	// default ingestion slack is 2 minutes. create some convenient times to help the test below
	now := time.Unix(1000000, 0)
	oneMinuteAgo := now.Add(-time.Minute)
	oneMinuteLater := now.Add(time.Minute)
	twoMinutesAgo := now.Add(-2 * time.Minute)
	threeMinutesAgo := now.Add(-3 * time.Minute)

	tcs := []struct {
		name          string
		startTime     time.Time
		endTime       time.Time
		recordTime    time.Time
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{ // records where the timestamp exactly matches the span timings
			name:          "exact match",
			startTime:     oneMinuteAgo,
			endTime:       oneMinuteAgo,
			recordTime:    oneMinuteAgo,
			expectedStart: oneMinuteAgo,
			expectedEnd:   oneMinuteAgo,
		},
		{ // records where the timestamp doesn't match the span timings, but within the ingestion slack
			name:          "within ingestion slack",
			startTime:     oneMinuteAgo,
			endTime:       oneMinuteLater,
			recordTime:    now,
			expectedStart: oneMinuteAgo,
			expectedEnd:   oneMinuteLater,
		},
		{ // records where the timestamp doesn't match the span timings and is outside the ingestion slack
			name:          "outside ingestion slack",
			startTime:     threeMinutesAgo,
			endTime:       now,
			recordTime:    now,
			expectedStart: twoMinutesAgo, // default ingestion slack is 2 minutes
			expectedEnd:   now,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(errors.New("test done")) })

			k, address := testkafka.CreateCluster(t, 1, testTopic)

			kafkaCommits := atomic.NewInt32(0)
			k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
				kafkaCommits.Inc()
				return nil, nil, false
			})

			store := newStore(ctx, t)
			cfg := blockbuilderConfig(t, address, []int32{0})

			client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)

			// Create a trace with specific start/end times
			traceID := generateTraceID(t)
			startTimeNano := uint64(tc.startTime.UnixNano())
			endTimeNano := uint64(tc.endTime.UnixNano())
			req := test.MakePushBytesRequest(t, 1, traceID, startTimeNano, endTimeNano)
			records, err := ingest.Encode(0, util.FakeTenantID, req, 1_000_000)
			require.NoError(t, err)

			// Set the record timestamp
			for _, record := range records {
				record.Timestamp = tc.recordTime
			}

			// Send the record
			res := client.ProduceSync(ctx, records...)
			require.NoError(t, res.FirstErr())

			b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
			require.NoError(t, err)
			require.NoError(t, services.StartAndAwaitRunning(ctx, b))
			t.Cleanup(func() {
				require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
			})

			// Wait for record to be consumed and committed
			require.Eventually(t, func() bool {
				return kafkaCommits.Load() > 0
			}, time.Minute, time.Second)

			// Wait for the block to be flushed
			require.Eventually(t, func() bool {
				return len(store.BlockMetas(util.FakeTenantID)) == 1
			}, time.Minute, time.Second)

			// Verify block timestamps
			metas := store.BlockMetas(util.FakeTenantID)
			require.Len(t, metas, 1)
			meta := metas[0]
			require.Equal(t, tc.expectedStart.Unix(), meta.StartTime.Unix())
			require.Equal(t, tc.expectedEnd.Unix(), meta.EndTime.Unix())
		})
	}
}

func TestBlockbuilder_marksOldBlocksCompacted(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	// Track commits
	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(_ kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	var (
		cfg    = blockbuilderConfig(t, address, []int32{0})
		client = testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	)

	// Send data for each tenant
	var (
		goodTenantID    = "1"
		badTenantID     = "2"
		producedRecords []*kgo.Record
	)
	producedRecords = append(producedRecords, testkafka.SendReq(ctx, t, client, ingest.Encode, goodTenantID)...)
	producedRecords = append(producedRecords, testkafka.SendReq(ctx, t, client, ingest.Encode, badTenantID)...)
	lastRecordOffset := producedRecords[len(producedRecords)-1].Offset

	// Simulate failures on the first cycle
	badWrites := atomic.NewInt32(0)
	goodWrites := atomic.NewInt32(0)
	goodBlockIDs := []backend.UUID{}
	store := newStoreWrapper(newStore(ctx, t), func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error {
		switch block.BlockMeta().TenantID {
		case badTenantID:
			// First flush on tenant 2 fails
			if badWrites.Inc() == 1 {
				// Wait until flush on good tenant is complete
				// and then return the error. Tenants are flushed in parallel
				// so this is required to ensure we get a block flushed on the first cycle.
				require.Eventually(t, func() bool {
					return goodWrites.Load() > 0
				}, 30*time.Second, 50*time.Millisecond)
				return errors.New("failed to write block")
			}
		case goodTenantID:
			// Save all blocks flushed for good tenant
			defer goodWrites.Inc()
			goodBlockIDs = append(goodBlockIDs, block.BlockMeta().BlockID)
		}
		return store.WriteBlock(ctx, block)
	})
	t.Cleanup(func() { require.NoError(t, services.StopAndAwaitTerminated(context.Background(), store)) })

	// Create the block builder
	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(context.Background(), b))
		cancel(errors.New("test cleanup"))
		store.Shutdown()
	})

	// Wait for the records to be consumed and committed
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, 30*time.Second, 50*time.Millisecond)

	// Check that the offset was committed correctly (lastRec.Offset + 1)
	requireLastCommitEquals(t, ctx, client, lastRecordOffset+1)

	// Repoll flushed blocks.
	store.PollNow(ctx)

	// Verify that each tenant only has 1 active block
	require.Equal(t, 1, len(store.BlockMetas(goodTenantID)))
	require.Equal(t, 1, len(store.BlockMetas(badTenantID)))

	// Verify the good tenant flushed 2 attempts
	require.Equal(t, 2, len(goodBlockIDs))

	// Verify the first block is compacted
	m, cm, err := store.BlockMeta(ctx, goodTenantID, goodBlockIDs[0])
	require.NoError(t, err)
	require.Nil(t, m)
	require.NotNil(t, cm)

	// Verify the second block is not compacted
	m, cm, err = store.BlockMeta(ctx, goodTenantID, goodBlockIDs[1])
	require.NoError(t, err)
	require.NotNil(t, m)
	require.Nil(t, cm)
}

func TestBlockbuilder_gracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	chConsumeStarted := make(chan struct{})
	chConsumeDone := make(chan struct{})
	chStopDone := make(chan struct{})

	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		close(chConsumeDone)
		return nil, nil, false
	})

	k.ControlKey(kmsg.OffsetFetch, func(kmsg.Request) (kmsg.Response, error, bool) {
		close(chConsumeStarted)
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0}) // Fix: Properly specify partition

	// Send initial traces to ensure the partition has records
	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	testkafka.SendReq(ctx, t, client, ingest.Encode, util.FakeTenantID)

	// Start sending traces in the background
	go func() {
		testkafka.SendTracesFor(t, ctx, client, 60*time.Second, time.Second, ingest.Encode)
	}()

	b, err := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, err)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))

	// Wait for cycle to be ongoing
	select {
	case <-chConsumeStarted:
	case <-time.After(cfg.ConsumeCycleDuration * 2):
		t.Fatal("Consume cycle didn't start")
	}

	go func() {
		defer close(chStopDone)
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	}()

	// Check if shutdown waits for consume cycle to finish
	select {
	case <-chConsumeDone:
	case <-chStopDone:
		t.Fatal("Shutdown completed before consume cycle finished - not graceful!")
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for consume cycle to finish")
	}

	// Now wait for the shutdown to complete after we've verified it's graceful
	select {
	case <-chStopDone:
		// Good, shutdown completed after we checked
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown didn't complete after consume cycle finished")
	}

	// Verify blocks were written
	require.Eventually(t, func() bool { return len(store.BlockMetas(util.FakeTenantID)) > 0 }, 30*time.Second, time.Second)
}

func blockbuilderConfig(t testing.TB, address string, assignedPartitions []int32) Config {
	cfg := Config{}
	flagext.DefaultValues(&cfg)

	flagext.DefaultValues(&cfg.BlockConfig)

	flagext.DefaultValues(&cfg.IngestStorageConfig.Kafka)
	cfg.IngestStorageConfig.Kafka.Address = address
	cfg.IngestStorageConfig.Kafka.Topic = testTopic
	cfg.IngestStorageConfig.Kafka.ConsumerGroup = testConsumerGroup
	cfg.AssignedPartitions = map[string][]int32{cfg.InstanceID: assignedPartitions}

	cfg.ConsumeCycleDuration = 5 * time.Second

	cfg.WAL.Filepath = t.TempDir()

	return cfg
}

var _ blocklist.JobSharder = (*ownEverythingSharder)(nil)

type ownEverythingSharder struct{}

func (o *ownEverythingSharder) Owns(string) bool { return true }

func newStore(ctx context.Context, t testing.TB) storage.Store {
	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t), false)
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger, skipNoCompactBlocks bool) storage.Store {
	tmpDir := t.TempDir()

	s, err := storage.NewStore(storage.Config{
		Trace: tempodb.Config{
			Backend: backend.Local,
			Local: &local.Config{
				Path: tmpDir,
			},
			Block: &common.BlockConfig{
				IndexDownsampleBytes: 2,
				BloomFP:              0.01,
				BloomShardSizeBytes:  100_000,
				Version:              encoding.LatestEncoding().Version(),
				Encoding:             backend.EncLZ4_1M,
				IndexPageSizeBytes:   1000,
			},
			WAL: &wal.Config{
				Filepath: tmpDir,
			},
			BlocklistPoll: 100 * time.Millisecond,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownEverythingSharder{}, skipNoCompactBlocks)
	return s
}

var _ storage.Store = (*storeWrapper)(nil)

type storeWrapper struct {
	storage.Store
	writeBlock func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error
}

func newStoreWrapper(s storage.Store, writeBlock func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error) *storeWrapper {
	return &storeWrapper{
		Store:      s,
		writeBlock: writeBlock,
	}
}

func (m *storeWrapper) WriteBlock(ctx context.Context, block tempodb.WriteableBlock) error {
	if m.writeBlock != nil {
		return m.writeBlock(ctx, block, m.Store)
	}
	return m.Store.WriteBlock(ctx, block)
}

var _ ring.PartitionRingReader = (*mockPartitionRingReader)(nil)

func newPartitionRingReader() *mockPartitionRingReader {
	return &mockPartitionRingReader{
		r: ring.NewPartitionRing(ring.PartitionRingDesc{
			Partitions: map[int32]ring.PartitionDesc{
				0: {State: ring.PartitionActive},
			},
		}),
	}
}

func newPartitionRingReaderWithPartitions(partitions map[int32]ring.PartitionDesc) *mockPartitionRingReader {
	return &mockPartitionRingReader{
		r: ring.NewPartitionRing(ring.PartitionRingDesc{
			Partitions: partitions,
		}),
	}
}

type mockPartitionRingReader struct {
	r *ring.PartitionRing
}

func (m *mockPartitionRingReader) PartitionRing() *ring.PartitionRing {
	return m.r
}

var _ Overrides = (*mockOverrides)(nil)

type mockOverrides struct {
	dc backend.DedicatedColumns
}

func (m *mockOverrides) MaxBytesPerTrace(_ string) int                      { return 0 }
func (m *mockOverrides) DedicatedColumns(_ string) backend.DedicatedColumns { return m.dc }

func countFlushedTraces(store storage.Store) int {
	count := 0
	for _, meta := range store.BlockMetas(util.FakeTenantID) {
		count += int(meta.TotalObjects)
	}
	return count
}

func generateTraceID(t testing.TB) []byte {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)
	return traceID
}

// nolint: revive
func requireLastCommitEquals(t testing.TB, ctx context.Context, client *kgo.Client, expectedOffset int64) {
	offsets, err := kadm.NewClient(client).FetchOffsetsForTopics(ctx, testConsumerGroup, testTopic)
	require.NoError(t, err)
	offset, ok := offsets.Lookup(testTopic, testPartition)
	require.True(t, ok)
	require.Equal(t, expectedOffset, offset.At)
}

func BenchmarkBlockBuilder(b *testing.B) {
	var (
		ctx        = context.Background()
		logger     = log.NewNopLogger()
		_, address = testkafka.CreateCluster(b, 1, testTopic)
		store      = newStoreWithLogger(ctx, b, logger, false)
		cfg        = blockbuilderConfig(b, address, []int32{0})
		client     = testkafka.NewKafkaClient(b, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
		o          = &mockOverrides{
			dc: backend.DedicatedColumns{
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res0", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res1", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res2", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res3", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res4", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res5", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res6", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res7", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res8", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeResource, Name: "res9", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span0", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span1", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span2", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span3", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span4", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span5", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span6", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span7", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span8", Type: backend.DedicatedColumnTypeString},
				backend.DedicatedColumn{Scope: backend.DedicatedColumnScopeSpan, Name: "span9", Type: backend.DedicatedColumnTypeString},
			},
		}
	)

	cfg.ConsumeCycleDuration = 1 * time.Hour

	bb, err := New(cfg, logger, newPartitionRingReader(), o, store)
	require.NoError(b, err)
	defer func() { require.NoError(b, bb.stopping(nil)) }()

	// Startup (without starting the background consume cycle)
	err = bb.starting(ctx)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		// Send more data
		b.StopTimer()
		size := 0
		for i := 0; i < 1000; i++ {
			for _, r := range testkafka.SendReq(ctx, b, client, ingest.Encode, util.FakeTenantID) {
				size += len(r.Value)
			}
		}
		b.StartTimer()

		_, err = bb.consume(ctx)
		require.NoError(b, err)

		b.SetBytes(int64(size))
	}
}

type slowStore struct {
	storage.Store
	wait chan struct{}
}

func (s *slowStore) WriteBlock(ctx context.Context, block tempodb.WriteableBlock) error {
	s.wait <- struct{}{} // send a signal to a goroutine
	<-s.wait             // wait for the signal from the goroutine
	return s.Store.WriteBlock(ctx, block)
}

// lock waits for the signal from WriteBlock locking the operation
func (s *slowStore) lock() {
	<-s.wait
}

// unlock sends a signal to WriteBlock unlocking the operation
func (s *slowStore) unlock() {
	s.wait <- struct{}{}
}

// TestBlockbuilder_twoPartitions_secondEmpty verifies correct handling of two Kafka
// partitions where the second partition is initially empty and receives data later.
// It uses a channel-gated store to step consumption, injects records between consume
// cycles (sleeping longer than ConsumeCycleDuration), and asserts that:
// - both partitions are assigned,
// - three blocks are flushed (p0 initial, p0 later, p1 later), and
// - committed offsets equal number of sent records.
// The test highly coupled with the blockbuilder implementation. In case it stuck
// due to channel, it is advised to debug consume cycle step by step.
func TestBlockbuilder_twoPartitions_secondEmpty(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })
	reqTime := time.Now().Add(-1 * time.Minute) // to be sure it won't be filtered out by cycle duration check

	// Create a Kafka cluster with 2 partitions
	_, address := testkafka.CreateCluster(t, 2, testTopic)

	// Setup block-builder
	ch := make(chan struct{})
	store := &slowStore{Store: newStore(ctx, t), wait: ch}
	cfg := blockbuilderConfig(t, address, []int32{0, 1})
	cfg.ConsumeCycleDuration = time.Second
	partitionRing := newPartitionRingReaderWithPartitions(map[int32]ring.PartitionDesc{
		0: {Id: 0, State: ring.PartitionActive},
		1: {Id: 1, State: ring.PartitionActive},
	})

	client := testkafka.NewKafkaClient(t, cfg.IngestStorageConfig.Kafka.Address, cfg.IngestStorageConfig.Kafka.Topic)
	// First, produce to partition 0
	testkafka.SendReqWithOpts(ctx, t, client, ingest.Encode, testkafka.ReqOpts{Partition: 0, Time: reqTime, TenantID: util.FakeTenantID})

	// And only then create block builder
	b, err := New(cfg, test.NewTestingLogger(t), partitionRing, &mockOverrides{}, store)
	require.NoError(t, err)

	// Verify builder is listening to both partitions
	parts := b.getAssignedPartitions()
	require.ElementsMatch(t, []int32{0, 1}, parts)

	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	// after initial consumption, add more records
	store.lock()
	testkafka.SendReqWithOpts(ctx, t, client, ingest.Encode, testkafka.ReqOpts{Partition: 0, Time: reqTime, TenantID: util.FakeTenantID})
	store.unlock()

	// after processing the first partition, add more records to the second partition
	store.lock()
	testkafka.SendReqWithOpts(ctx, t, client, ingest.Encode, testkafka.ReqOpts{Partition: 1, Time: reqTime, TenantID: util.FakeTenantID})
	store.unlock()

	// wait for the second partition to finish
	store.lock()
	store.unlock()

	// Wait for the block to be flushed (one block per each consumePartition call)
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 3 && countFlushedTraces(store) == 3
	}, 20*time.Second, time.Second)

	// Verify offsets
	offsets, err := kadm.NewClient(client).FetchOffsetsForTopics(ctx, testConsumerGroup, testTopic)
	require.NoError(t, err)
	for partition, expectedOffset := range map[int32]int64{
		0: 2,
		1: 1,
	} {
		offset, ok := offsets.Lookup(testTopic, partition)
		require.True(t, ok, "partition %d should have a committed offset", partition)
		require.Equal(t, expectedOffset, offset.At)
	}
}
