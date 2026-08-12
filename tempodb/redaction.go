package tempodb

import (
	"fmt"
	"math"
)

// RedactionWindow bounds the per-block scan of a TraceQL redaction in time: with a window set, only
// traces overlapping it are candidates for removal. Both bounds are unix nanoseconds, and the zero
// value is unbounded — every query match in the block, regardless of timestamp.
//
// Two limits are worth stating because neither is obvious and both destroy data:
//
//   - Overlap, not containment. The installed predicate matches a trace whose range intersects the
//     window, and a matched trace is dropped whole — so a trace starting before the window and ending
//     inside it loses its out-of-window spans too.
//   - Query selector only. The explicit trace-ID path resolves IDs with no time bound, so a window
//     cannot scope it; RedactBlock refuses that combination rather than validating a window it would
//     then ignore.
//
// The bounds travel as a named-field struct rather than two adjacent int64 parameters
// because transposing them fails silently: an inverted window matches nothing, the job
// reports success, and the operator is told the block was processed when nothing was
// removed. On a path that deletes data with no way to recover it, under-deletion reported
// as success is the failure mode with no external signal.
type RedactionWindow struct {
	StartNano int64
	EndNano   int64
}

// IsZero reports whether the window is unbounded, so a caller can skip work that only a bounded
// window needs.
func (w RedactionWindow) IsZero() bool {
	return w.StartNano == 0 && w.EndNano == 0
}

// Validate reports whether the window can be honoured, so a caller refuses it instead of
// scanning with it and reporting the empty result as a completed redaction.
//
// A one-sided window is deliberately accepted: a batch persisted by an older scheduler can
// still carry one, and fetchBounds materialises the open side to keep the scan bounded.
// Rejecting those would turn in-flight batches into permanently failing jobs.
func (w RedactionWindow) Validate() error {
	if w.StartNano < 0 || w.EndNano < 0 {
		return fmt.Errorf("window bounds must be non-negative unix nanoseconds, got start=%d end=%d", w.StartNano, w.EndNano)
	}

	// Only meaningful once both bounds are set; a one-sided window has nothing to order against.
	if w.StartNano > 0 && w.EndNano > 0 && w.StartNano >= w.EndNano {
		return fmt.Errorf("window start must be before end, got start=%d end=%d", w.StartNano, w.EndNano)
	}

	return nil
}

// fetchBounds resolves the window to the bounds handed to a block fetch, reporting whether
// a bound applies at all.
//
// Both fetch fields must be set together or the bound does not apply: vparquet{3,4,5}
// install the trace-time predicate only under `if start > 0 && end > 0`, so assigning one
// field and leaving the other at zero REMOVES the filter rather than narrowing it — the
// block is scanned in full and every query match is dropped regardless of its timestamp.
//
// SubmitRedaction rejects one-sided windows, but that check runs at submission: a batch
// persisted by an older scheduler, or any future caller, can still deliver one here.
// Materialising the open side keeps the scan bounded whatever the source.
func (w RedactionWindow) fetchBounds() (start, end uint64, ok bool) {
	if w.StartNano <= 0 && w.EndNano <= 0 {
		return 0, 0, false
	}

	lo, hi := w.StartNano, w.EndNano
	if lo <= 0 {
		lo = 1 // 0 would disable the predicate; 1ns is the earliest bound that keeps it installed
	}
	if hi <= 0 {
		hi = math.MaxInt64
	}

	// Validate rejects an inverted window and RedactBlock calls it first, so this is unreachable today.
	// It stays because the failure it prevents is silent: an inverted predicate matches nothing, so a
	// second caller added later would under-delete and report success with no signal.
	if lo >= hi {
		return 0, 0, false
	}

	return uint64(lo), uint64(hi), true
}
