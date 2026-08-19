package main

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/dskit/user"

	schedulerclient "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/pkg/tempopb"
)

type redactStatusCmd struct {
	SchedulerAddr string `arg:"" help:"backend scheduler gRPC address (host:port)"`

	TenantID string `name:"tenant" required:"" help:"tenant ID"`

	TLS           bool   `name:"tls" help:"use TLS transport" default:"false"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name (SNI)"`
	TLSCA         string `name:"tls-ca" help:"path to a PEM-encoded CA certificate file"`
}

func (cmd *redactStatusCmd) Run(_ *globalOptions) error {
	transportCred, err := schedulerTransportCredentials(cmd.TLS, cmd.TLSServerName, cmd.TLSCA)
	if err != nil {
		return fmt.Errorf("building transport credentials: %w", err)
	}

	c, err := schedulerclient.NewWithOptions(cmd.SchedulerAddr, defaultSchedulerClientConfig(), transportCred)
	if err != nil {
		return fmt.Errorf("creating scheduler client: %w", err)
	}
	defer c.Close()

	ctx := user.InjectOrgID(context.Background(), cmd.TenantID)
	ctx, err = user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return fmt.Errorf("injecting tenant ID into gRPC request: %w", err)
	}

	resp, err := c.GetRedactionStatus(ctx, &tempopb.GetRedactionStatusRequest{})
	if err != nil {
		return fmt.Errorf("getting redaction status: %w", err)
	}

	fmt.Print(formatRedactionStatus(resp))
	return nil
}

// formatRedactionStatus renders the status for a terminal. Split from Run so the wording is
// testable without a server.
func formatRedactionStatus(resp *tempopb.GetRedactionStatusResponse) string {
	// No batch is the normal state after a redaction finishes, so say that rather than implying
	// something is missing. The metric, not this command, is the record of what was removed.
	if !resp.GetActive() {
		return "no redaction in progress for this tenant\n"
	}

	out := fmt.Sprintf("batch_id:   %s\nmode:       %s\nphase:      %s\nsubmitted:  %s\n",
		resp.GetBatchId(),
		redactionModeLabel(resp.GetMode()),
		redactionPhaseLabel(resp.GetPhase()),
		time.Unix(0, resp.GetCreatedAtUnixNano()).UTC().Format(time.RFC3339),
	)

	if resp.GetStartTimeUnixNano() != 0 || resp.GetEndTimeUnixNano() != 0 {
		out += fmt.Sprintf("window:     %s .. %s\n",
			windowBoundLabel(resp.GetStartTimeUnixNano()),
			windowBoundLabel(resp.GetEndTimeUnixNano()))
	} else {
		out += "window:     whole tenant\n"
	}

	// jobs_created is 0 on a batch written by a binary predating the field. Submission refuses an
	// empty selection, so 0 here means unknown rather than "no work" -- report remaining alone
	// instead of a progress fraction that would read as 0 of 0.
	if created := resp.GetJobsCreated(); created > 0 {
		out += fmt.Sprintf("blocks:     %d of %d done, %d remaining (%d running)\n",
			created-resp.GetJobsRemaining(), created, resp.GetJobsRemaining(), resp.GetJobsRunning())
	} else {
		out += fmt.Sprintf("blocks:     %d remaining (%d running); total unknown\n",
			resp.GetJobsRemaining(), resp.GetJobsRunning())
	}

	// A failure means blocks the redaction was supposed to cover were not rewritten, so it needs to
	// be visible rather than folded into the counts above.
	if failed := resp.GetJobsFailed(); failed > 0 {
		out += fmt.Sprintf("failed:     %d block job(s) failed; those blocks were not redacted\n", failed)
	}

	return out
}

func redactionModeLabel(mode tempopb.RedactionMode) string {
	if mode == tempopb.RedactionMode_REDACTION_MODE_DRY_RUN {
		return "dry-run (no blocks are rewritten)"
	}
	return "apply"
}

func redactionPhaseLabel(phase tempopb.RedactionPhase) string {
	switch phase {
	case tempopb.RedactionPhase_REDACTION_PHASE_RUNNING:
		return "running"
	case tempopb.RedactionPhase_REDACTION_PHASE_AWAITING_RESCAN:
		return "awaiting rescan (covering blocks compaction produced from skipped inputs)"
	case tempopb.RedactionPhase_REDACTION_PHASE_QUIESCING:
		return "quiescing (work done; compaction stays paused until teardown)"
	default:
		return "unknown"
	}
}

func windowBoundLabel(nano int64) string {
	if nano == 0 {
		return "unbounded"
	}
	return time.Unix(0, nano).UTC().Format(time.RFC3339)
}
