// This file is WP18's named end-to-end deliverable (implementation plan
// §3 WP18 test plan): the rewind-replay-flip sequence exercised against a
// REAL (fake-broker) Consumer, rather than the synthetic fakeRewinder the
// rest of reconstruction_test.go uses for precise ordering control.
package bloomgateway

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/time/rate"

	"github.com/grafana/tempo/pkg/ingest/testkafka"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
)

// TestReconstruction_KfakeEndToEnd_RewindReplayFlip: an AddChunk event is
// published to a real (fake-broker) topic, but this instance's Consumer is
// deliberately started PAST it (offset 1, skipping offset 0 entirely) --
// exactly what a newly-acquired leaf range's consumer position looks like
// (DESIGN.md § Leaf lifecycle: "a leaf acquired at runtime must not be
// populated from topic tail alone"). Only reconstruction's rewind-and-
// replay can ever recover it; ordinary live consumption never will.
func TestReconstruction_KfakeEndToEnd_RewindReplayFlip(t *testing.T) {
	const topic = "bg-reconstruction-e2e"
	_, addr := testkafka.CreateCluster(t, 1, topic)
	produceClient := testkafka.NewKafkaClient(t, addr, topic)

	applier, dir, reg, _ := newTestApplier(t, false /* leaf starts nil */)
	tr := testTimeRange()

	const leafIdx = uint32(5)
	missedID := findTraceIDForLeaf(t, applier.hashSeed, testD, testF, leafIdx)
	missedUUID := testUUID(t, 42)

	ctx := context.Background()
	// Published at offset 0, BEFORE the consumer below ever starts -- this
	// instance resumes at offset 1, so ordinary live consumption never
	// sees it; only reconstruction's rewind can.
	res := produceClient.ProduceSync(ctx, &kgo.Record{
		Topic: topic, Partition: 0,
		Value: mustMarshalEvent(t, addEvent(missedUUID, "tenant-a", tr, [][]byte{missedID})),
	})
	require.NoError(t, res.FirstErr())

	cfg := newTestKafkaConfig(addr, topic)
	consumer, err := NewConsumer(cfg, "bloom-gateway-0", 1<<20, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(func() { _ = consumer.Close() })

	pool := NewWorkerPool(2, consumer.Records(), applier, log.NewNopLogger(), applier.metrics)
	poolCtx, poolCancel := context.WithCancel(ctx)
	defer poolCancel()
	pool.Start(poolCtx)
	defer pool.Stop()

	require.NoError(t, consumer.Start(ctx, map[int32]int64{0: 1}))

	reader := newFakeBackendReader()
	reader.setTenantIndex("tenant-a", &backend.TenantIndex{
		CreatedAt: time.Now(),
		Meta: []*backend.BlockMeta{
			blockMetaFixture(testUUID(t, 1), "tenant-a", tr, vparquet3.VersionString),
		},
	})

	q := NewReconstructionQueue(dir, applier, consumer, reader, ReconstructionConfig{Concurrency: 2}, rate.NewLimiter(rate.Inf, 0), applier.metrics, log.NewNopLogger())
	q.Enqueue([]LeafRange{{Start: leafIdx, End: leafIdx + 1}})

	batchCtx, batchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer batchCancel()
	stats, err := q.RunBatch(batchCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.LeavesStarted)
	require.Equal(t, LeafComplete, dir.State(leafIdx))

	leafIdxOfMissed, fp := Address(missedID, applier.hashSeed, testD, testF)
	require.Equal(t, leafIdx, leafIdxOfMissed, "test setup sanity: missedID must route to the reconstructed leaf")

	require.Eventually(t, func() bool {
		handles, ok := dir.Lookup(leafIdx, uint16(fp))
		if !ok {
			return false
		}
		block, ok := reg.LookupUUID(missedUUID)
		return ok && containsHandle(handles, block.Handle)
	}, 5*time.Second, 20*time.Millisecond,
		"the rewind-driven replay must recover and apply the event this instance's ordinary consumption would otherwise never see")
}

func containsHandle(hs []Handle, h Handle) bool {
	for _, x := range hs {
		if x == h {
			return true
		}
	}
	return false
}
