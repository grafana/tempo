package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// mockSchedulerClient captures the context and request from SubmitRedaction calls.
type mockSchedulerClient struct {
	tempopb.BackendSchedulerClient
	capturedCtx context.Context
	capturedReq *tempopb.SubmitRedactionRequest
}

func (m *mockSchedulerClient) SubmitRedaction(ctx context.Context, req *tempopb.SubmitRedactionRequest, _ ...grpc.CallOption) (*tempopb.SubmitRedactionResponse, error) {
	m.capturedCtx = ctx
	m.capturedReq = req
	return &tempopb.SubmitRedactionResponse{BatchId: "test-batch", JobsCreated: 1}, nil
}

func TestRedactCmdSubmit(t *testing.T) {
	const (
		tenant     = "test-tenant"
		traceIDHex = "931281e2a09876de16e15f45ff86283d"
	)

	traceIDBytes, err := util.HexStringToTraceID(traceIDHex)
	require.NoError(t, err)

	mock := &mockSchedulerClient{}
	cmd := &redactCmd{TenantID: tenant}

	resp, err := cmd.submit(context.Background(), mock, [][]byte{traceIDBytes})
	require.NoError(t, err)
	require.Equal(t, "test-batch", resp.BatchId)

	// Org ID must be present in the outgoing gRPC metadata.
	md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
	require.True(t, ok, "expected outgoing metadata on context")
	require.Equal(t, []string{tenant}, md["x-scope-orgid"])

	// Request body must carry the tenant and trace IDs.
	require.Equal(t, tenant, mock.capturedReq.TenantId)
	require.Equal(t, [][]byte{traceIDBytes}, mock.capturedReq.TraceIds)
}

func TestParseTraceIDs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ids, err := parseTraceIDs([]string{
			"931281e2a09876de16e15f45ff86283d",
			"00000000000000000000000000000001",
		})
		require.NoError(t, err)
		require.Len(t, ids, 2)
	})

	t.Run("invalid hex", func(t *testing.T) {
		_, err := parseTraceIDs([]string{"not-a-trace-id"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid trace ID")
	})
}
