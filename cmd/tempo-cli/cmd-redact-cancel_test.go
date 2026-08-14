package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/pkg/tempopb"
)

// mockCancelClient captures the context from CancelRedaction calls.
type mockCancelClient struct {
	tempopb.BackendSchedulerClient
	capturedCtx context.Context
	capturedReq *tempopb.CancelRedactionRequest
}

func (m *mockCancelClient) CancelRedaction(ctx context.Context, req *tempopb.CancelRedactionRequest, _ ...grpc.CallOption) (*tempopb.CancelRedactionResponse, error) {
	m.capturedCtx = ctx
	m.capturedReq = req
	return &tempopb.CancelRedactionResponse{BatchId: "test-batch", PendingPurged: 3}, nil
}

func TestRedactCancelCmd(t *testing.T) {
	const tenant = "test-tenant"

	mock := &mockCancelClient{}
	cmd := &redactCancelCmd{TenantID: tenant}

	resp, err := cmd.cancel(context.Background(), mock)
	require.NoError(t, err)
	require.Equal(t, "test-batch", resp.BatchId)
	require.Equal(t, int32(3), resp.PendingPurged)

	// Tenant must ride the X-Scope-OrgID header, never a request body field.
	md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
	require.True(t, ok, "expected outgoing metadata on context")
	require.Equal(t, []string{tenant}, md["x-scope-orgid"])
}
