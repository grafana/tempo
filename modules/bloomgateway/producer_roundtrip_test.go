// This file is WP7's named end-to-end deliverable: a REAL
// pkg/bloomgatewayevents.Publisher (not a hand-encoded event, unlike every
// other test in this package predating the producer hooks landing) publishes
// through a real (fake-broker) Kafka topic into this package's own real
// Consumer/WorkerPool/Applier pipeline, and the result is observed through
// the real query Server -- proving the write path this package's other
// tests have always synthesized by hand actually round-trips end-to-end now
// that a production producer exists.
package bloomgateway

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/bloomgatewayevents"
	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/pkg/tempopb"
)

// roundTripFixture bundles one full producer -> gateway pipeline: a real
// bloomgatewayevents.Publisher and this package's real Directory/Registry/
// TenantSet/Applier/Consumer/WorkerPool/Server, all pointed at the same
// kfake broker/topic and sharing one hashSeed -- so entries the pipeline
// inserts on the write side and the leaf/fingerprint addresses the Server
// computes on the read side always agree.
type roundTripFixture struct {
	publisher *bloomgatewayevents.Publisher
	reg       *Registry
	tenants   *TenantSet
	srv       *Server
	hashSeed  uint64
}

// newRoundTripFixture wires the fixture and leaves it running: the consumer
// and worker pool are already started (consuming from offset 0 across every
// partition) by the time this returns, matching how the gateway runs in
// production -- publishes happen against an already-live pipeline, not the
// other way around.
func newRoundTripFixture(t *testing.T, topic string, numPartitions int32, chunkSize int) roundTripFixture {
	t.Helper()
	_, addr := testkafka.CreateCluster(t, numPartitions, topic)

	dir := NewDirectory(testD)
	for i := range uint32(1) << testD {
		leaf, started := dir.BeginConstructing(i)
		require.True(t, started)
		require.NoError(t, dir.Complete(i, leaf))
	}
	reg := NewRegistry()
	tenants := NewTenantSet()
	m := newMetrics(prometheus.NewRegistry())
	// Per-topic so two fixtures constructed within the same test run never
	// share a hashSeed by accident -- each still derives one seed shared
	// between its own Applier and Server, which is the property that
	// actually matters (NewServer's own contract: it derives hashSeed from
	// the raw seed itself, so the raw seed -- not the derived value -- is
	// what must be passed here).
	seed := []byte("bloom-gateway-roundtrip-" + topic + "-seed")
	hashSeed := HashSeed(seed)
	applier := NewApplier(dir, reg, tenants, hashSeed, testD, testF, m)
	srv := NewServer(dir, reg, tenants, seed, testD, testF, m)

	kafkaCfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(kafkaCfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	pool := NewWorkerPool(2, consumer.Records(), applier, log.NewNopLogger(), m)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)
	// One combined teardown, in the same order this pipeline's pieces
	// should stop in: cancel first (signals everything), then the pool
	// (drains in-flight work), then the consumer (closes the Kafka client)
	// -- mirroring worker_test.go's own
	// TestWorkerPool_EndToEndThroughKfakeQueueAndApplier teardown order.
	t.Cleanup(func() {
		cancel()
		pool.Stop()
		_ = consumer.Close()
	})

	require.NoError(t, consumer.Start(ctx, nil))

	var pubCfg bloomgatewayevents.Config
	pubCfg.RegisterFlagsAndApplyDefaults("test", flag.NewFlagSet("test", flag.ContinueOnError))
	pubCfg.Enabled = true
	pubCfg.ChunkSize = chunkSize
	pubCfg.Kafka.Address = addr
	pubCfg.Kafka.Topic = topic
	pubCfg.Kafka.AutoCreateTopicDefaultPartitions = int(numPartitions)
	// The topic is already seeded at exactly numPartitions
	// (testkafka.CreateCluster); auto-create must stay off, or
	// ingest.NewWriterClient's SetDefaultNumberOfPartitionsForAutocreatedTopics
	// call would replace the broker's entire config set via kfake's
	// old-style AlterConfigs (mirrors
	// pkg/bloomgatewayevents/publisher_test.go's own newTestConfig helper,
	// unexported there so replicated here rather than imported).
	pubCfg.Kafka.AutoCreateTopicEnabled = false
	publisher, err := bloomgatewayevents.New(pubCfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(publisher.Close)

	return roundTripFixture{publisher: publisher, reg: reg, tenants: tenants, srv: srv, hashSeed: hashSeed}
}

// freeLeafTraceID returns a trace ID that addresses a leaf none of avoid's
// entries do -- so a query for it is guaranteed matched=empty at that leaf,
// exercising the reject-all path deterministically instead of relying on a
// hand-picked ID that merely happens not to collide.
func freeLeafTraceID(t *testing.T, hashSeed uint64, avoid [][]byte) []byte {
	t.Helper()
	touched := make(map[uint32]bool, len(avoid))
	for _, id := range avoid {
		leafIdx, _ := Address(id, hashSeed, testD, testF)
		touched[leafIdx] = true
	}
	for i := uint32(0); i < uint32(1)<<testD; i++ {
		if !touched[i] {
			return findTraceIDForLeaf(t, hashSeed, testD, testF, i)
		}
	}
	t.Fatalf("test setup: all %d leaves were touched by %d IDs -- pick fewer IDs or a larger testD", uint32(1)<<testD, len(avoid))
	return nil
}

// TestProducerRoundTrip_AddThenQuery publishes one real, multi-chunk block
// through bloomgatewayevents.Publisher and waits for it to reach BlockLive
// via the real Kafka -> Consumer -> WorkerPool -> Applier pipeline, then
// confirms the query path treats it exactly like the hand-encoded fixtures
// every other test in this package uses: every one of the block's own trace
// IDs comes back as a candidate (never rejected), and an unrelated trace ID
// rejects it (reject-all, DESIGN.md § Query path).
func TestProducerRoundTrip_AddThenQuery(t *testing.T) {
	const topic = "bg-producer-roundtrip-add"
	const numPartitions = int32(2) // must equal the publisher's AutoCreateTopicDefaultPartitions below, or partitionForBlock can compute a partition the topic doesn't have
	const chunkSize = 3

	f := newRoundTripFixture(t, topic, numPartitions, chunkSize)

	const tenantID = "tenant-roundtrip-add"
	blockID := testUUID(t, 1)
	tr := testTimeRange()
	// 6 canonical 16-byte IDs plus one 8-byte ID: PublishAdd applies no
	// padding on the wire (pkg/bloomgatewayevents/publisher_test.go's
	// TestPublisher_PublishAdd_VersionAndShape), so this proves the short ID
	// survives the producer -> Kafka -> consumer hop byte-for-byte and is
	// still addressed correctly once Address pads it internally on the
	// apply side -- the padding contract end-to-end, not just in isolation.
	ids := traceIDs(1, 2, 3, 4, 5, 6)
	shortID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	ids = append(ids, shortID)
	require.Len(t, ids, 7)

	// chunkSize=3 over 7 IDs => ceil(7/3) = 3 chunks: a genuinely multi-chunk
	// block, not the single-chunk case every hand-encoded fixture elsewhere
	// in this package uses.
	f.publisher.PublishAdd(context.Background(), blockID, tenantID, tr.start, tr.end, ids)

	require.Eventually(t, func() bool {
		state, ok := f.reg.State(blockID)
		return ok && state == BlockLive
	}, 30*time.Second, 20*time.Millisecond, "the published block must flow producer -> kafka -> consumer -> workerpool -> applier and commit Live")

	for _, id := range ids {
		resp, err := f.srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: id})
		require.NoError(t, err)
		rejected := decodeRejected(t, resp.Rejected)
		assert.NotContains(t, rejected, [16]byte(blockID), "a published trace ID's own block must never be rejected -- it is a genuine candidate")
	}

	unrelated := freeLeafTraceID(t, f.hashSeed, ids)
	resp, err := f.srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: unrelated})
	require.NoError(t, err)
	rejected := decodeRejected(t, resp.Rejected)
	assert.Contains(t, rejected, [16]byte(blockID), "an unrelated trace ID must reject the only Live block in the tenant's window")
}

