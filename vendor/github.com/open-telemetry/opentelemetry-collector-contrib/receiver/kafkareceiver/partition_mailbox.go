// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"slices"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
)

// partitionMailbox stores fetched batches waiting for one partition worker.
// It also coordinates rewinds so fetched batches are not skipped.
type partitionMailbox struct {
	ctx context.Context

	mu sync.Mutex
	// batches is the bounded FIFO queue of batches waiting for the worker.
	batches []kgo.FetchTopicPartition
	// notify wakes the worker when work or a rewind is available. Its single
	// buffered signal represents the state change, not each individual batch.
	notify chan struct{}
	// pendingRewind is the earliest offset that the poll loop must fetch again.
	pendingRewind *kgo.EpochOffset
}

// newPartitionMailbox creates a mailbox with capacity waiting batch slots.
func newPartitionMailbox(ctx context.Context, capacity int) *partitionMailbox {
	return &partitionMailbox{
		ctx:     ctx,
		batches: make([]kgo.FetchTopicPartition, 0, capacity),
		notify:  make(chan struct{}, 1),
	}
}

// enqueue adds a batch when the mailbox can accept it.
func (m *partitionMailbox) enqueue(batch kgo.FetchTopicPartition, pause func(partitionPauseReason)) bool {
	if m.ctx.Err() != nil {
		// The next partition owner will fetch again from the committed offset.
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// PollRecords may return work while a rewind is pending or after the mailbox
	// reaches capacity. Remember the earliest rejected record so it is fetched again.
	if m.pendingRewind != nil || len(m.batches) == cap(m.batches) {
		m.requestRewindLocked(batch.Records[0], false)
		pause(partitionPauseRewind)
		return false
	}

	m.batches = append(m.batches, batch)
	m.notifyLocked()
	// Pause when the final slot is filled so franz-go discards any additional
	// records it buffered for this partition before another poll returns them.
	if len(m.batches) == cap(m.batches) {
		pause(partitionPauseBackpressure)
	}
	return true
}

// dequeue removes and returns the oldest waiting batch.
func (m *partitionMailbox) dequeue(resume func()) (kgo.FetchTopicPartition, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.batches) == 0 {
		return kgo.FetchTopicPartition{}, false
	}

	batch := m.batches[0]
	m.batches = slices.Delete(m.batches, 0, 1)
	// A pending rewind must move the fetch position before fetching resumes.
	// Calling resume under the lock also prevents it from overtaking an older
	// pause operation.
	if m.pendingRewind == nil {
		resume()
	}
	return batch, true
}

// requestRewind records the earliest offset that must be fetched again,
// optionally discards queued batches, and pauses fetching.
func (m *partitionMailbox) requestRewind(record *kgo.Record, clearQueue bool, pause func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestRewindLocked(record, clearQueue)
	pause()
}

// requestRewindLocked updates rewind state while the caller holds m.mu.
func (m *partitionMailbox) requestRewindLocked(record *kgo.Record, clearQueue bool) {
	if clearQueue {
		clear(m.batches)
		m.batches = m.batches[:0]
	}
	// Multiple rejected fetches may arrive before the poll loop applies the
	// rewind. Keep the earliest offset so no rejected record is skipped.
	if m.pendingRewind == nil || record.Offset < m.pendingRewind.Offset {
		m.pendingRewind = &kgo.EpochOffset{
			Epoch:  record.LeaderEpoch,
			Offset: record.Offset,
		}
	}
	m.notifyLocked()
}

// hasPendingOffsetChange reports whether the fetch position must change.
func (m *partitionMailbox) hasPendingOffsetChange() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pendingRewind != nil
}

// takeOffsetChange returns and clears the pending rewind.
func (m *partitionMailbox) takeOffsetChange() *kgo.EpochOffset {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pendingRewind == nil {
		return nil
	}
	offset := m.pendingRewind
	m.pendingRewind = nil
	return offset
}

// resumeAfterOffsetChange serializes a resume with mailbox pause callbacks.
func (m *partitionMailbox) resumeAfterOffsetChange(resume func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	resume()
}

// discard releases all queued and pending work.
func (m *partitionMailbox) discard() {
	m.mu.Lock()
	defer m.mu.Unlock()

	clear(m.batches)
	m.batches = m.batches[:0]
	m.pendingRewind = nil
}

// notifyLocked wakes the worker without blocking while the caller holds m.mu.
func (m *partitionMailbox) notifyLocked() {
	select {
	case m.notify <- struct{}{}:
	default:
	}
}
