// SPDX-License-Identifier: AGPL-3.0-only

package testkafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kmsg"
)

type controlFn func(kmsg.Request) (kmsg.Response, error, bool)

type Cluster struct {
	t                testing.TB
	fake             *kfake.Cluster
	topic            string
	numPartitions    int
	committedOffsets map[string][]int64
	controlFuncs     map[kmsg.Key]controlFn
}

// CreateCluster returns a fake Kafka cluster for unit testing.
func CreateCluster(t testing.TB, numPartitions int32, topicName string) (*Cluster, string) {
	fake, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(numPartitions, topicName))
	require.NoError(t, err)
	t.Cleanup(fake.Close)

	addrs := fake.ListenAddrs()
	require.Len(t, addrs, 1)

	c := &Cluster{
		t:                t,
		fake:             fake,
		topic:            topicName,
		numPartitions:    int(numPartitions),
		committedOffsets: map[string][]int64{},
		controlFuncs:     map[kmsg.Key]controlFn{},
	}

	// Add support for consumer groups
	c.fake.ControlKey(kmsg.OffsetCommit.Int16(), c.offsetCommit)
	c.fake.ControlKey(kmsg.OffsetFetch.Int16(), c.offsetFetch)

	return c, addrs[0]
}

func (c *Cluster) ControlKey(key kmsg.Key, fn controlFn) {
	switch key {
	case kmsg.OffsetCommit:
		// These are called by us for deterministic order
		c.controlFuncs[key] = fn
	default:
		// These are passed through
		c.fake.ControlKey(int16(key), fn)
	}
}

func (c *Cluster) ensureConsumerGroupExists(consumerGroup string) {
	if _, ok := c.committedOffsets[consumerGroup]; ok {
		return
	}
	c.committedOffsets[consumerGroup] = make([]int64, c.numPartitions+1)

	// Initialise the partition offsets with the special value -1 which means "no offset committed".
	for i := 0; i < len(c.committedOffsets[consumerGroup]); i++ {
		c.committedOffsets[consumerGroup][i] = -1
	}
}

// nolint: revive
func (c *Cluster) offsetCommit(request kmsg.Request) (kmsg.Response, error, bool) {
	c.fake.KeepControl()

	if fn := c.controlFuncs[kmsg.OffsetCommit]; fn != nil {
		res, err, handled := fn(request)
		if handled {
			return res, err, handled
		}
	}

	commitR := request.(*kmsg.OffsetCommitRequest)
	consumerGroup := commitR.Group
	c.ensureConsumerGroupExists(consumerGroup)
	require.Len(c.t, commitR.Topics, 1, "test only has support for one topic per request")
	topic := commitR.Topics[0]
	require.Equal(c.t, c.topic, topic.Topic)
	require.Len(c.t, topic.Partitions, 1, "test only has support for one partition per request")

	partitionID := topic.Partitions[0].Partition
	c.committedOffsets[consumerGroup][partitionID] = topic.Partitions[0].Offset

	resp := request.ResponseKind().(*kmsg.OffsetCommitResponse)
	resp.Default()
	resp.Topics = []kmsg.OffsetCommitResponseTopic{
		{
			Topic:      c.topic,
			Partitions: []kmsg.OffsetCommitResponseTopicPartition{{Partition: partitionID}},
		},
	}

	return resp, nil, true
}

// nolint: revive
func (c *Cluster) offsetFetch(kreq kmsg.Request) (kmsg.Response, error, bool) {
	c.fake.KeepControl()
	req := kreq.(*kmsg.OffsetFetchRequest)
	require.Len(c.t, req.Groups, 1, "test only has support for one consumer group per request")
	consumerGroup := req.Groups[0].Group
	c.ensureConsumerGroupExists(consumerGroup)

	const allPartitions = -1
	var partitionID int32

	if len(req.Groups[0].Topics) == 0 {
		// An empty request means fetch all topic-partitions for this group.
		partitionID = allPartitions
	} else {
		partitionID = req.Groups[0].Topics[0].Partitions[0]
		assert.Len(c.t, req.Groups[0].Topics, 1, "test only has support for one partition per request")
		assert.Len(c.t, req.Groups[0].Topics[0].Partitions, 1, "test only has support for one partition per request")
	}

	// Prepare the list of partitions for which the offset has been committed.
	// This mimics the real Kafka behaviour.
	var partitionsResp []kmsg.OffsetFetchResponseGroupTopicPartition
	if partitionID == allPartitions {
		for i := 0; i < c.numPartitions; i++ {
			if c.committedOffsets[consumerGroup][i] >= 0 {
				partitionsResp = append(partitionsResp, kmsg.OffsetFetchResponseGroupTopicPartition{
					Partition: int32(i),
					Offset:    c.committedOffsets[consumerGroup][i],
				})
			}
		}
	} else {
		if c.committedOffsets[consumerGroup][partitionID] >= 0 {
			partitionsResp = append(partitionsResp, kmsg.OffsetFetchResponseGroupTopicPartition{
				Partition: partitionID,
				Offset:    c.committedOffsets[consumerGroup][partitionID],
			})
		}
	}

	// Prepare the list topics for which there are some committed offsets.
	// This mimics the real Kafka behaviour.
	var topicsResp []kmsg.OffsetFetchResponseGroupTopic
	if len(partitionsResp) > 0 {
		topicsResp = []kmsg.OffsetFetchResponseGroupTopic{
			{
				Topic:      c.topic,
				Partitions: partitionsResp,
			},
		}
	}

	resp := kreq.ResponseKind().(*kmsg.OffsetFetchResponse)
	resp.Default()
	resp.Groups = []kmsg.OffsetFetchResponseGroup{
		{
			Group:  consumerGroup,
			Topics: topicsResp,
		},
	}
	return resp, nil, true
}
