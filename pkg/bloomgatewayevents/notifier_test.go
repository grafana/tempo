package bloomgatewayevents

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/goleak"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
)

// TestNotifier_BlockCompacted_PublishesAdd proves the adapter forwards
// BlockCompacted to PublishAdd with the meta's fields round-tripped and the
// exact trace ID set delivered, against a real Publisher and fake broker
// (newKfakeCluster/newTestConfig/pollRecords are defined in
// publisher_test.go, same package).
func TestNotifier_BlockCompacted_PublishesAdd(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "notifier-add"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	n := NewNotifier(pub)

	meta := &backend.BlockMeta{
		BlockID:   backend.NewUUID(),
		TenantID:  "tenant-notify-add",
		StartTime: time.Now().Add(-time.Hour).Truncate(time.Nanosecond),
		EndTime:   time.Now().Truncate(time.Nanosecond),
	}
	ids := [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 9, 9, 9}}

	n.BlockCompacted(meta, ids)

	recs := pollRecords(t, reader, 1, 5*time.Second)
	require.Len(t, recs, 1)

	event := &tempopb.BloomGatewayEvent{}
	require.NoError(t, event.Unmarshal(recs[0].Value))
	assert.EqualValues(t, 1, event.Version)
	assert.Equal(t, tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_ADD_CHUNK, event.Type)
	require.NotNil(t, event.AddChunk)
	assert.Equal(t, meta.BlockID.String(), event.AddChunk.BlockId)
	assert.Equal(t, meta.TenantID, event.AddChunk.TenantId)
	assert.Equal(t, meta.StartTime.UnixNano(), event.AddChunk.StartTimeUnixNano)
	assert.Equal(t, meta.EndTime.UnixNano(), event.AddChunk.EndTimeUnixNano)
	assert.ElementsMatch(t, ids, event.AddChunk.TraceIds)

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestNotifier_BlockDeleted_PublishesDelete proves the adapter forwards
// BlockDeleted to PublishDelete, reading the block ID off
// CompactedBlockMeta's embedded BlockMeta.
func TestNotifier_BlockDeleted_PublishesDelete(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "notifier-delete"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	reader, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.ConsumeTopics(topic))
	require.NoError(t, err)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	n := NewNotifier(pub)

	compactedMeta := &backend.CompactedBlockMeta{
		BlockMeta: backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: "tenant-notify-delete",
		},
		CompactedTime: time.Now(),
	}

	n.BlockDeleted(compactedMeta)

	recs := pollRecords(t, reader, 1, 5*time.Second)
	require.Len(t, recs, 1)

	event := &tempopb.BloomGatewayEvent{}
	require.NoError(t, event.Unmarshal(recs[0].Value))
	assert.EqualValues(t, 1, event.Version)
	assert.Equal(t, tempopb.BloomGatewayEventType_BLOOM_GATEWAY_EVENT_TYPE_DELETE, event.Type)
	require.NotNil(t, event.Delete)
	assert.Equal(t, compactedMeta.BlockID.String(), event.Delete.BlockId)

	pub.Close()
	reader.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestNotifier_EmptyIDs_NoPublish locks in that BlockCompacted with no
// trace IDs reaches PublishAdd's own empty no-op (publisher.go) rather than
// publishing an empty AddChunk -- no record is produced and no counter
// moves.
func TestNotifier_EmptyIDs_NoPublish(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "notifier-empty"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry())
	require.NoError(t, err)

	n := NewNotifier(pub)

	meta := &backend.BlockMeta{
		BlockID:  backend.NewUUID(),
		TenantID: "tenant-notify-empty",
	}
	n.BlockCompacted(meta, nil)

	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)))
	assert.Equal(t, float64(0), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultDropped)))

	pub.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}

// TestNotifier_BlockDeleted_RateLimitedUsesTenantID proves BlockDeleted
// threads meta.TenantID into PublishDelete's rate limiting. BloomGatewayDelete
// itself carries no tenant field on the wire (DESIGN.md's schema was never
// extended for this, since tenant is only ever used for limiting, never
// sent) -- so the only way to observe which tenant was used is indirectly:
// exhausting the limiter for meta.TenantID and then calling BlockDeleted
// again must be rate-limited, which could only happen if BlockDeleted passed
// exactly that tenant through.
func TestNotifier_BlockDeleted_RateLimitedUsesTenantID(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	const topic = "notifier-delete-ratelimited"
	const numPartitions = int32(4)
	cluster, addr := newKfakeCluster(t, numPartitions, topic, nil)

	cfg := newTestConfig(t, addr, topic, numPartitions)
	pub, err := New(cfg, log.NewNopLogger(), prometheus.NewRegistry(), WithTenantLimits(func(string) float64 { return 1 }))
	require.NoError(t, err)

	n := NewNotifier(pub)

	compactedMeta := &backend.CompactedBlockMeta{
		BlockMeta: backend.BlockMeta{
			BlockID:  backend.NewUUID(),
			TenantID: "tenant-notify-delete-rl",
		},
		CompactedTime: time.Now(),
	}

	n.BlockDeleted(compactedMeta) // consumes this tenant's only token
	n.BlockDeleted(compactedMeta) // must be rate-limited

	assert.Equal(t, float64(1), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultOK)))
	assert.Equal(t, float64(1), testutil.ToFloat64(pub.metrics.publishesTotal.WithLabelValues(resultRateLimited)))

	pub.Close()
	cluster.Close()
	goleak.VerifyNone(t, opts)
}
