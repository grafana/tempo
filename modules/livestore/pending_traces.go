package livestore

import (
	"sync"

	"github.com/grafana/tempo/pkg/livetraces"
	trace_v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

// pendingTrace holds a cut trace waiting to be flushed to a complete block.
type pendingTrace struct {
	ID      []byte
	Batches []*trace_v1.ResourceSpans
}

// pendingTraces is a thread-safe buffer for traces that have been cut from
// liveTraces but not yet written to a complete block. It replaces the
// headBlock + walBlocks in the WAL-less architecture.
type pendingTraces struct {
	mu     sync.RWMutex
	traces []*pendingTrace
	sz     uint64
}

func newPendingTraces() *pendingTraces {
	return &pendingTraces{}
}

// Add appends cut live traces to the pending buffer.
func (p *pendingTraces) Add(cut []*livetraces.LiveTrace[*trace_v1.ResourceSpans]) {
	if len(cut) == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range cut {
		pt := &pendingTrace{
			ID:      t.ID,
			Batches: t.Batches,
		}
		p.traces = append(p.traces, pt)
		for _, batch := range t.Batches {
			p.sz += uint64(batch.Size())
		}
	}
}

// Size returns the approximate size of all pending traces in bytes.
func (p *pendingTraces) Size() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sz
}

// Len returns the number of pending traces.
func (p *pendingTraces) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.traces)
}

// CutForBlock atomically removes and returns all pending traces, resetting
// the buffer. The caller owns the returned slice and can process it without
// holding any lock.
func (p *pendingTraces) CutForBlock() []*pendingTrace {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.traces) == 0 {
		return nil
	}

	result := p.traces
	p.traces = nil
	p.sz = 0
	return result
}

// Snapshot returns a copy of the current pending traces for read-only use
// (e.g., querying). The returned slice is safe to iterate without locks.
func (p *pendingTraces) Snapshot() []*pendingTrace {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.traces) == 0 {
		return nil
	}

	snapshot := make([]*pendingTrace, len(p.traces))
	copy(snapshot, p.traces)
	return snapshot
}
