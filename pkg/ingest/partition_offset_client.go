// SPDX-License-Identifier: AGPL-3.0-only

package ingest

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	// kafkaOffsetStart is a special offset value that means the beginning of the partition.
	kafkaOffsetStart = int64(-2)

	// kafkaOffsetEnd is a special offset value that means the end of the partition.
	kafkaOffsetEnd = int64(-1)

	// ReaderMetricsPrefix is the reader metrics prefix used by the ingest storage.
	ReaderMetricsPrefix = "cortex_ingest_storage_reader"
)

// partitionOffsetClient is a client used to read partition offsets.
type PartitionOffsetClient struct {
	client *kgo.Client
	admin  *kadm.Client

	topic string
}

func NewPartitionOffsetClient(client *kgo.Client, topic string) *PartitionOffsetClient {
	return &PartitionOffsetClient{
		client: client,
		admin:  kadm.NewClient(client),
		topic:  topic,
	}
}

// FetchPartitionsLastProducedOffsets fetches and returns the last produced offsets for all input partitions. The offset is
// -1 if a partition has been created but no record has been produced yet. The returned offsets for each partition
// are guaranteed to be always updated (no stale or cached offsets returned).
//
// The Kafka client used under the hood may retry a failed request until the retry timeout is hit.
func (p *PartitionOffsetClient) FetchPartitionsLastProducedOffsets(ctx context.Context, partitionIDs []int32) (_ kadm.ListedOffsets, returnErr error) {
	// Skip lookup and don't track any metric if no partition was requested.
	if len(partitionIDs) == 0 {
		return nil, nil
	}
	return p.fetchPartitionsOffsets(ctx, kafkaOffsetEnd, partitionIDs)
}

func (p *PartitionOffsetClient) FetchPartitionsStartProducedOffsets(ctx context.Context, partitionIDs []int32) (_ kadm.ListedOffsets, returnErr error) {
	// Skip lookup and don't track any metric if no partition was requested.
	if len(partitionIDs) == 0 {
		return nil, nil
	}

	return p.fetchPartitionsOffsets(ctx, kafkaOffsetStart, partitionIDs)
}

// fetchPartitionsEndOffsets returns the end offset of each partition in input. This function returns
// error if fails to get the offset of any partition (no partial result is returned).
func (p *PartitionOffsetClient) fetchPartitionsOffsets(ctx context.Context, timestamp int64, partitionIDs []int32) (kadm.ListedOffsets, error) {
	list := kadm.ListedOffsets{
		p.topic: make(map[int32]kadm.ListedOffset, len(partitionIDs)),
	}

	// Prepare the request to list offsets.
	topicReq := kmsg.NewListOffsetsRequestTopic()
	topicReq.Topic = p.topic
	for _, partitionID := range partitionIDs {
		partitionReq := kmsg.NewListOffsetsRequestTopicPartition()
		partitionReq.Partition = partitionID
		partitionReq.Timestamp = timestamp

		topicReq.Partitions = append(topicReq.Partitions, partitionReq)
	}

	req := kmsg.NewPtrListOffsetsRequest()
	req.IsolationLevel = 0 // 0 means READ_UNCOMMITTED.
	req.Topics = []kmsg.ListOffsetsRequestTopic{topicReq}

	// Execute the request.
	shards := p.client.RequestSharded(ctx, req)

	for _, shard := range shards {
		if shard.Err != nil {
			return nil, shard.Err
		}

		res := shard.Resp.(*kmsg.ListOffsetsResponse)
		if len(res.Topics) != 1 {
			return nil, fmt.Errorf("unexpected number of topics in the response (expected: %d, got: %d)", 1, len(res.Topics))
		}
		if res.Topics[0].Topic != p.topic {
			return nil, fmt.Errorf("unexpected topic in the response (expected: %s, got: %s)", p.topic, res.Topics[0].Topic)
		}

		for _, pt := range res.Topics[0].Partitions {
			if err := kerr.ErrorForCode(pt.ErrorCode); err != nil {
				return nil, err
			}

			list[p.topic][pt.Partition] = kadm.ListedOffset{
				Topic:       p.topic,
				Partition:   pt.Partition,
				Timestamp:   pt.Timestamp,
				Offset:      pt.Offset,
				LeaderEpoch: pt.LeaderEpoch,
			}
		}
	}

	return list, nil
}
