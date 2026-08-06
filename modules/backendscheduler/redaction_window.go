package backendscheduler

import "github.com/grafana/tempo/tempodb/backend"

// blockOverlapsWindow reports whether a block's data time range [meta.StartTime, meta.EndTime]
// overlaps the redaction window [startNano, endNano]. A zero bound is unbounded on that side, so
// 0/0 (the default) matches every block — the whole tenant, i.e. prior behavior.
//
// Selection keys strictly on the block's data range (StartTime/EndTime), never CompactedTime: the
// poller fudges CompactedTime to "now" at compaction discovery to avoid a per-block backend read,
// so using it here would misjudge a block's real content window and could UNDER-select — silently
// skipping redaction of in-window traces (a completeness failure). StartTime/EndTime are the real
// data range and are safe.
func blockOverlapsWindow(meta *backend.BlockMeta, startNano, endNano int64) bool {
	if endNano != 0 && meta.StartTime.UnixNano() > endNano {
		return false // block begins after the window ends
	}
	if startNano != 0 && meta.EndTime.UnixNano() < startNano {
		return false // block ends before the window begins
	}
	return true
}
