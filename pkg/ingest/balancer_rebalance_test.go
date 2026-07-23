package ingest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	rebalanceTopic = "rebalance-topic"
	rebalanceGroup = "rebalance-group"
)

// ownershipTracker records, per member, the partitions currently assigned to it, driven by the
// OnPartitionsAssigned / OnPartitionsRevoked callbacks.
type ownershipTracker struct {
	mu    sync.Mutex
	owned map[string]map[int32]struct{}
}

func newOwnershipTracker() *ownershipTracker {
	return &ownershipTracker{owned: map[string]map[int32]struct{}{}}
}

func (o *ownershipTracker) assign(member, topic string, m map[string][]int32) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.owned[member] == nil {
		o.owned[member] = map[int32]struct{}{}
	}
	for _, p := range m[topic] {
		o.owned[member][p] = struct{}{}
	}
}

func (o *ownershipTracker) revoke(member, topic string, m map[string][]int32) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, p := range m[topic] {
		delete(o.owned[member], p)
	}
}

func (o *ownershipTracker) snapshot(member string) map[int32]struct{} {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make(map[int32]struct{}, len(o.owned[member]))
	for p := range o.owned[member] {
		out[p] = struct{}{}
	}
	return out
}

func mustRebalanceRing(t *testing.T, activeCount int32) ring.PartitionRingReader {
	t.Helper()
	partitions := make(map[int32]ring.PartitionDesc, activeCount)
	for id := int32(0); id < activeCount; id++ {
		partitions[id] = ring.PartitionDesc{Id: id, State: ring.PartitionActive}
	}
	r, err := ring.NewPartitionRing(ring.PartitionRingDesc{Partitions: partitions})
	require.NoError(t, err)
	return fakeRebalanceRingReader{r: r}
}

type fakeRebalanceRingReader struct{ r *ring.PartitionRing }

func (f fakeRebalanceRingReader) PartitionRing() *ring.PartitionRing { return f.r }

// TestCooperativeActiveStickyBalancer_InactivePartitionsStayOnOwnerAcrossRebalance drives two real
// consumer-group members through kfake and asserts that when the second member joins, the inactive
// partitions already owned by the first member stay with it (rather than being reshuffled), and no
// partition is ever owned by both members. This guards against the stale duplicate-owner metrics
// that the round-robin inactive distribution used to produce.
func TestCooperativeActiveStickyBalancer_InactivePartitionsStayOnOwnerAcrossRebalance(t *testing.T) {
	const totalPartitions = 4
	const activePartitions = 2 // active: 0,1  inactive: 2,3

	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(totalPartitions, rebalanceTopic))
	require.NoError(t, err)
	t.Cleanup(fake.Close)
	addr := fake.ListenAddrs()[0]

	ringReader := mustRebalanceRing(t, activePartitions)
	tracker := newOwnershipTracker()

	cfg := ingest.KafkaConfig{
		Address:                        addr,
		Topic:                          rebalanceTopic,
		ConsumerGroup:                  rebalanceGroup,
		DisableKafkaTelemetry:          true,
		LastProducedOffsetRetryTimeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	var clients []*ingest.Client

	// Always stop the poll-loop goroutines before closing the clients (and, registered earlier,
	// the fake cluster), even if an assertion below fails early. t.Cleanup runs LIFO, so this
	// runs ahead of fake.Close and closes clients only after the goroutines have exited.
	t.Cleanup(func() {
		cancel()
		wg.Wait()
		for _, c := range clients {
			c.Close()
		}
	})

	startMember := func(member string) {
		client, err := ingest.NewGroupReaderClient(
			cfg, ringReader, nil, log.NewNopLogger(),
			kgo.OnPartitionsAssigned(func(_ context.Context, _ *kgo.Client, m map[string][]int32) {
				tracker.assign(member, rebalanceTopic, m)
			}),
			kgo.OnPartitionsRevoked(func(_ context.Context, _ *kgo.Client, m map[string][]int32) {
				tracker.revoke(member, rebalanceTopic, m)
			}),
		)
		require.NoError(t, err)
		clients = append(clients, client)

		// A poll loop is required to drive the group protocol (join/sync/heartbeat).
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				client.PollFetches(ctx)
			}
		}()
	}

	// Member A joins alone and should own every partition (active via sticky, inactive via the
	// round-robin fallback since no partition has a previous owner yet).
	startMember("a")
	require.Eventually(t, func() bool {
		return len(tracker.snapshot("a")) == totalPartitions
	}, 30*time.Second, 250*time.Millisecond, "member a should own all partitions while alone")

	// Member B joins, forcing a rebalance.
	startMember("b")

	// Wait for the group to settle: all partitions assigned, ownership disjoint, and B has picked
	// up at least one (active) partition.
	require.Eventually(t, func() bool {
		a, b := tracker.snapshot("a"), tracker.snapshot("b")
		if len(a)+len(b) != totalPartitions {
			return false
		}
		for p := range a {
			if _, dup := b[p]; dup {
				return false
			}
		}
		return len(b) >= 1
	}, 30*time.Second, 250*time.Millisecond, "group should settle with disjoint ownership after b joins")

	// The fix: inactive partitions 2 and 3 remain on their previous owner (a) instead of being
	// reshuffled to b.
	a, b := tracker.snapshot("a"), tracker.snapshot("b")
	require.Contains(t, a, int32(2), "inactive partition 2 must stay on a: a=%v b=%v", a, b)
	require.Contains(t, a, int32(3), "inactive partition 3 must stay on a: a=%v b=%v", a, b)
	require.NotContains(t, b, int32(2), "inactive partition 2 must not move to b: a=%v b=%v", a, b)
	require.NotContains(t, b, int32(3), "inactive partition 3 must not move to b: a=%v b=%v", a, b)
}
