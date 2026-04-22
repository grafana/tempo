package generator

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/ingest/ingesttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

// TestHandlePartitionsAssigned_CooperativeAppend verifies that incremental
// (cooperative) rebalances accumulate newly assigned partitions on top of
// stable ones rather than replacing the entire assignment.
//
// In franz-go's cooperative sticky balancer, OnPartitionsAssigned fires with
// only the delta — partitions newly added to this member.  Stable partitions
// (kept across the rebalance) are not re-reported.  The callback must therefore
// append, not replace.
func TestHandlePartitionsAssigned_CooperativeAppend(t *testing.T) {
	g := minimalGeneratorForKafkaTest()

	// 1. Initial join: member is new so the full assignment is "new".
	g.handlePartitionsAssigned(map[string][]int32{"test-topic": {0, 1, 2, 3}})
	assert.Equal(t, []int32{0, 1, 2, 3}, g.assignedPartitions)

	// 2. Cooperative partial revoke: another member joined and took {2, 3}.
	g.handlePartitionsRevoked(map[string][]int32{"test-topic": {2, 3}})
	assert.Equal(t, []int32{0, 1}, g.assignedPartitions)

	// 3. Cooperative incremental assign: the callback fires with only the
	//    newly added partitions {4, 5}.  Stable partitions {0, 1} must be
	//    retained — a replace operation would silently lose them.
	g.handlePartitionsAssigned(map[string][]int32{"test-topic": {4, 5}})
	assert.Equal(t, []int32{0, 1, 4, 5}, g.assignedPartitions,
		"stable partitions must be retained during an incremental cooperative rebalance")
}

// TestHandlePartitionsRevoked_RemovesOnlyRevokedPartitions verifies that only
// the specified partitions are removed and the rest are preserved.
func TestHandlePartitionsRevoked_RemovesOnlyRevokedPartitions(t *testing.T) {
	g := minimalGeneratorForKafkaTest()
	g.assignedPartitions = []int32{0, 1, 2, 3, 4, 5}

	g.handlePartitionsRevoked(map[string][]int32{"test-topic": {1, 3, 5}})
	assert.Equal(t, []int32{0, 2, 4}, g.assignedPartitions)
}

// TestHandlePartitionsAssigned_UnknownTopicIsIgnored verifies that assignment
// callbacks for a different topic do not affect the tracked partitions.
func TestHandlePartitionsAssigned_UnknownTopicIsIgnored(t *testing.T) {
	g := minimalGeneratorForKafkaTest()
	g.assignedPartitions = []int32{0, 1}

	g.handlePartitionsAssigned(map[string][]int32{"other-topic": {2, 3}})
	assert.Equal(t, []int32{0, 1}, g.assignedPartitions,
		"partitions for an unrelated topic must not be added")
}

// minimalGeneratorForKafkaTest returns a Generator populated with the fields
// required by handlePartitionsAssigned / handlePartitionsRevoked without
// starting any goroutines or real services.
func minimalGeneratorForKafkaTest() *Generator {
	return &Generator{
		cfg: &Config{
			Ingest: ingest.Config{
				Kafka: ingest.KafkaConfig{
					Topic:         "test-topic",
					ConsumerGroup: "test-group",
				},
			},
		},
		logger: log.NewNopLogger(),
	}
}

// TestGetAssignedActivePartitions_ReturnsCopy verifies that the slice returned
// by getAssignedActivePartitions is a copy of the internal slice. Mutating the
// returned value must not affect g.assignedPartitions, ensuring that the lag
// metrics goroutine (which iterates the returned slice without holding the lock)
// cannot race with handlePartitionsAssigned (which appends to the same backing
// array).
func TestGetAssignedActivePartitions_ReturnsCopy(t *testing.T) {
	g := minimalGeneratorForKafkaTest()
	g.assignedPartitions = []int32{0, 1, 2}

	got := g.getAssignedActivePartitions()
	assert.Equal(t, []int32{0, 1, 2}, got)

	got[0] = 99
	assert.Equal(t, []int32{0, 1, 2}, g.assignedPartitions,
		"mutating the returned slice must not affect internal state")
}

