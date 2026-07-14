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

func TestQueryTraceIDCmdRejectsKeepHierarchyWithoutQ(t *testing.T) {
	// --keep-hierarchy only shapes a filtered result, so it must fail fast without --q.
	cmd := &queryTraceIDCmd{
		APIEndpoint:   "http://localhost:0",
		TraceID:       "1234",
		KeepHierarchy: true,
	}

	err := cmd.Run(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--keep-hierarchy only applies with --q")
}
