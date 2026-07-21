// This file implements the fixed-size worker pool that drains
// Consumer.Records() (consumer.go) into events.go's Applier -- DESIGN.md §
// Event processing: "A worker pool (16 at reference sizing) drains a
// bounded in-memory queue fed by the consumer." Plain goroutines over a
// channel, NOT dskit/concurrency.ReusableGoroutinesPool (§0 Kafka-plumbing
// decision: that pool's fixed workers overflow to unbounded extra
// goroutines once busy, the wrong shape for a hard cap).
package bloomgateway

import (
	"context"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// WorkerPool applies records from a Records() channel (consumer.go) via
// Applier, tracking the per-partition contiguous "applied" watermark that
// backs AppliedOffsets -- the snapshot's actual resume authority (§
// Snapshots), deliberately distinct from Consumer's own (generally further
// ahead) fetch position.
type WorkerPool struct {
	n        int
	upstream <-chan Record // fed by Consumer (consumer.go)
	staged   chan Record   // fed by dispatchLoop; drained by the n worker goroutines
	apply    *Applier
	logger   log.Logger
	m        *metrics

	// pauseMu is the snapshot consistency pause's own mechanism (DESIGN.md §
	// Snapshots consistency, §0 D9; added by the top-level orchestrator,
	// bloomgateway.go -- this WP is its only caller). Every worker holds a
	// read lock for the duration of one record's decode+apply (process,
	// below); Pause takes the write lock, which by Go's own sync.RWMutex
	// contract blocks until every CURRENTLY in-flight record finishes
	// (never returns early leaving one mid-apply) and then blocks every
	// FURTHER record until Resume -- exactly the quiescent window
	// Snapshotter.Save needs to read the live directory/registry/tenant
	// structures directly (no extra copying).
	pauseMu sync.RWMutex

	progressMu sync.Mutex
	progress   map[int32]*partitionProgress

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewWorkerPool builds a pool of n fixed workers over records. Start must
// be called to actually begin processing.
func NewWorkerPool(n int, records <-chan Record, apply *Applier, logger log.Logger, m *metrics) *WorkerPool {
	return &WorkerPool{
		n:        n,
		upstream: records,
		staged:   make(chan Record),
		apply:    apply,
		logger:   logger,
		m:        m,
		progress: make(map[int32]*partitionProgress),
	}
}

// Start launches the single dispatch-loop goroutine plus n fixed worker
// goroutines, all running until upstream is closed or ctx (via Stop, or the
// ctx passed here) is done.
func (p *WorkerPool) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.dispatchLoop(runCtx)
	}()

	for i := 0; i < p.n; i++ {
		p.wg.Add(1)
		go p.worker(runCtx)
	}
}

