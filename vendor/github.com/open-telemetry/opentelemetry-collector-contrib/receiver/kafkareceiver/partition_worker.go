// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/otel/attribute"
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

	mailbox *partitionMailbox
	// skipConsume holds offsets this worker already finished above a rewind
	// hole. It stays nil until a rewind writes it. Entries drop when the
	// marked prefix advances past them, and the map dies with pc on rebalance.
	skipConsume map[int64]struct{}
	// window tracks the records in flight when max_in_flight.records is above
	// 1, across fetches. Only the worker goroutine sets it.
	window *inflightWindow
	// pauseReasons stores a bitmask of partitionPauseReason values.
	pauseReasons atomic.Uint32

	// offsetLag holds the last reported offset lag
	offsetLag atomic.Int64
	// offsetLagReportable flag to track when an active partition has a reportable offset lag
	offsetLagReportable atomic.Bool

	// currentOffset holds the offset of the last record handed to message processing
	currentOffset atomic.Int64
	// currentOffsetReportable flag to track when an active partition has a reportable current offset
	currentOffsetReportable atomic.Bool

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

// logProcessError reports a record the pipeline rejected. A cancelled partition
// is an expected shutdown path, so it logs at debug level.
func (p *pc) logProcessError(record *kgo.Record, err error) {
	fields := []zap.Field{zap.Error(err), zap.Int64("offset", record.Offset)}
	if p.ctx.Err() != nil {
		p.logger.Debug("message processing interrupted", fields...)
		return
	}
	p.logger.Error("unable to process message", fields...)
}