// TestStopKafka_LeaveGroupConditional verifies the three branches of the
// LeaveConsumerGroupOnShutdown conditional in stopKafka():
//   - leave=true, instanceID set  → leaveGroupFn is called
//   - leave=false, instanceID set → leaveGroupFn is NOT called
//   - leave=true, instanceID ""   → leaveGroupFn is NOT called
func TestStopKafka_LeaveGroupConditional(t *testing.T) {
	tests := []struct {
		name            string
		leaveOnShutdown bool
		instanceID      string
		expectLeave     bool
	}{
		{"leave=true instanceID set", true, "gen-1", true},
		{"leave=false instanceID set", false, "gen-1", false},
		{"leave=true instanceID empty", true, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake, err := kfake.NewCluster(kfake.NumBrokers(1))
			require.NoError(t, err)
			t.Cleanup(fake.Close)

			kgoClient, err := kgo.NewClient(
				kgo.SeedBrokers(fake.ListenAddrs()...),
				kgo.DisableClientMetrics(),
			)
			require.NoError(t, err)

			var leaveCalled bool
			g := &Generator{
				cfg: &Config{
					Ingest: ingest.Config{
						Kafka: ingest.KafkaConfig{
							Topic:         "test-topic",
							ConsumerGroup: "test-group",
						},
					},
					LeaveConsumerGroupOnShutdown: tc.leaveOnShutdown,
					InstanceID:                   tc.instanceID,
				},
				logger:      log.NewNopLogger(),
				kafkaStop:   func() {},
				kafkaCh:     make(chan *kgo.Record, 1),
				kafkaClient: ingesttest.NewClient(kgoClient),
				leaveGroupFn: func(_ context.Context) error {
					leaveCalled = true
					return nil
				},
			}

			g.stopKafka()

			assert.Equal(t, tc.expectLeave, leaveCalled, "leaveGroupFn called mismatch")
		})
	}
}

