package generator

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/ingest"
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
				kafkaClient: ingest.NewClientForTesting(kgoClient),
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
