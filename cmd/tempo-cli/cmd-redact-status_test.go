package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

func TestFormatRedactionStatusInactive(t *testing.T) {
	out := formatRedactionStatus(&tempopb.GetRedactionStatusResponse{Active: false})
	require.Equal(t, "no redaction in progress for this tenant\n", out,
		"absence of a batch is the normal post-completion state, not an error to report")
}

func TestFormatRedactionStatusProgress(t *testing.T) {
	out := formatRedactionStatus(&tempopb.GetRedactionStatusResponse{
		Active:            true,
		BatchId:           "b1",
		Mode:              tempopb.RedactionMode_REDACTION_MODE_APPLY,
		Phase:             tempopb.RedactionPhase_REDACTION_PHASE_RUNNING,
		CreatedAtUnixNano: 1,
		JobsCreated:       10,
		JobsRemaining:     4,
		JobsRunning:       2,
	})
	require.Contains(t, out, "blocks:     6 of 10 done, 4 remaining (2 running)")
	require.Contains(t, out, "phase:      running")
	require.Contains(t, out, "window:     whole tenant")
	require.NotContains(t, out, "failed:", "no failures means no failure line")
}

// TestFormatRedactionStatusUnknownTotal covers a batch persisted before jobs_created existed.
// Reporting "0 of 0 done" there would tell the operator the redaction is complete when it is not.
func TestFormatRedactionStatusUnknownTotal(t *testing.T) {
	out := formatRedactionStatus(&tempopb.GetRedactionStatusResponse{
		Active:        true,
		JobsCreated:   0,
		JobsRemaining: 7,
		JobsRunning:   1,
	})
	require.Contains(t, out, "7 remaining (1 running); total unknown")
	require.NotContains(t, out, "of 0 done")
}

func TestFormatRedactionStatusSurfacesFailures(t *testing.T) {
	out := formatRedactionStatus(&tempopb.GetRedactionStatusResponse{
		Active:      true,
		JobsCreated: 5,
		JobsFailed:  2,
	})
	require.Contains(t, out, "failed:     2 block job(s) failed; those blocks were not redacted")
}

func TestFormatRedactionStatusShowsWindowAndDryRun(t *testing.T) {
	out := formatRedactionStatus(&tempopb.GetRedactionStatusResponse{
		Active:            true,
		Mode:              tempopb.RedactionMode_REDACTION_MODE_DRY_RUN,
		StartTimeUnixNano: 1600000000000000000,
		EndTimeUnixNano:   0,
		JobsCreated:       1,
	})
	require.Contains(t, out, "dry-run")
	require.Contains(t, out, "2020-09-13T12:26:40Z .. unbounded",
		"a one-sided window must read as unbounded, not as the epoch")
}
