// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/cenkalti/backoff/v4"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type partitionPauseReason uint32

const (
	partitionPauseBackpressure partitionPauseReason = 1 << iota
	partitionPauseRewind
)

// partitionBatchResult describes the rewind work produced while processing one
// fetched partition batch.
type partitionBatchResult struct {
	rewindRecord *kgo.Record
	terminal     bool
}

// pc represents the partition consumer shared information.
type pc struct {
	logger *zap.Logger
	attrs  attribute.Set

	ctx    context.Context
	cancel context.CancelCauseFunc
	// Not safe for concurrent use, this field is never accessed concurrently.
	backOff *backoff.ExponentialBackOff

	mailbox *partitionMailbox
	// pauseReasons stores a bitmask of partitionPauseReason values.
	pauseReasons atomic.Uint32

	// mu prevents cancellation from racing a new wg.Add.
	mu sync.RWMutex
	// wg tracks the number of in-flight message processing goroutines for this
	// partition. New goroutines must call add() so none are added after the
	// partition consumer starts stopping.
	wg sync.WaitGroup
}

// add increments the wait group counter if the partition consumer is not
// stopping. It returns true if the counter was incremented, false otherwise.
func (p *pc) add() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.ctx.Err() != nil {
		return false
	}
	p.wg.Add(1)
	return true
}

// cancelContext cancels the partition consumer context while holding the write
// lock.
func (p *pc) cancelContext(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cancel(err)
}

// addPauseReason records why fetching must remain paused.
func (p *pc) addPauseReason(reason partitionPauseReason) {
	p.pauseReasons.Or(uint32(reason))
}

// clearPauseReasons reports whether the cleared reasons were the final active
// reasons, in which case the caller must resume fetching.
func (p *pc) clearPauseReasons(reasons partitionPauseReason) bool {
	mask := uint32(reasons)
	previous := p.pauseReasons.And(^mask)
	remaining := previous &^ mask
	return previous&mask != 0 && remaining == 0
}

// runPartitionWorker processes exactly one partition in offset order. Workers
// are independent, so downstream backpressure on one partition does not stop
// the shared poll loop or workers for other partitions.
func (c *franzConsumer) runPartitionWorker(pc *pc, tp topicPartition) {
	defer pc.wg.Done()
	partition := map[string][]int32{tp.topic: {tp.partition}}
	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-pc.mailbox.notify:
		}

		for {
			if pc.ctx.Err() != nil {
				return
			}
			batch, ok := pc.mailbox.dequeue(func() {
				if pc.clearPauseReasons(partitionPauseBackpressure) {
					c.client.ResumeFetchPartitions(partition)
				}
			})
			if !ok {
				if !pc.mailbox.hasPendingOffsetChange() {
					break
				}
				if !c.applyMailboxRewind(pc, tp, partition) {
					return
				}
				break
			}

			result := c.processPartitionBatch(pc.ctx, pc, batch)
			if result.rewindRecord != nil {
				pc.mailbox.requestRewind(result.rewindRecord, true, func() {
					pc.addPauseReason(partitionPauseRewind)
					c.client.PauseFetchPartitions(partition)
				})
			}
			if result.rewindRecord != nil {
				if !c.applyMailboxRewind(pc, tp, partition) {
					return
				}
				break
			}
			if result.terminal {
				pc.cancelContext(errors.New("stopping processing: terminal partition error"))
				pc.mailbox.discard()
				return
			}
		}
	}
}

// applyMailboxRewind asks the poll loop to apply the pending rewind, then
// resumes fetching. Returns false if this partition or the receiver is stopping.
func (c *franzConsumer) applyMailboxRewind(pc *pc, tp topicPartition, partition map[string][]int32) bool {
	if !c.sendControl(partitionControl{tp: tp, pc: pc}) {
		return false
	}
	pc.mailbox.resumeAfterOffsetChange(func() {
		if pc.clearPauseReasons(partitionPauseRewind | partitionPauseBackpressure) {
			c.client.ResumeFetchPartitions(partition)
		}
	})
	return true
}

