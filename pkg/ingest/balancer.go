// Forked from https://github.com/grafana/loki/blob/fa6ef0a2caeeb4d31700287e9096e5f2c3c3a0d4/pkg/kafka/partitionring/consumer/balancer.go

package ingest

import (
	"sort"

	"github.com/grafana/dskit/ring"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// topicPartition identifies a partition within a topic. Partition IDs are only unique per
// topic, so inactive-owner tracking is keyed by both to avoid cross-topic collisions.
type topicPartition struct {
	topic     string
	partition int32
}

type cooperativeActiveStickyBalancer struct {
	kgo.GroupBalancer
	partitionRing ring.PartitionRingReader

	// prevInactiveOwners maps an inactive (topic, partition) to the member that owned it in the
	// previous assignment. It is captured in MemberBalancer (before inactive partitions are
	// filtered out of the member metadata) and consumed by Balance so that inactive partitions
	// stay on their previous owner instead of being reshuffled round-robin on every rebalance.
	// Keeping inactive partitions sticky avoids the churn that otherwise leaves stale
	// per-partition metrics (and reader state) on former owners after a handoff.
	prevInactiveOwners map[topicPartition]string
}

// NewCooperativeActiveStickyBalancer creates a balancer that combines Kafka's cooperative sticky balancing
// with partition ring awareness. It works by:
//
// 1. Using the partition ring to determine which partitions are "active" (i.e. should be processed)
// 2. Filtering out inactive partitions from member assignments during rebalancing, but still assigning them
// 3. Applying cooperative sticky balancing only to the active partitions
//
// This ensures that:
//   - Active partitions are balanced evenly across consumers using sticky assignment for optimal processing
//   - Inactive partitions are still assigned and consumed; each is kept on its previous owner when that
//     member is still present (sticky), falling back to round-robin only for partitions with no prior owner,
//     so they are not reshuffled on every rebalance
//   - All partitions are monitored even if inactive, allowing quick activation when needed
//   - Partition handoff happens cooperatively to avoid stop-the-world rebalances
//
// This balancer should be used with [NewGroupClient] which monitors the partition ring and triggers
// rebalancing when the set of active partitions changes. This ensures optimal partition distribution
// as the active partition set evolves.
func NewCooperativeActiveStickyBalancer(partitionRing ring.PartitionRingReader) kgo.GroupBalancer {
	return &cooperativeActiveStickyBalancer{
		GroupBalancer: kgo.CooperativeStickyBalancer(),
		partitionRing: partitionRing,
	}
}

func (*cooperativeActiveStickyBalancer) ProtocolName() string {
	return "cooperative-active-sticky"
}

func (b *cooperativeActiveStickyBalancer) MemberBalancer(members []kmsg.JoinGroupResponseMember) (kgo.GroupMemberBalancer, map[string]struct{}, error) {
	// Get active partitions from ring
	activePartitions := make(map[int32]struct{})
	for _, id := range b.partitionRing.PartitionRing().PartitionIDs() {
		activePartitions[id] = struct{}{}
	}

	// Record who currently owns each inactive partition, so Balance can keep it there
	// (sticky) rather than reshuffling it round-robin. Rebuilt fresh every rebalance.
	b.prevInactiveOwners = make(map[topicPartition]string)

	// Filter member metadata to only include active partitions
	filteredMembers := make([]kmsg.JoinGroupResponseMember, len(members))
	for i, member := range members {
		var meta kmsg.ConsumerMemberMetadata
		err := meta.ReadFrom(member.ProtocolMetadata)
		if err != nil {
			continue
		}

		// Remember inactive owned partitions before we strip them below.
		for _, owned := range meta.OwnedPartitions {
			for _, p := range owned.Partitions {
				if _, isActive := activePartitions[p]; !isActive {
					b.prevInactiveOwners[topicPartition{owned.Topic, p}] = member.MemberID
				}
			}
		}

		// Filter owned partitions to only include active ones
		filteredOwned := make([]kmsg.ConsumerMemberMetadataOwnedPartition, 0, len(meta.OwnedPartitions))
		for _, owned := range meta.OwnedPartitions {
			filtered := kmsg.ConsumerMemberMetadataOwnedPartition{
				Topic:      owned.Topic,
				Partitions: make([]int32, 0, len(owned.Partitions)),
			}
			for _, p := range owned.Partitions {
				if _, isActive := activePartitions[p]; isActive {
					filtered.Partitions = append(filtered.Partitions, p)
				}
			}
			if len(filtered.Partitions) > 0 {
				filteredOwned = append(filteredOwned, filtered)
			}
		}
		meta.OwnedPartitions = filteredOwned

		// Create filtered member
		filteredMembers[i] = kmsg.JoinGroupResponseMember{
			MemberID:         member.MemberID,
			ProtocolMetadata: meta.AppendTo(nil),
		}
	}

	balancer, err := kgo.NewConsumerBalancer(b, filteredMembers)
	return balancer, balancer.MemberTopics(), err
}

// syncAssignments implements kgo.IntoSyncAssignment
type syncAssignments []kmsg.SyncGroupRequestGroupAssignment

func (s syncAssignments) IntoSyncAssignment() []kmsg.SyncGroupRequestGroupAssignment {
	return s
}

func (b *cooperativeActiveStickyBalancer) Balance(balancer *kgo.ConsumerBalancer, topics map[string]int32) kgo.IntoSyncAssignment {
	// Get active partition count
	actives := b.partitionRing.PartitionRing().PartitionsCount()

	// First, let the sticky balancer handle active partitions
	activeTopics := make(map[string]int32)
	inactiveTopics := make(map[string]int32)
	for topic, total := range topics {
		activeTopics[topic] = int32(actives)
		if total > int32(actives) {
			inactiveTopics[topic] = total - int32(actives)
		}
	}

	// Get active partition assignment
	assignment := b.GroupBalancer.(kgo.ConsumerBalancerBalance).Balance(balancer, activeTopics)

	plan := assignment.IntoSyncAssignment()

	// Get sorted list of members for deterministic round-robin
	members := make([]string, 0, len(plan))
	for _, m := range plan {
		members = append(members, m.MemberID)
	}
	sort.Strings(members)

	// Nothing to assign to; return the active plan as-is.
	if len(members) == 0 {
		return syncAssignments(plan)
	}

	memberSet := make(map[string]struct{}, len(members))
	for _, m := range members {
		memberSet[m] = struct{}{}
	}

	// planIdx lets us append an inactive partition to a member's assignment by member ID.
	planIdx := make(map[string]int, len(plan))
	for i, m := range plan {
		planIdx[m.MemberID] = i
	}

	assignInactive := func(topic string, memberID string, p int32) {
		i, ok := planIdx[memberID]
		if !ok {
			return
		}
		var meta kmsg.ConsumerMemberAssignment
		// A member with no active partitions may have an empty assignment; treat that as an
		// empty ConsumerMemberAssignment rather than dropping the inactive partition. Only a
		// genuinely malformed (non-empty) assignment is skipped.
		if len(plan[i].MemberAssignment) > 0 {
			if err := meta.ReadFrom(plan[i].MemberAssignment); err != nil {
				return
			}
		}
		found := false
		for j := range meta.Topics {
			if meta.Topics[j].Topic == topic {
				meta.Topics[j].Partitions = append(meta.Topics[j].Partitions, p)
				found = true
				break
			}
		}
		if !found {
			meta.Topics = append(meta.Topics, kmsg.ConsumerMemberAssignmentTopic{
				Topic:      topic,
				Partitions: []int32{p},
			})
		}
		plan[i].MemberAssignment = meta.AppendTo(nil)
	}

	// Distribute inactive partitions, keeping each on its previous owner when that member is
	// still present (sticky). Partitions with no valid previous owner fall back to round-robin.
	fallbackIdx := 0
	for topic, numInactive := range inactiveTopics {
		for p := int32(actives); p < int32(actives)+numInactive; p++ {
			target := ""
			if owner, ok := b.prevInactiveOwners[topicPartition{topic, p}]; ok {
				if _, stillMember := memberSet[owner]; stillMember {
					target = owner
				}
			}
			if target == "" {
				target = members[fallbackIdx]
				fallbackIdx = (fallbackIdx + 1) % len(members)
			}
			assignInactive(topic, target, p)
		}
	}

	return syncAssignments(plan)
}
