package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryTraceIDCmdRejectsQWithV1(t *testing.T) {
	// --q is v2-only, so combining it with --v1 must fail fast before any request is made.
	cmd := &queryTraceIDCmd{
		APIEndpoint:   "http://localhost:0",
		TraceID:       "1234",
		V1:            true,
		Q:             `{ .foo = "bar" }`,
		KeepHierarchy: true,
	}

	err := cmd.Run(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only supported on the v2 API")
}

// TestQueryTraceIDCmdShapingFlagsNeverFailFast covers the shaping flags that only refine a filtered
// result. None of them may fail client-side validation, whether or not the combination is meaningful:
// a flag with nothing to act on is ignored, so someone iterating on a trace never has to clear stale
// flags between runs. Each case must get past validation and die on the unreachable endpoint instead.
func TestQueryTraceIDCmdShapingFlagsNeverFailFast(t *testing.T) {
	tests := []struct {
		name string
		cmd  queryTraceIDCmd
	}{
		{name: "keep_hierarchy without q", cmd: queryTraceIDCmd{KeepHierarchy: true}},
		{name: "ancestor_depth without q", cmd: queryTraceIDCmd{AncestorDepth: 1}},
		{name: "match_depth without q", cmd: queryTraceIDCmd{MatchDepth: 2}},
		{name: "all shaping flags without q", cmd: queryTraceIDCmd{KeepHierarchy: true, MatchDepth: 2, AncestorDepth: 1}},
		// ancestor_depth has nothing to bound while keep_hierarchy is false, but that is inert, not invalid.
		{name: "ancestor_depth with q but keep_hierarchy false", cmd: queryTraceIDCmd{Q: `{ .foo = "bar" }`, AncestorDepth: 1}},
		{name: "all shaping flags with q", cmd: queryTraceIDCmd{Q: `{ .foo = "bar" }`, KeepHierarchy: true, MatchDepth: 2, AncestorDepth: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd
			cmd.APIEndpoint = "http://localhost:0"
			cmd.TraceID = "1234"

			// port 0 is unreachable, so reaching the request at all surfaces as a connection error.
			err := cmd.Run(nil)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "only supported on the v2 API")
			require.NotContains(t, err.Error(), "only applies with")
		})
	}
}