// Stop signals the dispatch loop and every worker to exit and waits for
// them to drain. Safe to call even if the underlying Consumer is still
// producing records -- everything simply stops pulling once ctx is done.
func (p *WorkerPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

// Pause blocks until every currently in-flight record finishes processing,
// then prevents any FURTHER record from being applied until Resume is
// called -- the snapshot consistency mechanism (DESIGN.md § Snapshots
// consistency: "v1 pauses the worker pool between events for the duration
// of the serialization"; §0 D9). The orchestrator's own snapshot ticker is
// this method's only caller, in the sequence Pause -> Snapshotter.Save ->
// Resume: Save reads the live directory/registry/tenant structures
// directly (no extra copying, matching e.g. Directory.Leaf's own "safe for
// a caller that has already paused all mutation" contract), which is only
// safe because Pause is held for Save's ENTIRE duration, not just while
// assembling its input.
//
// Pause does NOT stop the Kafka consumer's fetch loop or dispatchLoop:
// records keep being dequeued from staged into (at most p.n) workers, each
// of which blocks here rather than applying -- once all p.n workers are
// blocked, staged itself backs up, and eventually so does the Consumer's
// own byte-bounded queue. This is the documented, expected cost ("the
// consumer buffers and lag blips"), not a bug.
func (p *WorkerPool) Pause() {
	p.pauseMu.Lock()
}

// Resume reverses Pause, letting every worker resume applying staged
// records. Calling Resume without a preceding Pause panics (sync.RWMutex's
// own contract for an unlock of a lock not held) -- deliberately not
// guarded against here, since a caller invoking Resume out of sequence
// with Pause is a programming error this WP's own single caller (the
// snapshot ticker) never makes, not a runtime condition to tolerate
// silently.
func (p *WorkerPool) Resume() {
	p.pauseMu.Unlock()
}

// dispatchLoop is the ONLY goroutine that ever reads from upstream.
// Running single-threaded here (rather than having every worker read
// upstream directly) is what makes per-partition dispatch-order tracking
// (seed, below) race-free: a Go channel guarantees receives observe send
// order regardless of how many goroutines are receiving, but the CODE a
// receiving goroutine runs afterward is not itself cross-goroutine
// ordered -- if workers called seed directly after their own receive, a
// goroutine that dequeued an EARLIER offset could still be preempted
// before recording that fact, letting a goroutine that dequeued a LATER
// offset seed first and silently establish the wrong baseline. Centralizing
// the seed call here, strictly before handing the record on to staged,
// removes that race entirely: this goroutine's own statements execute in
// its own program order, full stop.
func (p *WorkerPool) dispatchLoop(ctx context.Context) {
	defer close(p.staged)
	for {
		select {
		case rec, ok := <-p.upstream:
			if !ok {
				return
			}
			p.seed(rec.Partition, rec.Offset)
			select {
			case p.staged <- rec:
			case <-ctx.Done():
				if rec.release != nil {
					rec.release()
				}
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case rec, ok := <-p.staged:
			if !ok {
				return
			}
			p.process(rec)
		case <-ctx.Done():
			return
		}
	}
}

// process decodes and applies one record, then marks its offset applied
// regardless of outcome -- a decode/apply failure is a producer-side
// problem (drop and count), not a transient condition to retry, and
// leaving it unmarked would stall AppliedOffsets' contiguous watermark
// forever for that partition (events.go's own DecodeEvent/Apply contract:
// "a non-nil error means drop and count ... never partially apply").
//
// rec.release is called first, immediately on dequeue and before any
// decode/apply work -- see Record's doc comment in consumer.go for why
// that timing is the byte-queue's actual "drain" point.
func (p *WorkerPool) process(rec Record) {
	if rec.release != nil {
		rec.release()
	}
	defer p.markApplied(rec.Partition, rec.Offset)

	// Held only around decode+apply, AFTER release() above: release()
	// frees the byte-bounded queue's admission budget regardless of
	// pausing (consumer.go's own documented "leaves the queue the instant
	// a worker picks it up" contract, unaffected by this WP's addition),
	// so a paused pool still drains staged into up to p.n blocked
	// goroutines before the queue itself backs up -- matching DESIGN.md's
	// own "reads continue; the consumer buffers and lag blips" framing of
	// the pause.
	p.pauseMu.RLock()
	defer p.pauseMu.RUnlock()

	event, err := p.apply.DecodeEvent(rec.Value)
	if err != nil {
		// Decode failures are counted HERE: they happen before the Applier
		// is ever involved, so nothing downstream would otherwise count them.
		p.m.addsTotal.WithLabelValues("dropped").Inc()
		p.dropped(rec, "decode", err)
		return
	}
	if err := p.apply.Apply(event); err != nil {
		// Do NOT increment adds_total here. The per-type appliers own that
		// accounting: ApplyAddChunk already counts adds_total{dropped} for
		// every AddChunk-shaped failure (the dominant apply-drop case), so
		// counting again would double-count exactly those. Rarer apply-stage
		// errors (a Delete with an unparseable block_id, or Apply's
		// nil-submessage/unknown-type dispatch errors) are logged below and
		// remain visible that way; keeping the metric single-owner is worth
		// more than counting those few in adds_total.
		p.dropped(rec, "apply", err)
	}
}

// dropped logs an event this worker could not process to completion, without
// touching adds_total -- counting is owned by process's decode branch and by
// the Applier's own per-type accounting (see process). Never logs rec.Value
// (raw trace-ID bytes) -- only its length, per the no-raw-PII-in-logs rule.
func (p *WorkerPool) dropped(rec Record, stage string, err error) {
	level.Warn(p.logger).Log("msg", "bloomgateway: dropping event", "stage", stage, "partition", rec.Partition, "offset", rec.Offset, "len", len(rec.Value), "err", err)
}

// getOrCreateProgress lazily creates partition's progress tracker on first
// use, shared by both seed (dispatch time) and markApplied (completion
// time).
func (p *WorkerPool) getOrCreateProgress(partition int32) *partitionProgress {
	p.progressMu.Lock()
	defer p.progressMu.Unlock()
	pp, ok := p.progress[partition]
	if !ok {
		pp = &partitionProgress{}
		p.progress[partition] = pp
	}
	return pp
}

// seed establishes partition's baseline watermark from the first record
// ever DISPATCHED for it this session. Called exactly once per record, in
// strict dispatch order, from dispatchLoop -- never from a worker
// goroutine; see dispatchLoop's doc comment for why that matters.
func (p *WorkerPool) seed(partition int32, offset int64) {
	p.getOrCreateProgress(partition).seed(offset)
}

// markApplied records that rec's offset finished processing (successfully
// or dropped -- see process's doc comment).
func (p *WorkerPool) markApplied(partition int32, offset int64) {
	p.getOrCreateProgress(partition).markApplied(offset)
}

// AppliedOffsets returns, per partition, the next offset to resume from:
// every offset below it has finished processing since this pool started
// (the per-partition CONTIGUOUS applied watermark, the snapshot's
// authoritative resume point, DESIGN.md § Snapshots). It never advances
// past an incomplete predecessor: workers finish out of order (apply
// duration varies with chunk size and lock contention), but a later
// offset's completion is only folded into the watermark once every lower
// offset has also completed (see partitionProgress.markApplied). Before
// anything has completed, a partition's value is its first-dispatched
// offset unchanged -- still a safe, accurate resume point (resuming there
// simply re-fetches and re-applies it, which is idempotent).
//
// A partition absent from the returned map has not had any record
// DISPATCHED to it yet this session -- the caller (snapshot /
// reconstruction orchestrator, out of this WP's scope) is expected to fall
// back to its own last-known offset for such a partition, exactly as it
// would for one this pool has never been told about at all.
func (p *WorkerPool) AppliedOffsets() map[int32]int64 {
	p.progressMu.Lock()
	defer p.progressMu.Unlock()
	out := make(map[int32]int64, len(p.progress))
	for partition, pp := range p.progress {
		out[partition] = pp.get()
	}
	return out
}

// partitionProgress tracks one partition's contiguous-applied watermark
// under out-of-order completion. watermark is the next offset expected to
// complete (equivalently: every offset below it has already completed, and
// nothing below it will ever need to be waited for again); pending holds
// offsets >= watermark that completed before one of their predecessors
// did, waiting to be folded in once the gap closes.
//
// seeded distinguishes "no baseline established yet" from "watermark is
// legitimately 0" -- see seed's doc comment for how and when the baseline
// is established.
type partitionProgress struct {
	mu        sync.Mutex
	seeded    bool
	watermark int64
	pending   map[int64]struct{}
}

// seed establishes watermark as offset -- the first offset ever DISPATCHED
// for this partition this session -- if no baseline exists yet. A no-op on
// every subsequent call. Deliberately separate from, and always called
// before, markApplied for the same offset (WorkerPool.dispatchLoop calls
// this at dequeue time, in strict dispatch order; markApplied is called
// later, at completion time, possibly out of order): seeding from
// DISPATCH order rather than from whichever offset happens to COMPLETE
// first is what makes this correct under concurrent out-of-order
// completion. If seeding instead happened on first completion, a later
// offset finishing before an earlier, still-outstanding one would wrongly
// become the baseline -- permanently hiding the still-incomplete earlier
// offset from the watermark. Sound because Kafka delivers one partition's
// records in strictly increasing offset order starting exactly at the
// requested resume point, so there is, by construction, nothing earlier
// this session that will ever arrive for this pool to wait for.
func (pp *partitionProgress) seed(offset int64) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	if !pp.seeded {
		pp.watermark = offset
		pp.seeded = true
	}
}

// markApplied folds offset into the watermark if it is the next expected
// one (and, recursively, any run of previously-pending offsets that
// contiguously follow it); otherwise it is parked in pending until its
// predecessors close the gap. This is the one piece of logic the
// invariant -- "never advances past an incomplete predecessor" -- actually
// depends on; see worker_test.go's
// TestWorkerPool_AppliedOffsetsNeverSkipsIncompletePredecessor for the
// exhaustive out-of-order-completion regression test.
func (pp *partitionProgress) markApplied(offset int64) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if offset < pp.watermark {
		return // already covered by a prior contiguous advance
	}
	if pp.pending == nil {
		pp.pending = make(map[int64]struct{})
	}
	pp.pending[offset] = struct{}{}
	for {
		if _, ok := pp.pending[pp.watermark]; !ok {
			break
		}
		delete(pp.pending, pp.watermark)
		pp.watermark++
	}
}

func (pp *partitionProgress) get() int64 {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return pp.watermark
}
