package tempodb

import (
	"fmt"
	"math"
)

// RedactionWindow bounds the per-block scan of a TraceQL redaction. Both bounds are unix nanoseconds;
// the zero value is unbounded, matching every query match in the block regardless of timestamp.
//
// Two limits are easy to misread and both destroy data:
//
//   - Overlap, not containment. A trace whose range merely intersects the window matches, and a matched
//     trace is dropped whole, so its out-of-window spans go too.
//   - Query selector only. The trace-ID path resolves IDs with no time bound; RedactBlock refuses that
//     combination rather than accepting a window it would ignore.
//
// The bounds travel as named fields because transposing two adjacent int64 parameters fails silently:
// an inverted window matches nothing and the job reports success.
type RedactionWindow struct {
	StartNano int64
	EndNano   int64
}

// IsZero reports whether the window is unbounded.
func (w RedactionWindow) IsZero() bool {
	return w.StartNano == 0 && w.EndNano == 0
}

// Validate reports whether the window can be honoured, so a caller refuses it rather than scanning with
// it and reporting the empty result as a completed redaction.
//
// A one-sided window is accepted here — fetchBounds materialises the open side — even though
// SubmitRedaction rejects one at the API edge.
func (w RedactionWindow) Validate() error {
	if w.StartNano < 0 || w.EndNano < 0 {
		return fmt.Errorf("window bounds must be non-negative unix nanoseconds, got start=%d end=%d", w.StartNano, w.EndNano)
	}

	// Ordering is only meaningful once both bounds are set.
	if w.StartNano > 0 && w.EndNano > 0 && w.StartNano >= w.EndNano {
		return fmt.Errorf("window start must be before end, got start=%d end=%d", w.StartNano, w.EndNano)
	}

	return nil
}

// fetchBounds resolves the window to block-fetch bounds, reporting whether a bound applies at all.
//
// The open side is materialised rather than left at zero because vparquet{3,4,5} install the trace-time
// predicate only under `start > 0 && end > 0` — a half-set window would remove the filter instead of
// narrowing it, scanning the block in full.
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

	// Unreachable via RedactBlock, which validates first. Kept because an inverted predicate matches
	// nothing, so a second caller added later would under-delete and report success.
	if lo >= hi {
		return 0, 0, false
	}

	return uint64(lo), uint64(hi), true
}
