package ingest

import (
	"sort"
	"testing"

	"github.com/grafana/dskit/ring"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const balancerTestTopic = "t"

type fakePartitionRingReader struct{ r *ring.PartitionRing }

func (f fakePartitionRingReader) PartitionRing() *ring.PartitionRing { return f.r }

// activeRing returns a ring reader whose ring contains partitions [0, activeCount)
// all Active. The balancer treats partitions present in the ring as active and Kafka
// partitions with an index >= the ring's partition count as inactive.
func activeRing(t *testing.T, activeCount int32) fakePartitionRingReader {
	t.Helper()
	partitions := make(map[int32]ring.PartitionDesc, activeCount)
	for id := int32(0); id < activeCount; id++ {
		partitions[id] = ring.PartitionDesc{Id: id, State: ring.PartitionActive}
	}
	r, err := ring.NewPartitionRing(ring.PartitionRingDesc{Partitions: partitions})
	require.NoError(t, err)
	return fakePartitionRingReader{r: r}
}

// balancerMember builds a group member subscribed to the test topic that currently owns
// the given partitions.
func balancerMember(t *testing.T, id string, owned ...int32) kmsg.JoinGroupResponseMember {
	t.Helper()
	meta := kmsg.NewConsumerMemberMetadata()
	meta.Version = 3
	meta.Topics = []string{balancerTestTopic}
	if len(owned) > 0 {
		op := kmsg.NewConsumerMemberMetadataOwnedPartition()
		op.Topic = balancerTestTopic
		op.Partitions = append([]int32(nil), owned...)
		meta.OwnedPartitions = []kmsg.ConsumerMemberMetadataOwnedPartition{op}
	}
	return kmsg.JoinGroupResponseMember{MemberID: id, ProtocolMetadata: meta.AppendTo(nil)}
}

// runBalance drives MemberBalancer then Balance for a topic with totalPartitions and returns
// memberID -> sorted assigned partitions.
func runBalance(t *testing.T, rr fakePartitionRingReader, totalPartitions int32, members []kmsg.JoinGroupResponseMember) map[string][]int32 {
	t.Helper()
	b := &cooperativeActiveStickyBalancer{
		GroupBalancer: kgo.CooperativeStickyBalancer(),
		partitionRing: rr,
	}
	gmb, _, err := b.MemberBalancer(members)
	require.NoError(t, err)
	cb, ok := gmb.(*kgo.ConsumerBalancer)
	require.True(t, ok, "MemberBalancer must return a *kgo.ConsumerBalancer")

	plan := b.Balance(cb, map[string]int32{balancerTestTopic: totalPartitions}).IntoSyncAssignment()

	out := make(map[string][]int32, len(plan))
	for _, m := range plan {
		// A member with no active partitions has an empty assignment (a valid case the
		// balancer handles); mirror that here instead of failing on ReadFrom.
		if len(m.MemberAssignment) == 0 {
			continue
		}
		var asg kmsg.ConsumerMemberAssignment
		require.NoError(t, asg.ReadFrom(m.MemberAssignment))
		for _, tp := range asg.Topics {
			if tp.Topic != balancerTestTopic {
				continue
			}
			ps := append([]int32(nil), tp.Partitions...)
			sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
			out[m.MemberID] = ps
		}
	}
	return out
}

// requireNoDoubleOwnership asserts no partition is assigned to more than one member. Active
// partitions may be held back across a single cooperative round, so this does not require every
// partition to be present; it only guards the invariant the fix protects: no duplicate ownership.
func requireNoDoubleOwnership(t *testing.T, assignments map[string][]int32) {
	t.Helper()
	owner := make(map[int32]string)
	for member, ps := range assignments {
		for _, p := range ps {
			prev, dup := owner[p]
			require.Falsef(t, dup, "partition %d assigned to both %s and %s", p, prev, member)
			owner[p] = member
		}
	}
}

// requireAssigned asserts each of the given partitions is assigned to exactly one member.
func requireAssigned(t *testing.T, assignments map[string][]int32, partitions ...int32) {
	t.Helper()
	for _, p := range partitions {
		count := 0
		for _, ps := range assignments {
			for _, got := range ps {
				if got == p {
					count++
				}
			}
		}
		require.Equalf(t, 1, count, "partition %d must be assigned to exactly one member: %v", p, assignments)
	}
}

// ownsAll reports whether member's assignment contains all of the given partitions.
func ownsAll(assignments map[string][]int32, member string, partitions ...int32) bool {
	set := make(map[int32]struct{}, len(assignments[member]))
	for _, p := range assignments[member] {
		set[p] = struct{}{}
	}
	for _, p := range partitions {
		if _, ok := set[p]; !ok {
			return false
		}
	}
	return true
}

func ownsAny(assignments map[string][]int32, member string, partitions ...int32) bool {
	set := make(map[int32]struct{}, len(assignments[member]))
	for _, p := range assignments[member] {
		set[p] = struct{}{}
	}
	for _, p := range partitions {
		if _, ok := set[p]; ok {
			return true
		}
	}
	return false
}

// Inactive partitions (2, 3) previously owned by a single member stay with that member
// rather than being reshuffled round-robin to the other member.
func TestBalancer_InactivePartitionsStickToSolePreviousOwner(t *testing.T) {
	rr := activeRing(t, 2) // active: 0,1  inactive: 2,3
	members := []kmsg.JoinGroupResponseMember{
		balancerMember(t, "a", 0, 1, 2, 3),
		balancerMember(t, "b"),
	}

	got := runBalance(t, rr, 4, members)

	requireNoDoubleOwnership(t, got)
	requireAssigned(t, got, 2, 3)
	require.True(t, ownsAll(got, "a", 2, 3), "a must keep its inactive partitions: %v", got)
	require.False(t, ownsAny(got, "b", 2, 3), "b must not receive a's inactive partitions: %v", got)
}

// Inactive partitions stay with their respective previous owners when spread across members.
func TestBalancer_InactivePartitionsStickToRespectiveOwners(t *testing.T) {
	rr := activeRing(t, 2) // active: 0,1  inactive: 2,3
	members := []kmsg.JoinGroupResponseMember{
		balancerMember(t, "a", 0, 2),
		balancerMember(t, "b", 1, 3),
	}

	got := runBalance(t, rr, 4, members)

	requireNoDoubleOwnership(t, got)
	requireAssigned(t, got, 2, 3)
	require.True(t, ownsAll(got, "a", 2), "a must keep inactive partition 2: %v", got)
	require.True(t, ownsAll(got, "b", 3), "b must keep inactive partition 3: %v", got)
}

// Inactive partitions with no previous owner are distributed round-robin over sorted members.
func TestBalancer_UnownedInactivePartitionsRoundRobin(t *testing.T) {
	rr := activeRing(t, 2) // active: 0,1  inactive: 2,3
	members := []kmsg.JoinGroupResponseMember{
		balancerMember(t, "a", 0),
		balancerMember(t, "b", 1),
	}

	got := runBalance(t, rr, 4, members)

	requireNoDoubleOwnership(t, got)
	requireAssigned(t, got, 2, 3)
	// Sorted members are [a, b]; fallback round-robin assigns 2->a, 3->b.
	require.True(t, ownsAll(got, "a", 2), "expected 2 on a: %v", got)
	require.True(t, ownsAll(got, "b", 3), "expected 3 on b: %v", got)
}
