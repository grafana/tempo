package backendscheduler

import (
	"context"
	"errors"

	"github.com/grafana/dskit/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/modules/backendscheduler/work"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/validation"
)

// GetRedactionStatus reports the state of the calling tenant's redaction batch. Read-only: it
// creates nothing, removes nothing, and does not touch the batch lifecycle.
//
// No batch is not an error. A redaction that finishes leaves no batch behind, so an operator
// polling a completed run gets active=false; the durable record of what was removed is the
// redaction_traces_found_total metric, not this call.
//
// As with SubmitRedaction, the tenant comes only from the X-Scope-OrgID header -- the request
// carries no tenant field, so one tenant cannot read another's redaction state.
func (s *BackendScheduler) GetRedactionStatus(ctx context.Context, _ *tempopb.GetRedactionStatusRequest) (*tempopb.GetRedactionStatusResponse, error) {
	_, span := tracer.Start(ctx, "GetRedactionStatus")
	defer span.End()

	tenant, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		if errors.Is(err, user.ErrNoOrgID) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	snap, ok := s.work.RedactionStatus(tenant)
	if !ok {
		return &tempopb.GetRedactionStatusResponse{Active: false}, nil
	}

	counts := s.work.RedactionJobCounts(tenant, snap.BatchID)

	return &tempopb.GetRedactionStatusResponse{
		Active:            true,
		BatchId:           snap.BatchID,
		Mode:              snap.Mode,
		Phase:             redactionPhase(snap, counts),
		CreatedAtUnixNano: snap.CreatedAtUnixNano,
		StartTimeUnixNano: snap.StartTimeUnixNano,
		EndTimeUnixNano:   snap.EndTimeUnixNano,
		JobsCreated:       snap.JobsCreated,
		JobsRemaining:     int32(counts.Remaining),
		JobsRunning:       int32(counts.Running),
		JobsFailed:        int32(counts.Failed),
	}, nil
}

// redactionPhase classifies why the batch still exists, using the same two conditions as
// redactionBatchActive (outstanding jobs, or a pending rescan) so the reported phase cannot
// disagree with the scheduler's own notion of done.
//
// Outstanding jobs are judged from the reported remaining count rather than HasJobsForTenant, so
// phase and jobs_remaining are always drawn from one read and can never contradict each other in
// the response. A rescan is pending from submission onward whenever blocks were skipped, so it is
// only meaningful once the jobs have drained -- hence the ordering here.
func redactionPhase(snap work.RedactionStatus, counts work.RedactionJobCounts) tempopb.RedactionPhase {
	switch {
	case counts.Remaining > 0:
		return tempopb.RedactionPhase_REDACTION_PHASE_RUNNING
	case snap.RescanPending:
		return tempopb.RedactionPhase_REDACTION_PHASE_AWAITING_RESCAN
	default:
		// Jobs drained and no rescan outstanding: the batch is held only to keep compaction off
		// until removal. QuiesceUntilUnixNano may still be 0 if the maintenance tick that records
		// the deadline has not run yet, which is the same waiting-for-teardown state.
		return tempopb.RedactionPhase_REDACTION_PHASE_QUIESCING
	}
}