// TestPartitionHandoff_LeaveGroupTriggersImmediateReassignment is an end-to-end
// test of the core problem this PR solves.
//
// With static membership (InstanceID set), franz-go intentionally skips the
// automatic LeaveGroup on Close() so that a pod that restarts with the same
// InstanceID can rejoin without waiting for session timeout.  When using a
// Deployment (changing pod names = changing InstanceIDs), the departing pod
// must explicitly send LeaveGroup so the coordinator can reassign the partition
// immediately rather than after the 3-minute session timeout.
//
// kfake models session timeouts accurately: it only triggers a rebalance after
// SessionTimeoutMillis when no heartbeat has been received, and immediately on
// a LeaveGroup request.  This makes the test genuinely meaningful — the two
// behaviours produce measurably different transfer times.
//
// Test layout:
//  1. 1-partition topic, two consumers with distinct InstanceIDs in the same group.
//  2. gen-1 starts and acquires partition 0; gen-2 starts and gets nothing.
//  3. gen-1 simulates stopKafka(LeaveConsumerGroupOnShutdown=true): sends an
//     explicit LeaveGroup then closes.
//  4. gen-2 must acquire partition 0 well within the session timeout.
//
// Negative case: a second sub-test closes gen-1 without LeaveGroup and asserts
// that gen-2 does NOT acquire the partition within the same window, confirming
// that LeaveGroup is what makes the handoff fast.
func TestPartitionHandoff_LeaveGroupTriggersImmediateReassignment(t *testing.T) {
	const (
		topic          = "handoff-topic"
		group          = "handoff-group"
		sessionTimeout = 8 * time.Second // long enough that "no LeaveGroup" does not transfer within handoffTimeout
		handoffTimeout = 3 * time.Second // must be < sessionTimeout
	)

	// newConsumer creates a kgo consumer with static membership.  It returns the
	// client and a channel that receives the partition slice the first time any
	// non-empty OnPartitionsAssigned fires.
	newConsumer := func(t *testing.T, addr, instanceID string) (*kgo.Client, <-chan []int32) {
		t.Helper()
		assigned := make(chan []int32, 1)
		client, err := kgo.NewClient(
			kgo.SeedBrokers(addr),
			kgo.ConsumerGroup(group),
			kgo.ConsumeTopics(topic),
			kgo.InstanceID(instanceID),
			kgo.SessionTimeout(sessionTimeout),
			kgo.HeartbeatInterval(500*time.Millisecond),
			kgo.Balancers(kgo.CooperativeStickyBalancer()),
			kgo.DisableClientMetrics(),
			kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
			kgo.OnPartitionsAssigned(func(_ context.Context, _ *kgo.Client, m map[string][]int32) {
				if p := m[topic]; len(p) > 0 {
					select {
					case assigned <- append([]int32(nil), p...):
					default:
					}
				}
			}),
		)
		require.NoError(t, err)
		return client, assigned
	}

	// poll drives group coordination for a client until the context is cancelled.
	poll := func(ctx context.Context, client *kgo.Client) {
		for ctx.Err() == nil {
			client.PollFetches(ctx)
		}
	}

	t.Run("with LeaveGroup", func(t *testing.T) {
		fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic))
		require.NoError(t, err)
		t.Cleanup(fake.Close)
		addr := fake.ListenAddrs()[0]

		client1, gen1Assigned := newConsumer(t, addr, "gen-1")
		t.Cleanup(client1.Close)
		ctx1, cancel1 := context.WithCancel(context.Background())
		t.Cleanup(cancel1)
		go poll(ctx1, client1)

		// Wait for gen-1 to own partition 0.
		select {
		case p := <-gen1Assigned:
			require.Equal(t, []int32{0}, p)
		case <-time.After(10 * time.Second):
			cancel1()
			t.Fatal("gen-1 did not receive partition 0")
		}

		// Start gen-2 while gen-1 still holds the partition.
		client2, gen2Assigned := newConsumer(t, addr, "gen-2")
		t.Cleanup(client2.Close)
		ctx2, cancel2 := context.WithCancel(context.Background())
		t.Cleanup(cancel2)
		go poll(ctx2, client2)

		// Simulate stopKafka() with LeaveConsumerGroupOnShutdown=true.
		cancel1()
		require.NoError(t, ingest.LeaveConsumerGroupByInstanceID(
			context.Background(), client1, group, "gen-1", log.NewNopLogger(),
		))
		client1.Close()

		// gen-2 must pick up partition 0 quickly — well before the session timeout.
		select {
		case p := <-gen2Assigned:
			require.Equal(t, []int32{0}, p)
		case <-time.After(handoffTimeout):
			t.Fatalf("gen-2 did not acquire partition 0 within %s; LeaveGroup handoff is broken", handoffTimeout)
		}
	})

	t.Run("without LeaveGroup", func(t *testing.T) {
		fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic))
		require.NoError(t, err)
		t.Cleanup(fake.Close)
		addr := fake.ListenAddrs()[0]

		client1, gen1Assigned := newConsumer(t, addr, "gen-1")
		t.Cleanup(client1.Close)
		ctx1, cancel1 := context.WithCancel(context.Background())
		t.Cleanup(cancel1)
		go poll(ctx1, client1)

		select {
		case p := <-gen1Assigned:
			require.Equal(t, []int32{0}, p)
		case <-time.After(10 * time.Second):
			cancel1()
			t.Fatal("gen-1 did not receive partition 0")
		}

		client2, gen2Assigned := newConsumer(t, addr, "gen-2")
		t.Cleanup(client2.Close)
		ctx2, cancel2 := context.WithCancel(context.Background())
		t.Cleanup(cancel2)
		go poll(ctx2, client2)

		// Close without LeaveGroup — gen-2 must wait for the session timeout.
		cancel1()
		client1.Close() // no explicit LeaveGroup

		// gen-2 must NOT receive the partition within handoffTimeout because the
		// coordinator is still waiting for the session timeout to expire.
		select {
		case p := <-gen2Assigned:
			t.Fatalf("gen-2 received partition %v before session timeout; expected to wait", p)
		case <-time.After(handoffTimeout):
			// correct: partition did not transfer before session timeout
		}
	})
}
