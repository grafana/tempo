package backendscheduler

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

// TestRecordRedactionResult verifies the per-tenant traces-found counter is incremented by the
// reported count and labelled by mode, so apply (real redactions) and dry-run (previewed blast
// radius) are distinguishable, and that a zero/empty result is a no-op.
func TestRecordRedactionResult(t *testing.T) {
	recordRedactionResult("t-apply", tempopb.RedactionMode_REDACTION_MODE_APPLY, 5)
	require.Equal(t, 5.0, testutil.ToFloat64(metricRedactionTracesFound.WithLabelValues("t-apply", "apply")))

	recordRedactionResult("t-dry", tempopb.RedactionMode_REDACTION_MODE_DRY_RUN, 7)
	require.Equal(t, 7.0, testutil.ToFloat64(metricRedactionTracesFound.WithLabelValues("t-dry", "dry_run")))

	// Zero found: no-op (block scanned clean).
	recordRedactionResult("t-zero", tempopb.RedactionMode_REDACTION_MODE_APPLY, 0)
	require.Equal(t, 0.0, testutil.ToFloat64(metricRedactionTracesFound.WithLabelValues("t-zero", "apply")))
}
