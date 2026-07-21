package bloomgateway

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/goleak"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// newScaledApplier is a local variant of events_test.go's newTestApplier,
// parameterized on d (rather than hardcoding the package's small testD),
// for tests whose entry count would make testD's 16 leaves pathologically
// slow: each leaf's sorted-array insert is O(current length) (leaf.go), so
// a few hundred thousand entries crammed into only 16 leaves is quadratic.
// Spreading them over more leaves keeps this file's large-chunk tests fast
// without touching events_test.go's own fixtures.
func newScaledApplier(t *testing.T, d uint8) (*Applier, *Directory, *Registry, *TenantSet) {
	t.Helper()
	dir := NewDirectory(d)
	for i := range uint32(1) << d {
		leaf, started := dir.BeginConstructing(i)
		require.True(t, started)
		require.NoError(t, dir.Complete(i, leaf))
	}
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	seed := HashSeed([]byte("bloom-gateway-worker-test-seed"))
	return NewApplier(dir, reg, tenants, seed, d, testF, m), dir, reg, tenants
}

// newLocalKfakeCluster constructs a kfake cluster directly (bypassing
// pkg/ingest/testkafka.CreateCluster, which offers no way to pass
// brokerConfigs) so this file's large-chunk test can raise
// message.max.bytes (ingest-kafka report gotcha 3: kfake's default
// ~1 MiB broker limit otherwise silently breaks a >=4 MiB produce with
// kerr.MessageTooLarge).
func newLocalKfakeCluster(t testing.TB, numPartitions int32, topic string, brokerConfigs map[string]string) string {
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
	return addrs[0]
}

// newLargeMessageProducer builds a producer client with its own
// client-side batch-size cap raised to match: franz-go's default
// ProducerBatchMaxBytes (~1,000,012 B) would otherwise reject a >=4 MiB
// record before it ever reaches the broker, independently of the
// broker-side message.max.bytes raised by newLocalKfakeCluster.
func newLargeMessageProducer(t testing.TB, addr, topic string, maxBytes int32) *kgo.Client {
	t.Helper()
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(addr),
		kgo.AllowAutoTopicCreation(),
		kgo.DefaultProduceTopic(topic),
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
		kgo.DisableClientMetrics(),
		kgo.ProducerBatchMaxBytes(maxBytes),
		kgo.MaxBufferedBytes(int(maxBytes)*2),
	)
	require.NoError(t, err)
	t.Cleanup(cl.Close)
	return cl
}

// addEvent wraps chunkFor (events_test.go) in a full envelope -- the
// test-only encode helper the plan calls for (no production producer
// exists yet by design: DESIGN.md's Kafka producer hooks are out of scope
// for this plan, §6). Every call site in this package builds a single-chunk
// block (chunk_index 0, chunk_count 1), so those arguments are hardcoded
// rather than carried as parameters that never vary.
func addEvent(uuid backend.UUID, tenant string, tr timeRange, ids [][]byte) *tempopb.BloomGatewayEvent {
	return &tempopb.BloomGatewayEvent{
		Version:  supportedEventVersion,
		Type:     tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK,
		AddChunk: chunkFor(uuid, tenant, tr, 0, 1, ids),
	}
}

func mustMarshalEvent(t testing.TB, event *tempopb.BloomGatewayEvent) []byte {
	t.Helper()
	b, err := event.Marshal()
	require.NoError(t, err)
	return b
}

// forEachPermutation calls fn once per permutation of items (Heap's
// algorithm), reusing one backing array -- used so the AppliedOffsets
// invariant below is checked against every possible completion-order
// interleaving, not a hand-picked sample, matching this repo's own
// TestApply_IdempotentUnderRedeliveryAndReorder precedent (events_test.go)
// for "every permutation" style regression tests. Per the harness's output
// discipline, permutations are GENERATED at runtime, never enumerated in
// source.
func forEachPermutation(items []int64, fn func([]int64)) {
	work := append([]int64(nil), items...)
	var generate func(k int)
	generate = func(k int) {
		if k == 1 {
			fn(work)
			return
		}
		for i := 0; i < k; i++ {
			generate(k - 1)
			if k%2 == 0 {
				work[i], work[k-1] = work[k-1], work[i]
			} else {
				work[0], work[k-1] = work[k-1], work[0]
			}
		}
	}
	generate(len(work))
}