// TestProducerRoundTrip_DeleteRemovesBlock publishes a real Delete through
// the same producer -> gateway pipeline and confirms the query-time
// consequence.
//
// Expected semantics, verified against events.go and query.go (and already
// locked in for the hand-encoded case by events_test.go's
// TestApply_DeleteIsTerminal_NoResurrection): ApplyDelete's RemoveBlock
// takes the deleted block's handle OUT of the tenant's A_T window entirely
// -- it does not leave it in the window to be "rejected". query.go's Query
// computes rejected := window - candidates, and rejected is therefore
// always a SUBSET of window: a handle absent from window can be neither a
// candidate nor ever appear in rejected. A deleted block must vanish from a
// query's response exactly as if it had never existed, which is the only
// shape consistent with STATE.md's one invariant ("a block may be rejected
// only if it is live in the registry AND in A_T" -- a deleted block is
// neither). This is a materially different assertion from "the deleted
// block ends up in the rejection set", which the module's actual behavior
// contradicts.
//
// To make that "vanished, not merely rejected" distinction observable (and
// not vacuously true only because the tenant's window happened to become
// empty), a second, still-Live sibling block is published for the same
// tenant/window and must remain correctly rejected throughout.
func TestProducerRoundTrip_DeleteRemovesBlock(t *testing.T) {
	const topic = "bg-producer-roundtrip-delete"
	const numPartitions = int32(2)
	const chunkSize = 3

	f := newRoundTripFixture(t, topic, numPartitions, chunkSize)

	const tenantID = "tenant-roundtrip-delete"
	tr := testTimeRange()
	deletedID := testUUID(t, 1)
	deletedIDs := traceIDs(101, 102, 103)
	siblingID := testUUID(t, 2)
	siblingIDs := traceIDs(201, 202, 203)

	f.publisher.PublishAdd(context.Background(), deletedID, tenantID, tr.start, tr.end, deletedIDs)
	f.publisher.PublishAdd(context.Background(), siblingID, tenantID, tr.start, tr.end, siblingIDs)

	require.Eventually(t, func() bool {
		s1, ok1 := f.reg.State(deletedID)
		s2, ok2 := f.reg.State(siblingID)
		return ok1 && s1 == BlockLive && ok2 && s2 == BlockLive
	}, 30*time.Second, 20*time.Millisecond, "both blocks must commit Live before the delete is published")

	// Sanity, mirroring TestProducerRoundTrip_AddThenQuery: before the
	// delete, deletedID is a genuine candidate for its own trace IDs.
	resp, err := f.srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: deletedIDs[0]})
	require.NoError(t, err)
	before := decodeRejected(t, resp.Rejected)
	require.NotContains(t, before, [16]byte(deletedID), "test setup: deletedID must be a genuine candidate before the delete")

	f.publisher.PublishDelete(context.Background(), deletedID, tenantID)

	require.Eventually(t, func() bool {
		state, ok := f.reg.State(deletedID)
		return ok && state == BlockDeleted
	}, 30*time.Second, 20*time.Millisecond, "the delete must flow producer -> kafka -> consumer -> workerpool -> applier")
	assert.False(t, handleInWindow(f.tenants, tenantID, mustHandle(t, f.reg, deletedID)),
		"ApplyDelete's RemoveBlock must take the deleted block out of A_T entirely, not merely leave it there to be rejected")

	// deletedID's own trace ID must never come back rejected: its handle is
	// absent from window, and rejected is always a subset of window.
	resp, err = f.srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: deletedIDs[0]})
	require.NoError(t, err)
	afterOwnID := decodeRejected(t, resp.Rejected)
	assert.NotContains(t, afterOwnID, [16]byte(deletedID), "a deleted block must never appear in a query's rejected set -- it is no longer a candidate, but it is not wrongly rejected either: it is simply gone from A_T")

	// An unrelated trace ID's reject-all set must now be exactly the
	// sibling -- proving the window shrank by precisely the deleted
	// block's handle, not emptied wholesale and not left stale.
	unrelated := freeLeafTraceID(t, f.hashSeed, append(append([][]byte{}, deletedIDs...), siblingIDs...))
	resp, err = f.srv.Query(context.Background(), &tempopb.BloomGatewayQueryRequest{TenantId: tenantID, TraceId: unrelated})
	require.NoError(t, err)
	afterUnrelated := decodeRejected(t, resp.Rejected)
	assert.NotContains(t, afterUnrelated, [16]byte(deletedID), "the deleted block must not be rejected via the reject-all path either")
	assert.Contains(t, afterUnrelated, [16]byte(siblingID), "the still-Live sibling must remain correctly rejected -- the delete must not disturb unrelated A_T membership")
}
