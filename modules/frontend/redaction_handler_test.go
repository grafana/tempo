package frontend

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/pkg/tempopb"
)

type mockRedactionClient struct {
	tempopb.BackendSchedulerClient
	submit func(context.Context, *tempopb.SubmitRedactionRequest) (*tempopb.SubmitRedactionResponse, error)
}

func (m *mockRedactionClient) SubmitRedaction(ctx context.Context, req *tempopb.SubmitRedactionRequest, _ ...grpc.CallOption) (*tempopb.SubmitRedactionResponse, error) {
	return m.submit(ctx, req)
}

func TestRedactionHandlerSubmit(t *testing.T) {
	const traceID = "931281e2a09876de16e15f45ff86283d"
	client := &mockRedactionClient{
		submit: func(ctx context.Context, req *tempopb.SubmitRedactionRequest) (*tempopb.SubmitRedactionResponse, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			require.True(t, ok)
			require.Equal(t, []string{"tenant-a"}, md.Get(user.OrgIDHeaderName))
			require.Equal(t, tempopb.RedactionMode_REDACTION_MODE_APPLY, req.Mode)
			require.Len(t, req.TraceIds, 1)
			require.Equal(t, traceID, hex.EncodeToString(req.TraceIds[0]))
			require.Nil(t, req.GetQuery())
			return &tempopb.SubmitRedactionResponse{BatchId: "84413dad-684f-4e3b-8283-06dc50bb74c7", JobsCreated: 3}, nil
		},
	}
	handler := NewRedactionHandler(client)
	request := httptest.NewRequest(http.MethodPost, "/api/redactions", strings.NewReader(`{"trace_ids":["931281E2A09876DE16E15F45FF86283D","931281e2a09876de16e15f45ff86283d"]}`))
	request = request.WithContext(user.InjectOrgID(request.Context(), "tenant-a"))
	response := httptest.NewRecorder()

	handler.Submit(response, request)

	require.Equal(t, http.StatusAccepted, response.Code)
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.JSONEq(t, `{"batch_id":"84413dad-684f-4e3b-8283-06dc50bb74c7","jobs_created":3}`, response.Body.String())
}

func TestRedactionHandlerSubmitRejectsMutableSelectorsAndInvalidScope(t *testing.T) {
	calls := 0
	client := &mockRedactionClient{
		submit: func(context.Context, *tempopb.SubmitRedactionRequest) (*tempopb.SubmitRedactionResponse, error) {
			calls++
			return nil, nil
		},
	}
	handler := NewRedactionHandler(client)

	tests := []struct {
		name   string
		tenant string
		body   string
	}{
		{name: "query field", tenant: "tenant-a", body: `{"query":"{ true }"}`},
		{name: "multiple tenants", tenant: "tenant-a|tenant-b", body: `{"trace_ids":["931281e2a09876de16e15f45ff86283d"]}`},
		{name: "short trace ID", tenant: "tenant-a", body: `{"trace_ids":["1234"]}`},
		{name: "zero trace ID", tenant: "tenant-a", body: `{"trace_ids":["00000000000000000000000000000000"]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/redactions", strings.NewReader(tc.body))
			request = request.WithContext(user.InjectOrgID(request.Context(), tc.tenant))
			response := httptest.NewRecorder()
			handler.Submit(response, request)
			require.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
	require.Zero(t, calls)
}

func TestRedactionHandlerMapsSchedulerErrors(t *testing.T) {
	client := &mockRedactionClient{
		submit: func(context.Context, *tempopb.SubmitRedactionRequest) (*tempopb.SubmitRedactionResponse, error) {
			return nil, status.Error(codes.ResourceExhausted, "redaction capacity exhausted")
		},
	}
	handler := NewRedactionHandler(client)
	request := httptest.NewRequest(http.MethodPost, "/api/redactions", strings.NewReader(`{"trace_ids":["931281e2a09876de16e15f45ff86283d"]}`))
	request = request.WithContext(user.InjectOrgID(request.Context(), "tenant-a"))
	response := httptest.NewRecorder()

	handler.Submit(response, request)

	require.Equal(t, http.StatusTooManyRequests, response.Code)
	require.JSONEq(t, `{"error":"redaction capacity exhausted"}`, response.Body.String())
}
