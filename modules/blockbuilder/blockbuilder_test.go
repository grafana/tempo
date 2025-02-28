package blockbuilder

import (
	"context"
	"crypto/rand"
	"errors"
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

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	producedRecords := sendReq(t, ctx, client)

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
	require.ErrorContains(t, err, "No partitions assigned")
}

func TestBlockbuilder_getAssignedPartitions(t *testing.T) {
	ctx := context.Background()

	store := newStore(ctx, t)
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

	b, err := New(cfg, test.NewTestingLogger(t), partitionRing, &mockOverrides{}, store)
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

	k, address := testkafka.CreateCluster(t, 1, testTopic)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit, func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Inc()
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address, []int32{0})

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	producedRecords := sendTracesFor(t, ctx, client, 5*time.Second, 100*time.Millisecond) // Send for 5 seconds

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

	records := sendTracesFor(t, ctx, client, 5*time.Second, 100*time.Millisecond) // Send for 5 seconds
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

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	producedRecords := sendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond) // Send for 1 second, <1 consumption cycles

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

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	producedRecords := sendReq(t, ctx, client)

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

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	producedRecords := sendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond) // Send for 1 second, <1 consumption cycles

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

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)

	// Send a single record
	producedRecords := sendReq(t, ctx, client)
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
	newRecords := sendReq(t, ctx, client)
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
	return newStoreWithLogger(ctx, t, test.NewTestingLogger(t))
}

func newStoreWithLogger(ctx context.Context, t testing.TB, log log.Logger) storage.Store {
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
			BlocklistPoll: 5 * time.Second,
		},
	}, nil, log)
	require.NoError(t, err)

	s.EnablePolling(ctx, &ownEverythingSharder{})
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

func newKafkaClient(t testing.TB, config ingest.KafkaConfig) *kgo.Client {
	writeClient, err := kgo.NewClient(
		kgo.SeedBrokers(config.Address),
		kgo.AllowAutoTopicCreation(),
		kgo.DefaultProduceTopic(config.Topic),
		// We will choose the partition of each record.
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
	)
	require.NoError(t, err)
	t.Cleanup(writeClient.Close)

	return writeClient
}

func countFlushedTraces(store storage.Store) int {
	count := 0
	for _, meta := range store.BlockMetas(util.FakeTenantID) {
		count += int(meta.TotalObjects)
	}
	return count
}

// nolint: revive
func sendReq(t testing.TB, ctx context.Context, client *kgo.Client) []*kgo.Record {
	traceID := generateTraceID(t)

	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 10, traceID, startTime, endTime)
	records, err := ingest.Encode(0, util.FakeTenantID, req, 1_000_000)
	require.NoError(t, err)

	res := client.ProduceSync(ctx, records...)
	require.NoError(t, res.FirstErr())

	return records
}

// nolint: revive,unparam
func sendTracesFor(t *testing.T, ctx context.Context, client *kgo.Client, dur, interval time.Duration) []*kgo.Record {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timer := time.NewTimer(dur)
	defer timer.Stop()

	producedRecords := make([]*kgo.Record, 0)

	for {
		select {
		case <-ctx.Done(): // Exit the function if the context is done
			return producedRecords
		case <-timer.C: // Exit the function when the timer is done
			return producedRecords
		case <-ticker.C:
			records := sendReq(t, ctx, client)
			producedRecords = append(producedRecords, records...)
		}
	}
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
		store      = newStoreWithLogger(ctx, b, logger)
		cfg        = blockbuilderConfig(b, address, []int32{0})
		client     = newKafkaClient(b, cfg.IngestStorageConfig.Kafka)
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
			for _, r := range sendReq(b, ctx, client) {
				size += len(r.Value)
			}
		}
		b.StartTimer()

		_, err = bb.consume(ctx)
		require.NoError(b, err)

		b.SetBytes(int64(size))
	}
}
