package backendscheduler

import (
	"time"

	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
)

// blockTimeGranularity is the resolution of BlockMeta.StartTime/EndTime: ObjectAdded builds them from
// uint32 epoch SECONDS, so real data can extend up to a second past the recorded EndTime. Selection pads
// by this much so a window opening inside that final second still picks the block up, matching
// traceql.TrimToBlockOverlap.
const blockTimeGranularity = time.Second

// blockOverlapsWindow reports whether a block's data range overlaps [startNano, endNano], and whether
// that answer had to be assumed. A zero bound is unbounded on that side, so 0/0 matches every block.
//
// Doubt resolves toward inclusion: for a query redaction the per-block scan bound decides what is
// actually deleted, so an extra block costs I/O while a missing one silently leaves data the operator
// asked to delete. A block whose recorded range is unusable is therefore included and reported as
// indeterminate so the caller can count it.
func blockOverlapsWindow(meta *backend.BlockMeta, startNano, endNano int64) (overlaps, indeterminate bool) {
	if startNano == 0 && endNano == 0 {
		return true, false // no window: every block is in scope, and the range is never consulted
	}

	// Unusable recorded range: no times at all (what a replayed WAL produces), or an inverted one, which
	// describes no interval. Both are included and reported rather than compared, because the padded
	// comparisons below would exclude an inverted range outright.
	if meta.StartTime.IsZero() || meta.EndTime.IsZero() || meta.StartTime.After(meta.EndTime) {
		return true, true
	}

	// Pad outward: the recorded range is truncated to whole seconds, so it understates the real end.
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

// coveredRange reports the span of data the selected blocks actually hold — the honest blast radius for
// an operation with no undo — with a flag per bound saying whether any block supplied it.
//
// Unusable timestamps contribute nothing. Selection deliberately includes blocks with no recorded range,
// and time.Time{} both precedes every real timestamp and looks exactly like an unseeded accumulator, so
// admitting one would drag the reported start back to year 1. The two bounds are tracked separately so a
// half-usable meta still contributes its good half; a bound with no contributor renders as unknown.
func coveredRange(metas []*backend.BlockMeta) (start, end time.Time, startOK, endOK bool) {
	for _, meta := range metas {
		// A meta whose recorded range is inverted describes no interval; neither bound is trustworthy.
		if !meta.StartTime.IsZero() && !meta.EndTime.IsZero() && meta.StartTime.After(meta.EndTime) {
			continue
		}

		// The two bounds accumulate independently: a meta can carry a usable start and a zero end, and
		// discarding its start along with the unusable end would understate the range on a block that
		// was nonetheless enqueued.
		if !meta.StartTime.IsZero() && (!startOK || meta.StartTime.Before(start)) {
			start, startOK = meta.StartTime, true
		}
		if !meta.EndTime.IsZero() && (!endOK || meta.EndTime.After(end)) {
			end, endOK = meta.EndTime, true
		}
	}

	return start, end, startOK, endOK
}

// dropBatchesWithUnusableWindows removes any loaded batch whose persisted window cannot be honoured. This
// is the trust boundary for windows arriving off disk, so Next() and performRescan can both assume a
// usable window.
//
// Refusing is the only safe option: skipping the batch's blocks under-deletes silently, and treating the
// window as unbounded would delete every query match in them regardless of time. The operator must
// resubmit, which the error log says.
func (s *BackendScheduler) dropBatchesWithUnusableWindows() {
	for _, batch := range s.work.ListBatches() {
		w := tempodb.RedactionWindow{StartNano: batch.StartTimeUnixNano, EndNano: batch.EndTimeUnixNano}
		err := w.Validate()
		if err == nil {
			continue
		}

		level.Error(log.Logger).Log(
			"msg", "discarding persisted redaction batch with an unusable time window; nothing was redacted for it, resubmit if the traces are still present",
			"tenant", batch.TenantId, "batch_id", batch.BatchId,
			"start_time_unix_nano", batch.StartTimeUnixNano, "end_time_unix_nano", batch.EndTimeUnixNano,
			"err", err,
		)
		s.work.RemoveBatch(batch.TenantId)
	}
}

// coveredRangeLabel renders a covered bound for an audit record, so "no block reported a usable range"
// is distinguishable from a real timestamp instead of both printing as year 1.
func coveredRangeLabel(t time.Time, ok bool) string {
	if !ok {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}

// validateRedactionRequest rejects a submission the scheduler cannot honour, returning a gRPC status
// error. Every check fails closed: on a redaction, a refused request destroys nothing while a
// misinterpreted one cannot be undone.
func validateRedactionRequest(req *tempopb.SubmitRedactionRequest, querySel *tempopb.TraceQLSelector) error {
	// Exactly one selector. The proto reserves a single-member oneof for query; the XOR is enforced
	// here until trace_ids migrates into it.
	hasIDs := len(req.TraceIds) > 0
	hasQuery := querySel.GetQuery() != "" // nil-safe
	switch {
	case hasIDs && hasQuery:
		return status.Error(codes.InvalidArgument, "trace_ids and query are mutually exclusive")
	case !hasIDs && !hasQuery:
		return status.Error(codes.InvalidArgument, "one of trace_ids or query must be set")
	case hasQuery:
		if err := validateRedactionQuery(querySel.Query); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Only DRY_RUN is checked downstream, so an unrecognised mode would fall through to a destructive
	// rewrite. Reject it rather than defaulting.
	switch req.Mode {
	case tempopb.RedactionMode_REDACTION_MODE_APPLY, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN:
	default:
		return status.Errorf(codes.InvalidArgument, "unknown redaction mode %d", int32(req.Mode))
	}

	// The optional [start, end] window, already resolved to absolute nanos by the client.
	//
	// Both bounds are required by policy, not by a storage limit: fetchBounds does materialise a
	// one-sided window, but a half-specified window is almost always a typo and guessing has no undo.
	if (req.StartTimeUnixNano == 0) != (req.EndTimeUnixNano == 0) {
		return status.Error(codes.InvalidArgument, "start_time_unix_nano and end_time_unix_nano must both be set or both be omitted")
	}
	// A negative bound cannot describe real data (block times are post-epoch) and reads differently at
	// different layers, so it either widens or empties the selection.
	if req.StartTimeUnixNano < 0 || req.EndTimeUnixNano < 0 {
		return status.Errorf(codes.InvalidArgument, "window bounds must be non-negative unix nanoseconds, got start=%d end=%d", req.StartTimeUnixNano, req.EndTimeUnixNano)
	}
	// start == end matches only traces spanning that exact instant, i.e. nothing, while reporting success.
	if req.StartTimeUnixNano != 0 && req.StartTimeUnixNano >= req.EndTimeUnixNano {
		return status.Errorf(codes.InvalidArgument, "start_time_unix_nano must be before end_time_unix_nano, got start=%d end=%d", req.StartTimeUnixNano, req.EndTimeUnixNano)
	}

	// A window scopes which blocks are read, and the trace-ID path applies no time bound at all, so the
	// pair would delete each listed trace from the overlapping blocks and leave the rest of it behind
	// while reporting success. See tempodb.RedactionWindow.
	if hasIDs && req.StartTimeUnixNano != 0 {
		return status.Error(codes.InvalidArgument,
			"a time window cannot be combined with trace_ids: the window is not applied per trace, so only the parts of each trace held by in-window blocks would be removed")
	}

	return nil
}
