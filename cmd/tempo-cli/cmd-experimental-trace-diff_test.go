package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/tempo/pkg/model/tracediff"
	"github.com/stretchr/testify/require"
)

func TestExperimentalTraceDiffWritesTracePatch(t *testing.T) {
	dir := t.TempDir()
	traceA := filepath.Join(dir, "trace-a.json")
	traceB := filepath.Join(dir, "trace-b.json")
	out := filepath.Join(dir, "diff.json")
	require.NoError(t, os.WriteFile(traceA, []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(traceB, []byte(`{}`), 0o600))

	cmd := experimentalTraceDiffCmd{
		TraceA: traceA,
		TraceB: traceB,
		Format: string(tracediff.FormatTracePatchV0),
		Out:    out,
	}
	require.NoError(t, cmd.Run(nil))

	bytes, err := os.ReadFile(out)
	require.NoError(t, err)

	var result tracediff.Result
	require.NoError(t, json.Unmarshal(bytes, &result))
	require.Equal(t, tracediff.VersionTracePatchV0, result.Version)
	require.Empty(t, result.Modified)
	require.Empty(t, result.Added)
	require.Empty(t, result.Removed)
}