// TestWorkerPool_AppliedOffsetsNeverSkipsIncompletePredecessor is the named
// deliverable for the WP14 AppliedOffsets invariant: for EVERY possible
// order in which a partition's completions could arrive (workers finish out
// of order under real concurrency; this test makes that exhaustive and
// deterministic instead of timing-dependent), AppliedOffsets must equal
// exactly the length of the longest completed PREFIX of offsets -- never
// skipping ahead over an offset that hasn't completed yet, and never
// lagging behind one that has. Dispatch order is simulated by seeding
// offset 0 first (mirroring WorkerPool.dispatchLoop, which always seeds
// before any completion can race in), matching how a real partition's
// records are delivered.
func TestWorkerPool_AppliedOffsetsNeverSkipsIncompletePredecessor(t *testing.T) {
	const partition = int32(0)
	const n = 6
	offsets := make([]int64, n)
	for i := range offsets {
		offsets[i] = int64(i)
	}

	forEachPermutation(offsets, func(order []int64) {
		p := &WorkerPool{progress: make(map[int32]*partitionProgress)}
		p.seed(partition, 0)

		applied := make(map[int64]bool, n)
		for _, off := range order {
			p.markApplied(partition, off)
			applied[off] = true

			want := int64(0)
			for applied[want] {
				want++
			}
			got := p.AppliedOffsets()[partition]
			require.Equalf(t, want, got, "completion order %v: after applying offset %d", order, off)
		}
	})
}