// runPartitionWorker processes exactly one partition in offset order. Workers
// are independent, so downstream backpressure on one partition does not stop
// the shared poll loop or workers for other partitions.
func (c *franzConsumer) runPartitionWorker(pc *pc, tp topicPartition) {
	defer pc.wg.Done()
	// Nil on the sequential path, meaning it never fires in the select below.
	var windowFailed <-chan struct{}
	if c.maxInFlight() > 1 {
		pc.window = newInflightWindow(c, pc)
		windowFailed = pc.window.failedCh
		// This runs before pc.wg.Done, so revoke and shutdown wait for every call
		// the window started, whichever fetch it came from.
		defer pc.window.wg.Wait()
	}
	partition := map[string][]int32{tp.topic: {tp.partition}}
	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-pc.mailbox.notify:
		case <-windowFailed:
		}

		for {
			if pc.ctx.Err() != nil {
				return
			}
			var batch kgo.FetchTopicPartition
			ok := true
			if pc.window != nil && pc.window.hasFailed() {
				// A record failed, so stop taking new batches. Processing an empty
				// batch waits for the window to drain, then rewinds or pauses at the
				// failed record.
				batch = kgo.FetchTopicPartition{
					Topic:          tp.topic,
					FetchPartition: kgo.FetchPartition{Partition: tp.partition},
				}
			} else {
				batch, ok = pc.mailbox.dequeue(func() {
					if pc.clearPauseReasons(partitionPauseBackpressure) {
						c.client.ResumeFetchPartitions(partition)
					}
				})
			}
			if !ok {
				if !pc.mailbox.hasPendingOffsetChange() {
					break
				}
				if !c.applyMailboxRewind(pc, tp, partition) {
					return
				}
				break
			}

			result := c.processPartitionBatch(pc, batch)
			if result.rewindRecord != nil {
				pc.mailbox.requestRewind(result.rewindRecord, true, func() {
					pc.addPauseReason(partitionPauseRewind)
					c.client.PauseFetchPartitions(partition)
				})
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
//   - Independent with max_in_flight above 1 always marks after processing,
//     because concurrent calls have no single record to mark before.
//     processRecordsConcurrent also marks the contiguous prefix as soon as
//     earlier in-flight records finish. With window cross_fetch it returns
//     once the batch's records have started, and their calls keep running in
//     the partition window.
func (c *franzConsumer) processPartitionBatch(pc *pc, p kgo.FetchTopicPartition) partitionBatchResult {
	var fatalRecord *kgo.Record
	fatalIsPermanent := false
	var lastProcessed *kgo.Record
	concurrent := c.maxInFlight() > 1
	if concurrent {
		fatalRecord, fatalIsPermanent = c.processRecordsConcurrent(pc, p)
	} else {
		for _, msg := range p.Records {
			// Stop before marking once the partition consumer is cancelled. The
			// records left here are still committable, and lost() commits the marks,
			// so marking one now drops it: nothing processed it and nothing
			// redelivers it. Leaving them unmarked hands them to the next owner
			// after a revocation, or to the next run after a shutdown.
			//
			// This also keeps the wait in lost() down to the in-flight record
			// instead of the whole batch, and that wait has to fit in the re-balance
			// timeout.
			//
			// break, not return, so processed records still get their After mark and
			// lag telemetry below.
			if pc.ctx.Err() != nil {
				pc.logger.Debug("stopped processing records, leaving the rest of the batch unmarked",
					zap.Int64("offset", msg.Offset),
				)
				break
			}
			if !c.config.MessageMarking.After {
				c.markCommitRecords(pc, p.Topic, p.Partition, msg)
			}
			// Record the current consumer offset.
			pc.currentOffset.Store(msg.Offset)
			if err := c.handleMessage(pc, msg); err != nil {
				pc.logProcessError(msg, err)
				fatalRecord = msg
				fatalIsPermanent = consumererror.IsPermanent(err)
				break
			}
			lastProcessed = msg
		}
	}

	result := partitionBatchResult{}
	terminallyPaused := false
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

				// Stop reporting lag and current offset because this partition
				// cannot resume without reassignment.
				pc.offsetLagReportable.Store(false)
				pc.currentOffsetReportable.Store(false)
				terminallyPaused = true
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

	if len(p.Records) > 0 && !terminallyPaused {
		// Every loop iteration stores the record offset before any break, so
		// currentOffset is set before the flag becomes observable.
		pc.currentOffsetReportable.Store(true)
	}

	if lastProcessed == nil {
		// no messages were processed, return early. The concurrent path
		// reports lag itself, each time the mark moves.
		return result
	}

	// Record the current consumer lag.
	// Skip for terminally paused partitions. Reporting resumes after rebalance.
	if !terminallyPaused {
		pc.offsetLag.Store((p.HighWatermark - 1) - lastProcessed.Offset)
		pc.offsetLagReportable.Store(true)
	}

	if c.config.MessageMarking.After && !concurrent {
		// Mark the latest accepted record after processing. This also covers
		// every earlier record in the batch. The concurrent path marks as it goes.
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

// maxInFlight returns how many handleMessage calls one partition worker may run
// at once. Only independent workers go above 1.
func (c *franzConsumer) maxInFlight() int {
	if c.config.PartitionProcessing.Independent {
		return c.config.PartitionProcessing.MaxInFlight.Records
	}
	return 1
}

// processRecordsConcurrent starts a handleMessage call for each record of the
// fetch in the partition window. The window allows max_in_flight.records calls
// and max_in_flight.bytes bytes, counted across fetches. As records finish, it
// marks up to the first record that isn't accepted yet, in fetch order. A
// failure that message marking is configured to skip counts as accepted.
//
// When a record fails and isn't skipped, it starts no new calls, waits for the
// ones running, and returns the failed record and whether the error is
// permanent. Before a rewind it remembers which records above the failed one
// already finished, so the next fetch doesn't send them to Consume again. With
// window per_fetch it also waits for the window when nothing failed, so calls
// never span fetches.
func (c *franzConsumer) processRecordsConcurrent(pc *pc, p kgo.FetchTopicPartition) (fatalRecord *kgo.Record, fatalIsPermanent bool) {
	w := pc.window
	for _, msg := range p.Records {
		if pc.ctx.Err() != nil {
			break
		}
		pc.currentOffset.Store(msg.Offset)
		size := inflightSize(msg)
		if _, ok := pc.skipConsume[msg.Offset]; ok {
			w.complete(w.push(msg, p.HighWatermark, size), nil)
			continue
		}
		if !w.acquire(size) {
			break
		}
		w.start(w.push(msg, p.HighWatermark, size))
	}
	// Only this goroutine starts calls, so once the wait returns nothing is
	// running and failed can't change. If a call fails right after the check,
	// failedCh wakes the worker and the next pass handles it.
	waited := c.config.PartitionProcessing.MaxInFlight.Window != InFlightWindowCrossFetch || w.hasFailed()
	if waited {
		w.wg.Wait()
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if waited && w.failed {
		// After wg.Wait every call is done and its writes are visible here. The
		// first entry is the record the mark can't pass.
		hole := w.entries[0]
		fatalRecord, fatalIsPermanent = hole.rec, consumererror.IsPermanent(hole.err)
		pc.logProcessError(fatalRecord, hole.err)
		// Only a rewind on this pc retries the hole. A pause or a cancelled
		// partition hands the offsets to a new owner, which must Consume them.
		if c.config.ErrorBackOff.Enabled && !fatalIsPermanent && pc.ctx.Err() == nil {
			w.rememberFinished()
		}
		w.reset()
	}
	pc.dropSkipConsume(w.lastMarked)
	return fatalRecord, fatalIsPermanent
}

// recordOverhead and headerOverhead approximate what a held record costs
// besides its payload: the kgo.Record (160 bytes on 64-bit), its inflightEntry
// and slot in entries (56), and 40 per kgo.RecordHeader. Counting them keeps
// max_in_flight.bytes a bound when records are tiny or empty. They are rounded
// estimates rather than unsafe.Sizeof, since allocator rounding and the shared
// decompressed buffer make the real cost approximate anyway.
const (
	recordOverhead = 256
	headerOverhead = 40
)

// inflightSize returns the bytes rec holds against max_in_flight.bytes: its
// key, value, and headers, plus the per-record and per-header structs.
func inflightSize(rec *kgo.Record) int {
	size := recordOverhead + len(rec.Key) + len(rec.Value)
	for _, h := range rec.Headers {
		size += headerOverhead + len(h.Key) + len(h.Value)
	}
	return size
}

// inflightEntry is one started record in the partition window.
type inflightEntry struct {
	rec *kgo.Record
	// hw is the high watermark of the fetch the record came from.
	hw int64
	// size is the bytes this record holds against max_in_flight.bytes until
	// the mark passes it, see inflightSize. A replay that skips Consume
	// holds them too.
	size int
	done bool
	// err is set when the record failed and message marking doesn't skip the
	// failure. The mark can't move past it.
	err error
}

// inflightWindow holds the in-flight calls of one partition across fetches.
type inflightWindow struct {
	c  *franzConsumer
	pc *pc
	// wg tracks every call the window started.
	wg sync.WaitGroup
	// failedCh wakes an idle worker after the first failure, so it rewinds
	// even when no new batch arrives.
	failedCh chan struct{}

	mu sync.Mutex
	// room is signaled when a call finishes. Only the goroutine that starts
	// calls waits on it, so Signal is enough.
	room *sync.Cond
	// inflight counts the calls between acquire and the call returning.
	// inflightBytes counts the bytes of the records between acquire and the mark
	// passing them, finished or not.
	inflight      int
	inflightBytes int
	// entries holds the started records the mark hasn't passed yet, in fetch
	// order. Offsets can have gaps, from compaction or transaction markers, so
	// marking walks this list instead of counting offset + 1.
	entries []*inflightEntry
	// failed stops new calls once any record leaves a hole.
	failed bool
	// lastMarked is the offset of the last record marked, or -1. It keeps the
	// offset rather than the record, which would keep its whole decompressed
	// record batch in memory while the partition is idle.
	lastMarked int64
}

func newInflightWindow(c *franzConsumer, pc *pc) *inflightWindow {
	w := &inflightWindow{c: c, pc: pc, failedCh: make(chan struct{}, 1), lastMarked: -1}
	w.room = sync.NewCond(&w.mu)
	return w
}

func (w *inflightWindow) hasFailed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.failed
}

// push appends rec to the FIFO and counts its size bytes until the mark passes
// it. Only the worker goroutine calls it.
func (w *inflightWindow) push(rec *kgo.Record, hw int64, size int) *inflightEntry {
	e := &inflightEntry{rec: rec, hw: hw, size: size}
	w.mu.Lock()
	w.entries = append(w.entries, e)
	w.inflightBytes += size
	w.mu.Unlock()
	return e
}

// acquire waits until a record of size bytes fits under both max_in_flight
// limits. It returns false when a record has failed or the partition context is
// done, so a revoked partition does not start another Consume after this wait.
func (w *inflightWindow) acquire(size int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	// A waiter is woken by a call finishing, so it sees a cancelled context at
	// the next release.
	for {
		if w.failed || w.pc.ctx.Err() != nil {
			return false
		}
		if w.hasRoom(size) {
			break
		}
		w.room.Wait()
	}
	w.inflight++
	return true
}

// hasRoom reports whether a record of size bytes fits under both max_in_flight
// limits. The caller must hold w.mu.
func (w *inflightWindow) hasRoom(size int) bool {
	cfg := w.c.config.PartitionProcessing.MaxInFlight
	if w.inflight >= cfg.Records {
		return false
	}
	// A window holding no bytes always admits one record, so a record above
	// bytes runs alone instead of blocking the partition.
	return cfg.Bytes == 0 || w.inflightBytes == 0 || w.inflightBytes+size <= cfg.Bytes
}

// start runs handleMessage for e on its own goroutine. The caller must have
// called acquire.
func (w *inflightWindow) start(e *inflightEntry) {
	w.wg.Go(func() {
		w.complete(e, w.c.handleMessage(w.pc, e.rec))
		// Free the slot after complete, so a worker woken in acquire already
		// sees failed. The bytes stay held until the mark passes the record.
		w.mu.Lock()
		w.inflight--
		w.mu.Unlock()
		w.room.Signal()
	})
}

// complete records that e finished, with err set when message marking doesn't
// skip the failure. It moves the mark over every finished record at the front,
// frees their bytes, and reports lag when the mark moves.
func (w *inflightWindow) complete(e *inflightEntry, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e.done, e.err = true, err
	if err != nil && !w.failed {
		w.failed = true
		select {
		case w.failedCh <- struct{}{}:
		default:
		}
	}
	var last *inflightEntry
	for len(w.entries) > 0 && w.entries[0].done && w.entries[0].err == nil {
		last = w.entries[0]
		w.inflightBytes -= last.size
		// Drop the pointer so a marked record's payload can be collected.
		w.entries[0] = nil
		w.entries = w.entries[1:]
	}
	if last == nil {
		return
	}
	w.lastMarked = last.rec.Offset
	w.c.markCommitRecords(w.pc, last.rec.Topic, last.rec.Partition, last.rec)
	w.pc.offsetLag.Store((last.hw - 1) - last.rec.Offset)
	w.pc.offsetLagReportable.Store(true)
}

// rememberFinished adds the records that finished above the failed one to
// pc.skipConsume, so the replay doesn't send them to Consume again. The caller
// holds mu and the window has drained.
func (w *inflightWindow) rememberFinished() {
	for _, e := range w.entries[1:] {
		if !e.done || e.err != nil {
			continue
		}
		if w.pc.skipConsume == nil {
			w.pc.skipConsume = make(map[int64]struct{})
		}
		w.pc.skipConsume[e.rec.Offset] = struct{}{}
	}
}

// reset empties a drained window after a failure was handled. The caller holds mu.
func (w *inflightWindow) reset() {
	clear(w.entries)
	w.entries = nil
	// The mark never passed the failed record, so it and every record behind it
	// still hold their bytes. Drop them all at once.
	w.inflightBytes = 0
	w.failed = false
	select {
	case <-w.failedCh:
	default:
	}
}

// dropSkipConsume removes skip entries at or below the last marked offset. The
// map returns to nil when it is empty so later success batches do not keep a set.
func (p *pc) dropSkipConsume(marked int64) {
	if p.skipConsume == nil {
		return
	}
	for off := range p.skipConsume {
		if off <= marked {
			delete(p.skipConsume, off)
		}
	}
	if len(p.skipConsume) == 0 {
		p.skipConsume = nil
	}
}
