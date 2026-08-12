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
// A one-sided window is accepted here — materialise pins the open side — even though SubmitRedaction
// rejects one at the API edge.
func (w RedactionWindow) Validate() error {
	if w.StartNano < 0 || w.EndNano < 0 {
		return fmt.Errorf("window bounds must be non-negative unix nanoseconds, got start=%d end=%d", w.StartNano, w.EndNano)
	}

	// Judge the MATERIALISED range, not the raw fields. A one-sided window is pinned before the scan, so
	// an end bound of 1ns resolves to [1,1] and a start bound at the maximum instant to [max,max]. Both
	// have ordered raw fields and neither can install a predicate, which would leave the block scanned in
	// full and every query match dropped whatever its timestamp.
	if lo, hi, bounded := w.materialise(); bounded && lo >= hi {
		return fmt.Errorf("window start must be before end: start=%d end=%d resolves to the scan range [%d, %d]",
			w.StartNano, w.EndNano, lo, hi)
	}

	return nil
}

// materialise resolves the window to the inclusive bounds a block scan needs, reporting whether any
// bound applies at all.
//
// The open side is pinned rather than left at zero because vparquet{3,4,5} install the trace-time
// predicate only under `start > 0 && end > 0`: a half-set window would remove the filter instead of
// narrowing it. Validate judges this same resolved range, so the two cannot disagree about which windows
// are usable.
func (w RedactionWindow) materialise() (lo, hi int64, bounded bool) {
	if w.StartNano <= 0 && w.EndNano <= 0 {
		return 0, 0, false
	}

	lo, hi = w.StartNano, w.EndNano
	if lo <= 0 {
		lo = 1 // 0 would disable the predicate; 1ns is the earliest bound that keeps it installed
	}
	if hi <= 0 {
		hi = math.MaxInt64
	}

	return lo, hi, true
}

// fetchBounds resolves the window to block-fetch bounds, reporting whether a bound applies at all.
//
// Requires a window that Validate has accepted. That guarantee is what makes the result safe to trust:
// for any non-zero validated window this returns ok=true with lo < hi, so a caller can never silently
// fall back to scanning the block in full. TestRedactionWindowValidateImpliesBoundedScan pins the link.
func (w RedactionWindow) fetchBounds() (start, end uint64, ok bool) {
	lo, hi, bounded := w.materialise()
	if !bounded {
		return 0, 0, false
	}

	return uint64(lo), uint64(hi), true
}
