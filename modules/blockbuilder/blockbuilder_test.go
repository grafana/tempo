package blockbuilder

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

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
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.uber.org/atomic"
)

const testTopic = "test-topic"

// When the partition starts with no existing commit,
// the block-builder looks back to consume all available records from the start and ensures they are committed and flushed into a block.
func TestBlockbuilder_lookbackOnNoCommit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Add(1)
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address)

	b := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	sendReq(t, ctx, client)

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1
	}, time.Minute, time.Second)
}

// Starting with a pre-existing commit,
// the block-builder resumes from the last known position, consuming new records,
// and ensures all of them are properly committed and flushed into blocks.
func TestBlockbuilder_startWithCommit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Add(1)
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address)

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

	b := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
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
}

// In case a block flush initially fails, the system retries until it succeeds.
func TestBlockbuilder_flushingFails(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Add(1)
		return nil, nil, false
	})

	storageWrites := atomic.NewInt32(0)
	store := newStoreWrapper(newStore(ctx, t), func(ctx context.Context, block tempodb.WriteableBlock, store storage.Store) error {
		// Fail the first block write
		if storageWrites.Add(1) == 1 {
			return errors.New("failed to write block")
		}
		return store.WriteBlock(ctx, block)
	})
	cfg := blockbuilderConfig(t, address)
	logger := test.NewTestingLogger(t)

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	sendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond) // Send for 1 second, <1 consumption cycles

	b := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
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
}

// Receiving records with older timestamps the block-builder processes them in the current cycle,
// ensuring they're written into a new block despite "belonging" to another cycle.
func TestBlockbuilder_receivesOldRecords(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		k.KeepControl()
		kafkaCommits.Add(1)
		return nil, nil, false
	})

	store := newStore(ctx, t)
	cfg := blockbuilderConfig(t, address)

	b := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
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
		return kafkaCommits.Load() == 2
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 2
	}, time.Minute, time.Second)
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

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(req kmsg.Request) (kmsg.Response, error, bool) {
		k.KeepControl()
		defer kafkaCommits.Add(1)

		if kafkaCommits.Load() == 0 { // First commit fails
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
	cfg := blockbuilderConfig(t, address)
	logger := test.NewTestingLogger(t)

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	sendTracesFor(t, ctx, client, time.Second, 100*time.Millisecond) // Send for 1 second, <1 consumption cycles

	b := New(cfg, logger, newPartitionRingReader(), &mockOverrides{}, store)
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
}

func TestCycleEndAtStartup(t *testing.T) {
	now := time.Date(1995, 8, 26, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name                             string
		now, cycleEnd, cycleEndAtStartup time.Time
		waitDur, interval                time.Duration
	}{
		{
			name:              "now doesn't need to be truncated",
			now:               now,
			cycleEnd:          now.Add(5 * time.Minute),
			cycleEndAtStartup: now,
			waitDur:           5 * time.Minute,
			interval:          5 * time.Minute,
		},
		{
			name:              "now needs to be truncated",
			now:               now.Add(2 * time.Minute),
			cycleEnd:          now.Truncate(5 * time.Minute).Add(5 * time.Minute),
			cycleEndAtStartup: now.Truncate(5 * time.Minute),
			waitDur:           3 * time.Minute,
			interval:          5 * time.Minute,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cycleEndAtStartup := cycleEndAtStartup(tc.now, tc.interval)
			require.Equal(t, tc.cycleEndAtStartup, cycleEndAtStartup)

			cycleEnd, waitDur := nextCycleEnd(tc.now, tc.interval)
			require.Equal(t, tc.cycleEnd, cycleEnd)
			require.Equal(t, tc.waitDur, waitDur)
		})
	}
}

func TestNextCycleEnd(t *testing.T) {
	tests := []struct {
		name         string
		t            time.Time
		interval     time.Duration
		expectedTime time.Time
		expectedWait time.Duration
	}{
		{
			name:         "ExactInterval",
			t:            time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			interval:     time.Hour,
			expectedTime: time.Date(2023, 10, 1, 13, 0, 0, 0, time.UTC),
			expectedWait: time.Hour,
		},
		{
			name:         "PastInterval",
			t:            time.Date(2023, 10, 1, 12, 30, 0, 0, time.UTC),
			interval:     time.Hour,
			expectedTime: time.Date(2023, 10, 1, 13, 0, 0, 0, time.UTC),
			expectedWait: 30 * time.Minute,
		},
		{
			name:         "FutureInterval",
			t:            time.Date(2023, 10, 1, 12, 0, 0, 1, time.UTC),
			interval:     time.Hour,
			expectedTime: time.Date(2023, 10, 1, 13, 0, 0, 0, time.UTC),
			expectedWait: 59*time.Minute + 59*time.Second + 999999999*time.Nanosecond,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resultTime, resultWait := nextCycleEnd(tc.t, tc.interval)
			require.Equal(t, tc.expectedTime, resultTime)
			require.Equal(t, tc.expectedWait, resultWait)
		})
	}
}

func blockbuilderConfig(t *testing.T, address string) Config {
	cfg := Config{}
	flagext.DefaultValues(&cfg)

	flagext.DefaultValues(&cfg.BlockConfig)

	flagext.DefaultValues(&cfg.IngestStorageConfig.Kafka)
	cfg.IngestStorageConfig.Kafka.Address = address
	cfg.IngestStorageConfig.Kafka.Topic = testTopic
	cfg.IngestStorageConfig.Kafka.ConsumerGroup = "test-consumer-group"

	cfg.AssignedPartitions = map[string][]int32{cfg.InstanceID: {0}}
	cfg.LookbackOnNoCommit = 15 * time.Second
	cfg.ConsumeCycleDuration = 5 * time.Second

	cfg.WAL.Filepath = t.TempDir()

	return cfg
}

var _ blocklist.JobSharder = (*ownEverythingSharder)(nil)

type ownEverythingSharder struct{}

func (o *ownEverythingSharder) Owns(string) bool { return true }

func newStore(ctx context.Context, t *testing.T) storage.Store {
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
	}, nil, test.NewTestingLogger(t))
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

func (m *mockOverrides) DedicatedColumns(_ string) backend.DedicatedColumns { return m.dc }

func newKafkaClient(t *testing.T, config ingest.KafkaConfig) *kgo.Client {
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

func sendReq(t *testing.T, ctx context.Context, client *kgo.Client) []*kgo.Record {
	traceID := generateTraceID(t)

	req := test.MakePushBytesRequest(t, 10, traceID)
	records, err := ingest.Encode(0, util.FakeTenantID, req, 1_000_000)
	require.NoError(t, err)

	res := client.ProduceSync(ctx, records...)
	require.NoError(t, res.FirstErr())

	return records
}

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

func generateTraceID(t *testing.T) []byte {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)
	return traceID
}