// TestWorkerPool_AppliedOffsetsConvergesWithRealOutOfOrderCompletion
// corroborates the exhaustive test above through the REAL Start/process
// pipeline (real goroutines, real Applier, -race-checked), rather than
// calling seed/markApplied directly. Offset 0 carries a large chunk
// (measurably slower to apply -- real hashing/inserting work, DESIGN.md's
// own cost model: apply cost scales with trace-ID count) dispatched
// immediately before several tiny chunks; with more than one worker, the
// tiny ones have every opportunity to finish first. This "injects" a
// per-offset delay via real, data-driven work rather than an artificial
// sleep hook in production code.
func TestWorkerPool_AppliedOffsetsConvergesWithRealOutOfOrderCompletion(t *testing.T) {
	applier, _, regy, _ := newTestApplier(t, true)
	m := newMetrics(prometheus.NewRegistry())
	records := make(chan Record, 8)
	pool := NewWorkerPool(4, records, applier, log.NewNopLogger(), m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Stop()

	tr := testTimeRange()
	bigIDs := make([][]byte, 150_000)
	for i := range bigIDs {
		bigIDs[i] = traceID(i)
	}
	records <- Record{Partition: 0, Offset: 0, Value: mustMarshalEvent(t, addEvent(testUUID(t, 0), "tenant-a", tr, bigIDs))}
	for i := 1; i <= 4; i++ {
		records <- Record{Partition: 0, Offset: int64(i), Value: mustMarshalEvent(t, addEvent(testUUID(t, i), "tenant-a", tr, traceIDs(i)))}
	}
	close(records)

	require.Eventually(t, func() bool {
		return pool.AppliedOffsets()[0] == 5
	}, 10*time.Second, 5*time.Millisecond, "all 5 offsets must eventually be reflected, regardless of completion order")

	for i := 0; i < 5; i++ {
		_, ok := regy.LookupUUID(testUUID(t, i))
		assert.True(t, ok, "block %d must have been applied", i)
	}
}

// TestWorkerPool_DropsAndCountsDecodeFailureWithoutStallingWatermark uses
// an unrecognized envelope version (events.go's DecodeEvent: "a non-nil
// error means drop and count") as a deterministic decode failure -- more
// reliable than hand-corrupted protobuf bytes, which can accidentally
// parse as a valid-looking (if garbage) message.
func TestWorkerPool_DropsAndCountsDecodeFailureWithoutStallingWatermark(t *testing.T) {
	applier, _, _, _ := newTestApplier(t, true)
	m := newMetrics(prometheus.NewRegistry())
	records := make(chan Record, 4)
	pool := NewWorkerPool(2, records, applier, log.NewNopLogger(), m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Stop()

	tr := testTimeRange()
	badVersion := &tempopb.BloomGatewayEvent{
		Version:  supportedEventVersion + 1,
		Type:     tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK,
		AddChunk: chunkFor(testUUID(t, 0), "tenant-a", tr, 0, 1, traceIDs(0)),
	}
	records <- Record{Partition: 0, Offset: 0, Value: mustMarshalEvent(t, badVersion)}
	records <- Record{Partition: 0, Offset: 1, Value: mustMarshalEvent(t, addEvent(testUUID(t, 500), "tenant-a", tr, traceIDs(1)))}
	close(records)

	require.Eventually(t, func() bool {
		return pool.AppliedOffsets()[0] == 2
	}, 5*time.Second, 5*time.Millisecond, "a decode failure must still advance the watermark (drop and count, never stall)")

	assert.Equal(t, float64(1), promtestCounter(t, m.addsTotal, "dropped"))
}

// TestWorkerPool_PauseBlocksNewApplyUntilResume is Pause/Resume's basic
// deliverable (bloomgateway.go's snapshot consistency mechanism, §0 D9):
// nothing dispatched after Pause is applied until Resume is called, and it
// applies promptly afterward.
func TestWorkerPool_PauseBlocksNewApplyUntilResume(t *testing.T) {
	applier, _, regy, _ := newTestApplier(t, true)
	m := newMetrics(prometheus.NewRegistry())
	records := make(chan Record, 8)
	pool := NewWorkerPool(2, records, applier, log.NewNopLogger(), m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Stop()

	tr := testTimeRange()
	records <- Record{Partition: 0, Offset: 0, Value: mustMarshalEvent(t, addEvent(testUUID(t, 0), "tenant-a", tr, traceIDs(0)))}
	require.Eventually(t, func() bool { return pool.AppliedOffsets()[0] == 1 }, 5*time.Second, 5*time.Millisecond,
		"sanity: the pool must be genuinely live before pausing it")

	pool.Pause()

	records <- Record{Partition: 0, Offset: 1, Value: mustMarshalEvent(t, addEvent(testUUID(t, 1), "tenant-a", tr, traceIDs(1)))}
	require.Never(t, func() bool {
		_, ok := regy.LookupUUID(testUUID(t, 1))
		return ok
	}, 300*time.Millisecond, 10*time.Millisecond, "no record may be applied while paused")

	pool.Resume()

	require.Eventually(t, func() bool {
		_, ok := regy.LookupUUID(testUUID(t, 1))
		return ok
	}, 5*time.Second, 5*time.Millisecond, "a record queued during the pause must be applied once Resume is called")
}

// TestWorkerPool_PauseWaitsForInFlightRecordToFinish is Pause's other
// deliverable: it must not return while ANY record is still mid-apply (the
// property Snapshotter.Save's own "already-quiesced read" precondition
// depends on, snapshot.go).
//
// This drives pauseMu directly (holding the RLock to simulate an in-flight
// record) rather than racing a real Kafka-shaped pipeline against a large,
// slow chunk. An earlier version of this test did exactly that (a single
// worker, a 150,000-ID chunk, and require.Eventually(len(records)==0, ...)
// as the "the worker must have started processing it" signal) and DEADLOCKED
// intermittently under heavy machine load: "the record left the external
// channel" does not imply "the worker has already called
// p.pauseMu.RLock()" -- there is a genuine, if normally tiny, scheduling gap
// between a worker receiving from the internal (unbuffered) staged channel
// and actually entering process(). When Pause()'s Lock() call won that race
// (fast enough scheduling delay on the newly-spawned pause goroutine, or a
// slow-to-be-scheduled worker), Go's own RWMutex fairness rule queues the
// worker's subsequent RLock() behind the writer -- so Pause() returned
// almost immediately (nothing was actually held yet), this test's own
// "returned too early" branch fired t.Fatal WITHOUT reaching the un-deferred
// pool.Resume() call below it, and pauseMu was left permanently write-locked
// -- wedging the deferred pool.Stop() (worker.go) forever waiting for a
// worker that can now never acquire its own RLock. Driving pauseMu directly
// removes the real-pipeline scheduling dependency entirely and is a more
// direct test of the actual mechanism (the same RWMutex process() itself
// uses) besides.
func TestWorkerPool_PauseWaitsForInFlightRecordToFinish(t *testing.T) {
	applier, _, _, _ := newTestApplier(t, true)
	m := newMetrics(prometheus.NewRegistry())
	pool := NewWorkerPool(1, make(chan Record), applier, log.NewNopLogger(), m) // never started: no real worker/dispatch goroutines needed for this test

	pool.pauseMu.RLock() // simulates one record's in-flight decode+apply (process's own critical section)

	pauseDone := make(chan struct{})
	go func() {
		pool.Pause()
		close(pauseDone)
	}()

	select {
	case <-pauseDone:
		t.Fatal("Pause returned while the (simulated) in-flight record's RLock was still held")
	case <-time.After(100 * time.Millisecond):
		// still held, as expected -- fall through
	}

	pool.pauseMu.RUnlock() // the "record" finishes

	require.Eventually(t, func() bool {
		select {
		case <-pauseDone:
			return true
		default:
			return false
		}
	}, 5*time.Second, 5*time.Millisecond, "Pause must return once the in-flight record's lock is released")

	pool.Resume()
}

func TestWorkerPool_GoleakStartStop(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	applier, _, _, _ := newTestApplier(t, true)
	m := newMetrics(prometheus.NewRegistry())
	records := make(chan Record)
	pool := NewWorkerPool(4, records, applier, log.NewNopLogger(), m)

	pool.Start(context.Background())
	close(records)
	pool.Stop()

	goleak.VerifyNone(t, opts)
}

// TestWorkerPool_EndToEndThroughKfakeQueueAndApplier is WP14's end-to-end
// deliverable: a hand-encoded tempopb.BloomGatewayEvent (there is no
// production producer yet, by design -- plan §6) flows through a real
// (fake) Kafka broker, Consumer's byte-bounded queue, WorkerPool, and into
// Applier, landing in the shared directory/registry/tenant structures. The
// chunk is sized to exceed 4 MiB so this also exercises the raised
// message.max.bytes / ProducerBatchMaxBytes gotcha end-to-end, not just in
// isolation.
func TestWorkerPool_EndToEndThroughKfakeQueueAndApplier(t *testing.T) {
	const topic = "bg-e2e"
	const maxBytes = 8 << 20
	addr := newLocalKfakeCluster(t, 4, topic, map[string]string{"message.max.bytes": "8388608"})

	cfg := newTestKafkaConfig(addr, topic)
	// AutoCreateTopicEnabled defaults to true, and NewConsumer's
	// ingest.NewReaderClient would then call
	// SetDefaultNumberOfPartitionsForAutocreatedTopics, which issues the
	// OLD, non-incremental AlterConfigs API (kfake's 33_alter_configs.go:
	// "Replaces all configs with the provided set") -- REPLACING the
	// entire broker config set with just {num.partitions}, silently
	// wiping out the raised message.max.bytes set above and reverting the
	// broker to its ~1 MiB default. Confirmed by direct inspection of
	// kfake's AlterConfigs handler (a real-Kafka-accurate behavior of that
	// deprecated API, not a kfake-only quirk). The topic is already seeded
	// (newLocalKfakeCluster), so auto-create isn't needed here anyway.
	cfg.AutoCreateTopicEnabled = false
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", int64(maxBytes), log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	// D=12 (4096 leaves) rather than the package's small testD -- see
	// newScaledApplier's doc comment: this test's entry count would make
	// testD's 16 leaves pathologically slow to insert into.
	applier, dir, regy, tenants := newScaledApplier(t, 12)
	m := newMetrics(prometheus.NewRegistry())
	pool := NewWorkerPool(4, consumer.Records(), applier, log.NewNopLogger(), m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := newLargeMessageProducer(t, addr, topic, maxBytes)
	uuid := testUUID(t, 900)
	tr := testTimeRange()
	const numIDs = 260_000 // comfortably clears 4 MiB once protobuf-encoded
	ids := make([][]byte, numIDs)
	for i := range ids {
		ids[i] = traceID(i)
	}
	payload := mustMarshalEvent(t, addEvent(uuid, "tenant-e2e", tr, ids))
	require.GreaterOrEqual(t, len(payload), 4<<20, "test payload must exceed 4 MiB to exercise the gotcha")

	res := producer.ProduceSync(ctx, &kgo.Record{Topic: topic, Partition: 0, Value: payload})
	require.NoError(t, res.FirstErr())

	require.NoError(t, consumer.Start(ctx, nil))
	pool.Start(ctx)
	t.Cleanup(pool.Stop)

	require.Eventually(t, func() bool {
		// A single 260k-trace-ID chunk takes real, if brief, wall-clock
		// time to hash and insert (DESIGN.md's own cost model), so the
		// block is visible in the registry as BlockPending for a window
		// before ApplyAddChunk's completing chunk transitions it to Live
		// -- poll for the terminal state, not mere registry presence.
		// Registry.State (not block.State off a retained *Block) is the
		// lock-safe accessor -- registry.go's own doc comment on Block
		// warns that reading State directly off a *Block concurrently
		// mutated by CommitLive is unsynchronized.
		state, ok := regy.State(uuid)
		return ok && state == BlockLive
	}, 30*time.Second, 20*time.Millisecond, "the large AddChunk must flow queue -> workers -> Applier into the registry and reach BlockLive")

	block, ok := regy.LookupUUID(uuid)
	require.True(t, ok)
	state, ok := regy.State(uuid)
	require.True(t, ok)
	assert.Equal(t, BlockLive, state)
	assert.True(t, handleInWindow(tenants, "tenant-e2e", block.Handle))

	leafIdx, fp := Address(traceID(0), applier.hashSeed, 12, testF)
	handles, ok := dir.Lookup(leafIdx, uint16(fp))
	require.True(t, ok)
	assert.Contains(t, handles, block.Handle)
}
