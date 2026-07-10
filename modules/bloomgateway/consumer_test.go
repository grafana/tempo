package bloomgateway

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/goleak"

	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
)

// newTestKafkaConfig builds an ingest.KafkaConfig with every flag default
// applied (flagext.DefaultValues, the repo-wide convention for this --
// see modules/livestore/partition_reader_test.go), pointed at addr/topic.
func newTestKafkaConfig(addr, topic string) ingest.KafkaConfig {
	cfg := ingest.KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Address = addr
	cfg.Topic = topic
	return cfg
}

func TestConsumer_AssignsAllPartitionsAndConsumesFromStart(t *testing.T) {
	const topic = "bg-assign"
	const numPartitions = 4
	_, addr := testkafka.CreateCluster(t, numPartitions, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	recs := make([]*kgo.Record, numPartitions)
	for p := range recs {
		recs[p] = &kgo.Record{Topic: topic, Partition: int32(p), Value: []byte(fmt.Sprintf("payload-%d", p))}
	}
	res := produceClient.ProduceSync(ctx, recs...)
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, consumer.Start(ctx, nil))

	seen := map[int32]bool{}
	for len(seen) < numPartitions {
		select {
		case rec := <-consumer.Records():
			seen[rec.Partition] = true
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for all %d partitions; saw %v", numPartitions, seen)
		}
	}
	assert.Len(t, seen, numPartitions, "one AddConsumePartitions call must cover every partition")
}

func TestConsumer_ResumesFromGivenOffsets(t *testing.T) {
	const topic = "bg-resume"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	const total = 5
	recs := make([]*kgo.Record, total)
	for i := range recs {
		recs[i] = &kgo.Record{Topic: topic, Partition: 0, Value: []byte(fmt.Sprintf("v%d", i))}
	}
	res := produceClient.ProduceSync(ctx, recs...)
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	// Resume from offset 3: only offsets 3 and 4 should ever be delivered
	// -- resume is caller-authoritative, never re-derived from a broker
	// commit (§0 Kafka-plumbing decision).
	require.NoError(t, consumer.Start(ctx, map[int32]int64{0: 3}))

	var got []int64
	for len(got) < 2 {
		select {
		case rec := <-consumer.Records():
			got = append(got, rec.Offset)
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out; got %v", got)
		}
	}
	assert.Equal(t, []int64{3, 4}, got)
}

func TestConsumer_QueueBlocksFetchLoopWhenFullAndResumesAfterDrain(t *testing.T) {
	const topic = "bg-backpressure"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	const recordBytes = 40
	const numRecords = 5
	recs := make([]*kgo.Record, numRecords)
	for i := range recs {
		recs[i] = &kgo.Record{Topic: topic, Partition: 0, Value: make([]byte, recordBytes)}
	}
	res := produceClient.ProduceSync(ctx, recs...)
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	// Budget for exactly 2 records; a 3rd must block the fetch loop until
	// one is drained -- never silently dropped (§0 Kafka-plumbing
	// decision).
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 2*recordBytes, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, consumer.Start(ctx, nil))

	require.Eventually(t, func() bool { return len(consumer.Records()) >= 2 }, 5*time.Second, 10*time.Millisecond,
		"the first 2 records must fit the byte budget and be admitted")
	require.Never(t, func() bool { return len(consumer.Records()) > 2 }, 300*time.Millisecond, 10*time.Millisecond,
		"the fetch loop must block once the byte budget is exhausted, never race ahead or silently drop")

	drained := <-consumer.Records()
	if drained.release != nil {
		drained.release()
	}

	require.Eventually(t, func() bool { return len(consumer.Records()) >= 2 }, 5*time.Second, 10*time.Millisecond,
		"releasing one record's bytes must let the fetch loop admit the next one")
}

func TestConsumer_Rewind(t *testing.T) {
	const topic = "bg-rewind"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	const total = 5
	recs := make([]*kgo.Record, total)
	for i := range recs {
		recs[i] = &kgo.Record{Topic: topic, Partition: 0, Value: []byte(fmt.Sprintf("v%d", i))}
	}
	res := produceClient.ProduceSync(ctx, recs...)
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, consumer.Start(ctx, nil))

	drainAll := func() []int64 {
		var got []int64
		for len(got) < total {
			select {
			case rec := <-consumer.Records():
				got = append(got, rec.Offset)
			case <-time.After(10 * time.Second):
				t.Fatalf("timed out draining; got %v", got)
			}
		}
		return got
	}

	assert.Equal(t, []int64{0, 1, 2, 3, 4}, drainAll())

	require.NoError(t, consumer.Rewind(map[int32]int64{0: 0}))

	assert.Equal(t, []int64{0, 1, 2, 3, 4}, drainAll(), "rewind must re-deliver from the requested offset")
}

