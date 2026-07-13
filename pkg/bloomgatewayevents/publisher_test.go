package bloomgatewayevents

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/goleak"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// newKfakeCluster constructs a kfake cluster directly rather than via
// pkg/ingest/testkafka.CreateCluster: that helper registers its own
// t.Cleanup(fake.Close), which only runs after the test function returns --
// too late to observe with a goleak.VerifyNone call made from inside the
// test body, per modules/bloomgateway/consumer_test.go's
// TestConsumer_GoleakStartClose (which explains the same constraint and
// takes the same approach). kfake.Cluster.Close is idempotent (guarded by an
// atomic "dead" flag), so registering this t.Cleanup as a safety net AND
// closing explicitly before a goleak check in every test below is safe
// belt-and-suspenders.
func newKfakeCluster(t testing.TB, numPartitions int32, topic string, brokerConfigs map[string]string) (*kfake.Cluster, string) {
	t.Helper()
	opts := []kfake.Opt{kfake.NumBrokers(1), kfake.SeedTopics(numPartitions, topic)}
	if len(brokerConfigs) > 0 {
		opts = append(opts, kfake.BrokerConfigs(brokerConfigs))
	}
	cluster, err := kfake.NewCluster(opts...)
	require.NoError(t, err)
	t.Cleanup(cluster.Close)
	addrs := cluster.ListenAddrs()
	require.Len(t, addrs, 1)
	return cluster, addrs[0]
}

