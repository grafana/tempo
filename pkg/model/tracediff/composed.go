package tracediff

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	// VersionTraceSummaryV0Composed is an optional diff response shape: the
	// native summary is always present, plus the full trace-patch-v0
	// attached iff it fits the byte budget, omitted whole with a disclosure
	// otherwise. Never truncated: a truncated change list lies by omission.
	// Consumers needing the full patch when it exceeds the budget re-request
	// with format trace-patch-v0; the two-request cost is accepted for the
	// experimental endpoint rather than adding a bypass knob.
	VersionTraceSummaryV0Composed = "trace-summary-v0-composed"

	// DefaultPatchBudgetBytes bounds the patch attached to a composed
	// response. The benchmark validated the attach/omit mechanism (attached
	// at <=21.5KB fixtures, omitted at 2.3MB); 64KB is a policy pick inside
	// that validated band, not a measured optimum. Tooling that wants the
	// patch at any size asks for trace-patch-v0 directly.
	DefaultPatchBudgetBytes = 64 * 1024
)

// ComposedResult is the composed diff response: summary for triage and
// localization, patch for span-level evidence when it fits the budget.
type ComposedResult struct {
	Version      string          `json:"version"`
	Summary      *SummaryResult  `json:"summary"`
	Patch        json.RawMessage `json:"patch,omitempty"`
	PatchOmitted *PatchOmitted   `json:"patchOmitted,omitempty"`
}

// PatchOmitted discloses an omitted patch and its size, so consumers can
// decide whether to fetch it (scoped or whole) instead of silently not
// knowing it existed.
type PatchOmitted struct {
	Bytes  int    `json:"bytes"`
	Reason string `json:"reason"`
}

// Compose diffs the two traces once and returns the composed response. Extra
// warnings are added before building the summary and sizing the patch, keeping
// both views and the budget decision consistent.
func Compose(base, compare *tempopb.Trace, budgetBytes int, additionalWarnings []Warning) (*ComposedResult, error) {
	patch, baseTrace, compareTrace, err := diffTraceInputs(base, compare)
	if err != nil {
		return nil, err
	}
	patch.Warnings = append(patch.Warnings, additionalWarnings...)
	summary := summarizeFromPatch(patch, baseTrace, compareTrace)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal patch for composed response: %w", err)
	}

	result := &ComposedResult{Version: VersionTraceSummaryV0Composed, Summary: summary}
	if len(patchBytes) <= budgetBytes {
		result.Patch = patchBytes
	} else {
		result.PatchOmitted = &PatchOmitted{Bytes: len(patchBytes), Reason: "over_budget"}
	}
	return result, nil
}
