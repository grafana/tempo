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
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"go.uber.org/atomic"
)

const testTopic = "test-topic"

func TestBlockbuilder(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(errors.New("test done")) })

	k, address := testkafka.CreateClusterWithoutCustomConsumerGroupsSupport(t, 1, "test-topic")
	t.Cleanup(k.Close)

	kafkaCommits := atomic.NewInt32(0)
	k.ControlKey(kmsg.OffsetCommit.Int16(), func(kmsg.Request) (kmsg.Response, error, bool) {
		kafkaCommits.Add(1)
		return nil, nil, false
	})

	store := newStore(t)
	store.EnablePolling(ctx, nil)
	cfg := blockbuilderConfig(t, address)

	b := New(cfg, test.NewTestingLogger(t), newPartitionRingReader(), &mockOverrides{}, store)
	require.NoError(t, services.StartAndAwaitRunning(ctx, b))
	t.Cleanup(func() {
		require.NoError(t, services.StopAndAwaitTerminated(ctx, b))
	})

	req := test.MakePushBytesRequest(t, 10, nil)
	records, err := ingest.Encode(0, util.FakeTenantID, req, 1_000_000)
	require.NoError(t, err)

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)

	res := client.ProduceSync(ctx, records...)
	require.NoError(t, res.FirstErr())

	// Wait for record to be consumed and committed.
	require.Eventually(t, func() bool {
		return kafkaCommits.Load() > 0
	}, time.Minute, time.Second)

	// Wait for the block to be flushed.
	require.Eventually(t, func() bool {
		return len(store.BlockMetas(util.FakeTenantID)) == 1
	}, time.Minute, time.Second)
}

// FIXME - Test is unstable and will fail if records cross two consumption cycles,
//
//	because it's asserting that there is exactly two commits, one of which fails.
//	It can be 3 commits if the records cross two consumption cycles.
//	🤷
func TestBlockbuilderDeterministicComsumption(t *testing.T) {
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

	store := newStore(t)
	store.EnablePolling(ctx, nil)
	cfg := blockbuilderConfig(t, address)
	logger := test.NewTestingLogger(t)

	client := newKafkaClient(t, cfg.IngestStorageConfig.Kafka)
	// Send for 1 second, <1 consumption cycles
	timer := time.NewTimer(1 * time.Second)
sendLoop:
	for {
		select {
		case <-timer.C:
			break sendLoop
		default:
			traceID := make([]byte, 16)
			_, err := rand.Read(traceID)
			require.NoError(t, err)

			req := test.MakePushBytesRequest(t, 10, traceID)
			records, err := ingest.Encode(0, util.FakeTenantID, req, 1_000_000)
			require.NoError(t, err)

			res := client.ProduceSync(ctx, records...)
			require.NoError(t, res.FirstErr())

			time.Sleep(100 * time.Millisecond)
		}
	}

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
		return len(store.BlockMetas(util.FakeTenantID)) == 1 // Only one block should be written
	}, time.Minute, time.Second)
}

func blockbuilderConfig(_ *testing.T, address string) Config {
	cfg := Config{}
	flagext.DefaultValues(&cfg)

	flagext.DefaultValues(&cfg.blockConfig)

	flagext.DefaultValues(&cfg.IngestStorageConfig.Kafka)
	cfg.IngestStorageConfig.Kafka.Address = address
	cfg.IngestStorageConfig.Kafka.Topic = testTopic
	cfg.IngestStorageConfig.Kafka.ConsumerGroup = "test-consumer-group"

	cfg.AssignedPartitions = []int32{0}
	cfg.LookbackOnNoCommit = 15 * time.Second
	cfg.ConsumeCycleDuration = 5 * time.Second

	return cfg
}

func newStore(t *testing.T) storage.Store {
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
			BlocklistPoll:         50 * time.Second,
			BlocklistPollFallback: true,
		},
	}, nil, test.NewTestingLogger(t))
	require.NoError(t, err)
	return s
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