// TestConsumer_OffsetsAtOrBefore is the kfake spike AMENDMENT A6 calls for:
// it exercises the PRODUCTION kadm-based OffsetsAtOrBefore against a real
// (fake) broker rather than injecting a fake PositionRewinder, because
// kfake turns out to implement genuine timestamp-ordered binary search
// server-side (kfake/02_list_offsets.go's findBatchMeta), not just the
// -1/-2 sentinel offsets -- see the WP14 final report for what this
// confirmed kfake supports.
func TestConsumer_OffsetsAtOrBefore(t *testing.T) {
	const topic = "bg-offsets-at-or-before"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const n = 5
	// Produced one at a time, deliberately NOT as one batched ProduceSync
	// call: kfake's timestamp search refines its per-batch answer by
	// scanning that batch's OWN records, so multiple records sharing one
	// physical batch would let the search walk past the true crossing
	// point. One record per batch is what makes kfake's per-batch
	// findBatchMeta search (kfake/02_list_offsets.go) match real Kafka's
	// documented "first offset at or after timestamp" protocol semantics
	// for this test's per-record timestamp assertions.
	for i := 0; i < n; i++ {
		res := produceClient.ProduceSync(ctx, &kgo.Record{Topic: topic, Partition: 0, Value: []byte(fmt.Sprintf("v%d", i)), Timestamp: base.Add(time.Duration(i) * time.Minute)})
		require.NoError(t, res.FirstErr())
	}

	cfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	tt := []struct {
		name string
		at   time.Time
		want int64
	}{
		{"before every record floors at the start offset", base.Add(-time.Minute), 0},
		{"strictly between two records resolves to the earlier one", base.Add(2*time.Minute + 30*time.Second), 2},
		{"exactly at a record's timestamp still rounds down by one (conservative: never under-replay)", base.Add(3 * time.Minute), 2},
		{"after every record resolves to the last available record", base.Add(10 * time.Minute), n - 1},
	}
	for _, c := range tt {
		t.Run(c.name, func(t *testing.T) {
			got, err := consumer.OffsetsAtOrBefore(ctx, c.at)
			require.NoError(t, err)
			require.Contains(t, got, int32(0))
			assert.Equal(t, c.want, got[0])
		})
	}
}

// TestConsumer_CommitLoopCommitsForLagObservability validates the
// per-partition commit workaround (commitOffsets, consumer.go) against
// pkg/ingest/testkafka.Cluster's offsetCommit handler, which hard-asserts
// exactly one partition per OffsetCommit request (ingest-kafka report
// gotcha 4) -- a naive single batched multi-partition CommitAllOffsets call
// would trip that assertion the moment more than one partition is involved,
// which is exactly why this test uses 2 partitions rather than 1.
func TestConsumer_CommitLoopCommitsForLagObservability(t *testing.T) {
	const topic = "bg-commit-loop"
	const numPartitions = 2
	_, addr := testkafka.CreateCluster(t, numPartitions, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	ctx := context.Background()
	res := produceClient.ProduceSync(
		ctx,
		&kgo.Record{Topic: topic, Partition: 0, Value: []byte("p0")},
		&kgo.Record{Topic: topic, Partition: 1, Value: []byte("p1")},
	)
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	cfg.ConsumerGroupOffsetCommitInterval = 20 * time.Millisecond

	const instanceID = "bloom-gateway-7"
	consumer, err := NewConsumer(cfg, instanceID, 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, consumer.Start(ctx, nil))

	seen := map[int32]bool{}
	for len(seen) < numPartitions {
		select {
		case rec := <-consumer.Records():
			seen[rec.Partition] = true
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out; saw %v", seen)
		}
	}

	verifyClient := testkafka.NewKafkaClient(t, addr, topic)
	adm := kadm.NewClient(verifyClient)
	group := cfg.GetConsumerGroup(instanceID, 0)

	require.Eventually(t, func() bool {
		fetched, err := adm.FetchOffsets(ctx, group)
		if err != nil {
			return false
		}
		o0, ok0 := fetched.Lookup(topic, 0)
		o1, ok1 := fetched.Lookup(topic, 1)
		return ok0 && ok1 && o0.At == 1 && o1.At == 1
	}, 5*time.Second, 20*time.Millisecond, "both partitions' offsets must be committed under the per-instance group, never used for resume -- only for external lag tooling")
}

// TestConsumer_GoleakStartClose deliberately does NOT use
// testkafka.CreateCluster: that helper registers its own t.Cleanup(fake.
// Close), which only runs after this test function returns -- too late to
// observe with goleak.VerifyNone called from inside the test body. Owning
// the kfake.Cluster directly lets this test close it (and thus its
// listener/run goroutines) before checking for leaks, isolating the
// assertion to Consumer's own goroutines (fetch loop + commit loop).
func TestConsumer_GoleakStartClose(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "bg-goleak"
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic))
	require.NoError(t, err)
	addrs := fake.ListenAddrs()
	require.Len(t, addrs, 1)

	cfg := newTestKafkaConfig(addrs[0], topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	require.NoError(t, consumer.Start(context.Background(), nil))
	require.NoError(t, consumer.Close())
	fake.Close()

	goleak.VerifyNone(t, opts)
}