// newTestConfig returns a Config wired to a test Kafka cluster at
// addr/topic, with Enabled forced on -- mirroring config_test.go's own
// validConfig helper's pattern of registering flags for real defaults
// rather than hand-building a zero-value Config that would silently drift
// from production defaults (linger, timeouts, etc.) as they evolve.
func newTestConfig(t testing.TB, addr, topic string, numPartitions int32) Config {
	t.Helper()
	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("test", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.Enabled = true
	cfg.Kafka.Address = addr
	cfg.Kafka.Topic = topic
	cfg.Kafka.AutoCreateTopicDefaultPartitions = int(numPartitions)
	// The topic is already seeded at exactly numPartitions
	// (newKfakeCluster's kfake.SeedTopics); auto-create must stay off, or
	// ingest.NewWriterClient's SetDefaultNumberOfPartitionsForAutocreatedTopics
	// call replaces the ENTIRE broker config set via kfake's old-style
	// AlterConfigs (a real-Kafka-accurate quirk of that deprecated API --
	// see modules/bloomgateway/worker_test.go's newLocalKfakeCluster
	// comment), silently wiping out any broker config (e.g. a raised
	// message.max.bytes) a test may also have set.
	cfg.Kafka.AutoCreateTopicEnabled = false
	return cfg
}

// pollRecords polls reader until at least want records have been fetched or
// timeout elapses, returning whatever was collected. Centralizes the
// PollFetches/EachRecord loop (mirrors
// modules/distributor/distributor_test.go's
// TestPushTracesSkipMetricsGenerationIngestStorage) so individual tests
// don't each hand-roll their own polling loop.
func pollRecords(t testing.TB, reader *kgo.Client, want int, timeout time.Duration) []*kgo.Record {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var got []*kgo.Record
	for len(got) < want && ctx.Err() == nil {
		fetches := reader.PollFetches(ctx)
		fetches.EachRecord(func(r *kgo.Record) { got = append(got, r) })
	}
	return got
}

// TestPublisher_Disabled_NoClient locks in New's central contract: a
// disabled Publisher never dials Kafka and every method is a safe no-op, so
// callers never need to branch on cfg.Enabled themselves. The configured
// address is TEST-NET-3 (RFC 5737, guaranteed non-routable) -- if New or any
// method below ever tried to dial it, goleak would catch the dangling
// connection goroutine even if the dial itself just hung.
func TestPublisher_Disabled_NoClient(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("test", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.Enabled = false
	cfg.Kafka.Address = "203.0.113.1:9092"

	reg := prometheus.NewRegistry()
	pub, err := New(cfg, log.NewNopLogger(), reg)
	require.NoError(t, err)
	assert.False(t, pub.Enabled())

	require.NotPanics(t, func() {
		pub.PublishAdd(context.Background(), backend.NewUUID(), "tenant", time.Now(), time.Now(), [][]byte{{1, 2, 3, 4}})
		pub.PublishDelete(context.Background(), backend.NewUUID(), "tenant")
		pub.Close()
		pub.Close()
	})

	// A disabled Publisher must leave zero footprint on reg -- this is what
	// makes it safe for more than one disabled Publisher (e.g. block-builder
	// and backend-worker, or this package's own multi-construction tests) to
	// exist against the same prometheus.DefaultRegisterer without a
	// duplicate-registration panic.
	count, err := testutil.GatherAndCount(reg, "tempo_bloom_gateway_publishes_total")
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	goleak.VerifyNone(t, opts)
}

// TestPublisher_MultipleDisabledInstancesShareRegisterer_NoPanic is the
// direct regression test for the root cause New's disabled-path metrics
// change fixes: production code (block-builder, backend-worker) each
// construct their own Publisher against the same prometheus.DefaultRegisterer,
// and Enabled defaults to false, so this exact shape -- more than one
// disabled Publisher against one shared Registerer -- is the common case,
// not an edge case.
func TestPublisher_MultipleDisabledInstancesShareRegisterer_NoPanic(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("test", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.Enabled = false

	reg := prometheus.NewRegistry()
	var err1, err2 error
	require.NotPanics(t, func() {
		_, err1 = New(cfg, log.NewNopLogger(), reg)
		_, err2 = New(cfg, log.NewNopLogger(), reg)
	})
	require.NoError(t, err1)
	require.NoError(t, err2)

	goleak.VerifyNone(t, opts)
}

// TestPublisher_New_InvalidConfigRejected proves New actually calls
// cfg.Validate() -- nothing else validates this config -- rather than
// silently accepting a broken one.
func TestPublisher_New_InvalidConfigRejected(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	var cfg Config
	cfg.RegisterFlagsAndApplyDefaults("test", flag.NewFlagSet("test", flag.ContinueOnError))
	cfg.Enabled = true
	cfg.ChunkSize = 0

	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.Error(t, err)
	assert.Nil(t, pub)

	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_VersionAndShape publishes a real AddChunk through
// a fake Kafka broker and checks the wire shape byte-for-byte: version,
// type, block/tenant/time round-trip, and -- the case padding bugs hide in
// -- trace IDs shorter than 16 bytes surviving with no padding at all.
func TestPublisher_PublishAdd_VersionAndShape(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-shape"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	blockID := backend.NewUUID()
	start := time.Now().Add(-time.Hour).Truncate(time.Nanosecond)
	end := time.Now().Truncate(time.Nanosecond)
	id4 := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	id8 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	ids := [][]byte{id4, id8}

	pub.PublishAdd(context.Background(), blockID, "tenant-shape", start, end, ids)

	recs := pollRecords(t, reader, 1, 5*time.Second)
	require.Len(t, recs, 1)

	event := &tempopb.BloomGatewayEvent{}
	require.NoError(t, event.Unmarshal(recs[0].Value))
	assert.EqualValues(t, 1, event.Version)
	assert.Equal(t, tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK, event.Type)
	require.NotNil(t, event.AddChunk)
	assert.Equal(t, blockID.String(), event.AddChunk.BlockId)
	assert.Equal(t, "tenant-shape", event.AddChunk.TenantId)
	assert.Equal(t, start.UnixNano(), event.AddChunk.StartTimeUnixNano)
	assert.Equal(t, end.UnixNano(), event.AddChunk.EndTimeUnixNano)
	assert.EqualValues(t, 0, event.AddChunk.ChunkIndex)
	assert.EqualValues(t, 1, event.AddChunk.ChunkCount)
	assert.Equal(t, ids, event.AddChunk.TraceIds, "IDs must round-trip byte-identical, including short ones -- no padding on the producer side")

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_MultiChunk locks in chunking end-to-end: 7 IDs
// with ChunkSize=3 must arrive as exactly 3 records, indices 0/1/2, all
// sharing the same ChunkCount, whose trace IDs partition the original set
// with no overlap or loss.
func TestPublisher_PublishAdd_MultiChunk(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-multichunk"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	cfg.ChunkSize = 3
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	ids := idsOfLen(7, 8)
	blockID := backend.NewUUID()
	pub.PublishAdd(context.Background(), blockID, "tenant-mc", time.Now(), time.Now(), ids)

	recs := pollRecords(t, reader, 3, 5*time.Second)
	require.Len(t, recs, 3)

	var indices []uint32
	var gotIDs [][]byte
	for _, r := range recs {
		event := &tempopb.BloomGatewayEvent{}
		require.NoError(t, event.Unmarshal(r.Value))
		require.NotNil(t, event.AddChunk)
		assert.EqualValues(t, 3, event.AddChunk.ChunkCount)
		indices = append(indices, event.AddChunk.ChunkIndex)
		gotIDs = append(gotIDs, event.AddChunk.TraceIds...)
	}
	assert.ElementsMatch(t, []uint32{0, 1, 2}, indices)
	assert.ElementsMatch(t, ids, gotIDs, "chunking must not add, drop, reorder, or duplicate any ID")

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_DedupesAndFilters exercises PublishAdd's
// defensive filter and dedupe together: a duplicate plus a 0-byte and a
// 17-byte ID must all disappear from what's published, while the invalid
// ones (and only those) are counted -- never silently dropped without a
// trace, and never allowed to poison the valid IDs sharing their chunk.
func TestPublisher_PublishAdd_DedupesAndFilters(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-dedupe"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	valid1 := []byte{1, 2, 3, 4}
	valid2 := []byte{5, 6, 7, 8}
	dup := valid1
	tooShort := []byte{}
	tooLong := make([]byte, 17)

	ids := [][]byte{valid1, valid2, dup, tooShort, tooLong}
	blockID := backend.NewUUID()
	pub.PublishAdd(context.Background(), blockID, "tenant-dedup", time.Now(), time.Now(), ids)

	recs := pollRecords(t, reader, 1, 5*time.Second)
	require.Len(t, recs, 1)

	event := &tempopb.BloomGatewayEvent{}
	require.NoError(t, event.Unmarshal(recs[0].Value))
	require.NotNil(t, event.AddChunk)
	assert.ElementsMatch(t, [][]byte{valid1, valid2}, event.AddChunk.TraceIds)

	assert.Equal(t, float64(2), testutil.ToFloat64(pub.metrics.invalidTraceIDsTotal))

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_PartitionMatchesHash locks in that PublishAdd
// actually uses partitionForBlock's hash to route the record, not some
// other (e.g. client-chosen) partition.
func TestPublisher_PublishAdd_PartitionMatchesHash(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-partition"
	const numPartitions = int32(16)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	blockID := backend.NewUUID()
	pub.PublishAdd(context.Background(), blockID, "tenant-part", time.Now(), time.Now(), [][]byte{{9, 9, 9, 9}})

	recs := pollRecords(t, reader, 1, 5*time.Second)
	require.Len(t, recs, 1)

	want := partitionForBlock(blockID, numPartitions)
	assert.Equal(t, want, recs[0].Partition)

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_UnreachableBroker_DropsSilently models
// modules/distributor/distributor_test.go's
// TestPushTracesKafkaWriteErrorReturnsRetryableStatus: a broker that's gone
// before the first produce is attempted. Unlike the distributor, PublishAdd
// must never surface the error -- it has to return promptly (bounded by
// cfg.Kafka.WriteTimeout, not hang) and only count the failure.
func TestPublisher_PublishAdd_UnreachableBroker_DropsSilently(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-unreachable"
	cluster, addr := newKfakeCluster(t, 4, topic, nil)
	cluster.Close() // gone before Publisher is even constructed

	cfg := newTestConfig(t, addr, topic, 4)
	cfg.Kafka.WriteTimeout = time.Second
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	started := time.Now()
	pub.PublishAdd(context.Background(), backend.NewUUID(), "tenant-unreachable", time.Now(), time.Now(), [][]byte{{1, 2, 3, 4}})
	elapsed := time.Since(started)

	assert.Less(t, elapsed, 8*time.Second, "PublishAdd must return within the bounded retry budget, not hang")
	assert.Equal(t, float64(1), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultDropped)))
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)))

	pub.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_EmptyIDs_NoOp locks in that an empty input never
// reaches the client at all: no record built, no counter moved.
func TestPublisher_PublishAdd_EmptyIDs_NoOp(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-empty"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	pub.PublishAdd(context.Background(), backend.NewUUID(), "tenant-empty", time.Now(), time.Now(), nil)

	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)))
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultDropped)))
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.invalidTraceIDsTotal))

	pub.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishDelete_Shape checks Delete's wire shape and confirms
