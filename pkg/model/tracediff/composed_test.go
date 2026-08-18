package tracediff

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeUnderBudgetAttachesPatch(t *testing.T) {
	base, compare := summaryFixtureTraces()

	got, err := Compose(base, compare, DefaultPatchBudgetBytes, nil)
	require.NoError(t, err)

	assert.Equal(t, VersionTraceSummaryV0Composed, got.Version)
	require.NotNil(t, got.Summary)
	assert.Equal(t, VersionTraceSummaryV0Native, got.Summary.Version)
	assert.Nil(t, got.PatchOmitted)
	wantSummary, err := Summarize(base, compare)
	require.NoError(t, err)
	assert.Equal(t, wantSummary, got.Summary)

	var patch Result
	require.NoError(t, json.Unmarshal(got.Patch, &patch))
	assert.Equal(t, VersionTracePatchV0, patch.Version)

	want, err := Diff(base, compare, FormatTracePatchV0)
	require.NoError(t, err)
	wantBytes, err := json.Marshal(want)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(wantBytes), got.Patch)
}

func TestComposeBudgetBoundary(t *testing.T) {
	base, compare := summaryFixtureTraces()

	// Compose itself pins the serialized patch size the boundary is tested
	// against.
	got, err := Compose(base, compare, DefaultPatchBudgetBytes, nil)
	require.NoError(t, err)
	require.NotEmpty(t, got.Patch)
	patchSize := len(got.Patch)

	atBudget, err := Compose(base, compare, patchSize, nil)
	require.NoError(t, err)
	assert.Equal(t, got.Patch, atBudget.Patch)
	assert.Nil(t, atBudget.PatchOmitted)

	overBudget, err := Compose(base, compare, patchSize-1, nil)
	require.NoError(t, err)
	assert.Equal(t, VersionTraceSummaryV0Composed, overBudget.Version)
	require.NotNil(t, overBudget.Summary)
	assert.Equal(t, VersionTraceSummaryV0Native, overBudget.Summary.Version)
	assert.Nil(t, overBudget.Patch)
	require.NotNil(t, overBudget.PatchOmitted)
	assert.Equal(t, PatchOmitted{Bytes: patchSize, Reason: "over_budget"}, *overBudget.PatchOmitted)

	warning := Warning{Code: WarningPartialTrace, Message: "input may be incomplete"}
	withWarning, err := Compose(base, compare, patchSize, []Warning{warning})
	require.NoError(t, err)
	assert.Nil(t, withWarning.Patch)
	require.NotNil(t, withWarning.PatchOmitted)
	assert.Greater(t, withWarning.PatchOmitted.Bytes, patchSize)
	assert.Contains(t, withWarning.Summary.Warnings, warning)
}

func TestComposeJSONShape(t *testing.T) {
	tests := []struct {
		name     string
		budget   int
		wantKeys []string
	}{
		{
			name:     "patch attached",
			budget:   DefaultPatchBudgetBytes,
			wantKeys: []string{"version", "summary", "patch"},
		},
		{
			name:     "patch omitted",
			budget:   1,
			wantKeys: []string{"version", "summary", "patchOmitted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, compare := summaryFixtureTraces()
			got, err := Compose(base, compare, tt.budget, nil)
			require.NoError(t, err)
			data, err := json.Marshal(got)
			require.NoError(t, err)

			var doc map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &doc))
			keys := make([]string, 0, len(doc))
			for key := range doc {
				keys = append(keys, key)
			}
			assert.ElementsMatch(t, tt.wantKeys, keys)
			assert.Equal(t, json.RawMessage(`"trace-summary-v0-composed"`), doc["version"])
			if omitted, ok := doc["patchOmitted"]; ok {
				var disclosure map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(omitted, &disclosure))
				assert.Contains(t, disclosure, "bytes")
				assert.Contains(t, disclosure, "reason")
			}
		})
	}
}

func TestComposeRejectsNilInputs(t *testing.T) {
	tests := []struct {
		name    string
		base    *tempopb.Trace
		compare *tempopb.Trace
	}{
		{name: "nil base", base: nil, compare: &tempopb.Trace{}},
		{name: "nil compare", base: &tempopb.Trace{}, compare: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compose(tt.base, tt.compare, DefaultPatchBudgetBytes, nil)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrNilTrace))
			assert.Nil(t, got)
		})
	}
}
