package backendscheduler

import (
	"time"

	"github.com/grafana/tempo/tempodb/backend"
)

// blockTimeGranularity is the resolution of BlockMeta.StartTime/EndTime. ObjectAdded builds them from
// uint32 epoch SECONDS, so the recorded range is truncated and a block's real data can extend up to one
// second past EndTime. The overlap test pads by this much so a window opening inside that final second
// still selects the block. The same reasoning is recorded in traceql.TrimToBlockOverlap.
const blockTimeGranularity = time.Second

// blockOverlapsWindow reports whether a block's data range overlaps the redaction window
// [startNano, endNano], and whether that answer had to be assumed because the block's recorded range is
// unusable. A zero bound is unbounded on that side, so 0/0 (the default) matches every block — the whole
// tenant, i.e. prior behaviour.
//
// Selection keys strictly on the block's data range (StartTime/EndTime), never CompactedTime. The poller
// fudges CompactedTime to "now" at compaction discovery to avoid a per-block backend read, so keying on
// it would misjudge the block's real content window. (CompactedTime also lives on CompactedBlockMeta,
// not BlockMeta, and the metas reaching here come from the live blocklist — so this is a note about why
// the compacted metas are not consulted, not a field choice available at this call site.)
//
// Doubt resolves toward inclusion. The per-block scan bound decides which traces are actually deleted, so
// selecting an extra block costs I/O; excluding one silently leaves data the operator asked to delete,
// which they cannot detect. Two cases need it:
//
//   - An unusable recorded range. ObjectAdded skips zero timestamps, so a block completed from a
//     replayed WAL reaches the backend with no times at all, and time.Time{}.UnixNano() is a large
//     negative number rather than zero — comparing it directly excludes the block from every window with
//     a lower bound. Such blocks are reported as indeterminate so the caller can count them.
//   - Second-granularity truncation, handled by the padding above.
func blockOverlapsWindow(meta *backend.BlockMeta, startNano, endNano int64) (overlaps, indeterminate bool) {
	if startNano == 0 && endNano == 0 {
		return true, false // no window: every block is in scope, and the range is never consulted
	}

	if meta.StartTime.IsZero() || meta.EndTime.IsZero() {
		return true, true
	}

	// Pad outward: the recorded range is a truncated, therefore understated, view of the real data.
	blockStart := meta.StartTime.Add(-blockTimeGranularity)
	blockEnd := meta.EndTime.Add(blockTimeGranularity)

	if endNano != 0 && blockStart.After(time.Unix(0, endNano)) {
		return false, false // block begins after the window ends
	}
	if startNano != 0 && blockEnd.Before(time.Unix(0, startNano)) {
		return false, false // block ends before the window begins
	}
	return true, false
}

// coveredRange reports the span of data the selected blocks actually hold, and whether any block
// contributed a usable range.
//
// Blocks whose recorded range is unusable contribute nothing. Selection deliberately includes those
// (see blockOverlapsWindow), so they reach here by construction — and time.Time{} both precedes every
// real timestamp and is indistinguishable from an unseeded accumulator, so admitting one would drag the
// reported start back to year 1. That understates the blast radius on exactly the blocks selection was
// least confident about, in the one record an operator has for an operation with no undo.
//
// ok is false when no block had a usable range; callers report that as unknown rather than as year 1.
func coveredRange(metas []*backend.BlockMeta) (start, end time.Time, ok bool) {
	for _, meta := range metas {
		if meta.StartTime.IsZero() || meta.EndTime.IsZero() {
			continue
		}

		if !ok {
			start, end, ok = meta.StartTime, meta.EndTime, true
			continue
		}

		if meta.StartTime.Before(start) {
			start = meta.StartTime
		}
		if meta.EndTime.After(end) {
			end = meta.EndTime
		}
	}

	return start, end, ok
}

// coveredRangeLabel renders a covered bound for an audit record, so "no block reported a usable range"
// is distinguishable from a real timestamp instead of both printing as year 1.
func coveredRangeLabel(t time.Time, ok bool) string {
	if !ok {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}