// a block's Delete lands on the same partition as its own Adds did.
func TestPublisher_PublishDelete_Shape(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-delete"
	const numPartitions = int32(8)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	blockID := backend.NewUUID()
	pub.PublishAdd(context.Background(), blockID, "tenant-del", time.Now(), time.Now(), [][]byte{{1, 1, 1, 1}})
	pub.PublishDelete(context.Background(), blockID, "tenant-del")

	recs := pollRecords(t, reader, 2, 5*time.Second)
	require.Len(t, recs, 2)

	var addPartition, deletePartition int32
	var sawAdd, sawDelete bool
	for _, r := range recs {
		event := &tempopb.BloomGatewayEvent{}
		require.NoError(t, event.Unmarshal(r.Value))
		switch event.Type {
		case tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK:
			sawAdd = true
			addPartition = r.Partition
		case tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_DELETE:
			sawDelete = true
			deletePartition = r.Partition
			assert.EqualValues(t, 1, event.Version)
			require.NotNil(t, event.Delete)
			assert.Equal(t, blockID.String(), event.Delete.BlockId)
		}
	}
	require.True(t, sawAdd, "must have seen the AddChunk record")
	require.True(t, sawDelete, "must have seen the Delete record")
	assert.Equal(t, addPartition, deletePartition, "a block's Delete must land on the same partition as its Adds")

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_LargeChunk_AcceptedByRaisedBrokerLimit replicates (rather
// than imports -- newLocalKfakeCluster/newLargeMessageProducer are
// unexported helpers in modules/bloomgateway) the large-payload pattern from
// modules/bloomgateway/worker_test.go:42-81: kfake's default broker
// message.max.bytes (~1 MiB) would otherwise reject this chunk with
// kerr.MessageTooLarge before it's ever written, so it's raised directly via
// kfake.BrokerConfigs. Unlike that test's raw producer, this Publisher
// already goes through ingest.NewWriterClient, whose own
// ProducerBatchMaxBytes(16_000_000) comfortably covers this ~3.2 MiB chunk
// client-side -- only the broker-side limit needs raising here.
func TestPublisher_LargeChunk_AcceptedByRaisedBrokerLimit(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-large"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, map[string]string{"message.max.bytes": "8388608"})

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	const numIDs = 200_000 // == default ChunkSize: one chunk, ~3.2 MiB of trace IDs alone
	ids := idsOfLen(numIDs, 16)

	blockID := backend.NewUUID()
	pub.PublishAdd(context.Background(), blockID, "tenant-large", time.Now(), time.Now(), ids)

	recs := pollRecords(t, reader, 1, 15*time.Second)
	require.Len(t, recs, 1)

	event := &tempopb.BloomGatewayEvent{}
	require.NoError(t, event.Unmarshal(recs[0].Value))
	require.NotNil(t, event.AddChunk)
	assert.EqualValues(t, 1, event.AddChunk.ChunkCount)
	require.Len(t, event.AddChunk.TraceIds, numIDs)
	assert.Equal(t, ids, event.AddChunk.TraceIds, "the large chunk must be delivered byte-for-byte intact")

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_Close_Idempotent locks in Close's documented idempotency:
// safe to call more than once, and leaves nothing running afterward.
func TestPublisher_Close_Idempotent(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-close"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	require.NotPanics(t, func() {
		pub.Close()
		pub.Close()
	})

	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_PublishAdd_RateLimited_NothingProduced proves rate limiting
// is checked before any chunking work and is all-or-nothing per block
// (DESIGN.md § Multi-tenant cells): with a rate of 1/s, the first PublishAdd
// for a tenant consumes its only token and is fully delivered; an immediate
// second PublishAdd for the SAME tenant must produce zero records for its
// own block -- never a partial chunk set -- while counting exactly once as
// rate_limited.
func TestPublisher_PublishAdd_RateLimited_NothingProduced(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-ratelimited"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	cfg.ChunkSize = 2 // forces multiple chunks per block, so a partial delivery would be visible if it happened
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry(), WithTenantLimits(func(string) float64 { return 1 }))
	require.NoError(t, err)

	firstBlock := backend.NewUUID()
	pub.PublishAdd(context.Background(), firstBlock, "tenant-rl", time.Now(), time.Now(), idsOfLen(6, 8)) // 3 chunks at ChunkSize=2

	secondBlock := backend.NewUUID()
	pub.PublishAdd(context.Background(), secondBlock, "tenant-rl", time.Now(), time.Now(), idsOfLen(6, 8))

	// Exactly 3 records can ever exist (the first block's chunks): the
	// second PublishAdd returned having never called ProduceSync, so there
	// is nothing more for a longer poll to find.
	recs := pollRecords(t, reader, 3, 5*time.Second)
	require.Len(t, recs, 3, "only the first (unthrottled) block's chunks must be produced")

	for _, r := range recs {
		event := &tempopb.BloomGatewayEvent{}
		require.NoError(t, event.Unmarshal(r.Value))
		require.NotNil(t, event.AddChunk)
		assert.Equal(t, firstBlock.String(), event.AddChunk.BlockId, "no chunk from the rate-limited second block must ever appear on the wire")
	}

	assert.Equal(t, float64(1), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)), "exactly the first block's publish must count as ok")
	assert.Equal(t, float64(1), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultRateLimited)), "exactly the second block's publish must count as rate_limited")
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultDropped)))

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestPublisher_NoLimits_BehaviorUnchanged proves that constructing a
// Publisher without WithTenantLimits -- every call site and every other
// test in this file -- never throttles: N rapid-fire PublishAdd calls for
// the same tenant must all be delivered and counted ok.
func TestPublisher_NoLimits_BehaviorUnchanged(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "pub-nolimits"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry()) // no WithTenantLimits
	require.NoError(t, err)

	const n = 20
	for i := 0; i < n; i++ {
		pub.PublishAdd(context.Background(), backend.NewUUID(), "tenant-unlimited", time.Now(), time.Now(), [][]byte{{byte(i), 1, 1, 1}})
	}

	recs := pollRecords(t, reader, n, 5*time.Second)
	require.Len(t, recs, n)

	assert.Equal(t, float64(n), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)))
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultRateLimited)))

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}
