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

	// Org ID must be present in the outgoing gRPC metadata; it must NOT appear in the body.
	md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
	require.True(t, ok, "expected outgoing metadata on context")
	require.Equal(t, []string{tenant}, md["x-scope-orgid"])
	require.Empty(t, mock.capturedReq.TenantId, "tenant must not be sent in the request body")
	require.Equal(t, [][]byte{traceIDBytes}, mock.capturedReq.TraceIds)
}

func TestRedactCmdSubmitQuery(t *testing.T) {
	const query = `{resource.service_name = "checkout"}`

	mock := &mockSchedulerClient{}
	cmd := &redactCmd{TenantID: "test-tenant", Query: query}

	_, err := cmd.submit(context.Background(), mock, nil)
	require.NoError(t, err)

	require.NotNil(t, mock.capturedReq.GetQuery(), "query selector must be set")
	require.Equal(t, query, mock.capturedReq.GetQuery().GetQuery())
	require.Empty(t, mock.capturedReq.TraceIds, "query submission carries no trace IDs")
	require.Equal(t, tempopb.RedactionMode_REDACTION_MODE_APPLY, mock.capturedReq.Mode)
}

func TestRedactCmdSubmitDryRun(t *testing.T) {
	mock := &mockSchedulerClient{}
	cmd := &redactCmd{TenantID: "test-tenant", Query: `{resource.service_name = "x"}`, DryRun: true}

	_, err := cmd.submit(context.Background(), mock, nil)
	require.NoError(t, err)
	require.Equal(t, tempopb.RedactionMode_REDACTION_MODE_DRY_RUN, mock.capturedReq.Mode)
}

func TestRedactCmdValidate(t *testing.T) {
	cases := []struct {
		name    string
		cmd     redactCmd
		wantErr bool
	}{
		{"trace-ids only", redactCmd{TraceIDs: []string{"abc"}}, false},
		{"query only", redactCmd{Query: `{resource.service_name = "x"}`}, false},
		{"both", redactCmd{TraceIDs: []string{"abc"}, Query: `{resource.service_name = "x"}`}, true},
		{"neither", redactCmd{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