// processPartitionBatch processes one partition batch in both legacy and
// independent modes.
//
// Marking and committing depend on the processing mode:
//   - Legacy with autocommit enabled marks before or after processing according
//     to MessageMarking.After. franz-go commits the marks periodically.
//   - Independent with autocommit enabled behaves the same after a worker
//     dequeues the batch. Records still waiting in the mailbox are not marked.
//   - Legacy with autocommit disabled marks records here. consume commits all
//     marked partitions after the fetched batch finishes.
func (c *franzConsumer) processPartitionBatch(ctx context.Context, pc *pc, p kgo.FetchTopicPartition) partitionBatchResult {
	var fatalRecord *kgo.Record
	fatalIsPermanent := false
	var lastProcessed *kgo.Record
	for _, msg := range p.Records {
		if !c.config.MessageMarking.After {
			c.markCommitRecords(pc, p.Topic, p.Partition, msg)
		}
		c.telemetryBuilder.KafkaReceiverCurrentOffset.Record(ctx, msg.Offset, metric.WithAttributeSet(pc.attrs))
		if err := c.handleMessage(pc, msg); err != nil {
			if pc.ctx.Err() != nil {
				pc.logger.Debug("message processing interrupted",
					zap.Error(err),
					zap.Int64("offset", msg.Offset),
				)
			} else {
				pc.logger.Error("unable to process message",
					zap.Error(err),
					zap.Int64("offset", msg.Offset),
				)
			}
			// handleMessage only returns an error when After=true and
			// the message should not be marked, so checking !shouldMark
			// here is consistent with that contract.
			isPermanent := consumererror.IsPermanent(err)
			shouldMark := (!isPermanent && c.config.MessageMarking.OnError) || (isPermanent && c.config.MessageMarking.OnPermanentError)
			if !shouldMark {
				fatalRecord = msg
				fatalIsPermanent = isPermanent
				break
			}
		}
		lastProcessed = msg
	}

	result := partitionBatchResult{}
	if fatalRecord != nil {
		switch {
		case c.config.ErrorBackOff.Enabled && !fatalIsPermanent:
			// Skip rewind if the consumer is shutting down or the partition was
			// lost. In these cases the error is from context cancellation, not
			// a real processing failure, and calling SetOffsets could interfere
			// with the final offset commit.
			select {
			case <-pc.ctx.Done():
				result.terminal = true
			case <-c.closing:
				result.terminal = true
			default:
				if c.config.PartitionProcessing.Independent {
					result.rewindRecord = fatalRecord
				} else {
					// PollRecords is blocked on wg.Wait() until this goroutine
					// finishes, so SetOffsets is enough. The next poll retries
					// the failed record without a pause.
					c.client.SetOffsets(map[string]map[int32]kgo.EpochOffset{
						p.Topic: {p.Partition: {
							Epoch:  fatalRecord.LeaderEpoch,
							Offset: fatalRecord.Offset,
						}},
					})
				}
				pc.logger.Info("rewinding partition to retry failed record on next poll",
					zap.Int64("offset", fatalRecord.Offset),
				)
			}
		default:
			// No pause reason is needed because this worker exits permanently.
			// On reassignment, a new partition consumer resumes fetching with
			// no pause reasons.
			c.mu.RLock()
			current := c.assignments[topicPartition{topic: p.Topic, partition: p.Partition}]
			// A worker can pause only while it owns the assignment and remains active.
			canPause := current == pc && pc.ctx.Err() == nil
			if canPause {
				// Pause to prevent later records from passing the failed,
				// unmarked record.
				c.client.PauseFetchPartitions(map[string][]int32{p.Topic: {p.Partition}})
			}
			c.mu.RUnlock()
			if !canPause {
				// Exit without pausing because a replacement worker can now own
				// the partition.
				result.terminal = true
				break
			}
			if fatalIsPermanent {
				pc.logger.Error("pausing partition due to permanent processing error, partition will remain paused until rebalance",
					zap.Int64("offset", fatalRecord.Offset),
				)
			} else {
				pc.logger.Error("pausing partition due to processing error (no backoff configured), partition will remain paused until rebalance",
					zap.Int64("offset", fatalRecord.Offset),
				)
			}
			result.terminal = true
		}
	}
	if lastProcessed == nil {
		return result
	}
	c.telemetryBuilder.KafkaReceiverOffsetLag.Record(
		ctx,
		(p.HighWatermark-1)-lastProcessed.Offset,
		metric.WithAttributeSet(pc.attrs),
	)
	if c.config.MessageMarking.After {
		// Mark the latest accepted record after processing. This also covers
		// every earlier record in the batch.
		c.markCommitRecords(pc, p.Topic, p.Partition, lastProcessed)
	}
	return result
}

// markCommitRecords marks records for later commit. Independent mode also
// checks assignment ownership so a timed-out worker cannot mark after a
// replacement starts. Marks are keyed by topic-partition.
func (c *franzConsumer) markCommitRecords(pc *pc, topic string, partition int32, records ...*kgo.Record) {
	if c.config.PartitionProcessing.Independent {
		c.mu.RLock()
		defer c.mu.RUnlock()
		if c.assignments[topicPartition{topic: topic, partition: partition}] != pc {
			return
		}
	}
	c.client.MarkCommitRecords(records...)
}
