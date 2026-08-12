package main

import (
	"context"
	"testing"
	"time"

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
	const q = `{resource.service_name = "x"}`

	cases := []struct {
		name    string
		cmd     redactCmd
		wantErr bool
	}{
		{"trace-ids only", redactCmd{TraceIDs: []string{"abc"}}, false},
		{"query only", redactCmd{Query: `{resource.service_name = "x"}`}, false},
		{"both", redactCmd{TraceIDs: []string{"abc"}, Query: `{resource.service_name = "x"}`}, true},
		{"neither", redactCmd{}, true},

		// Window resolution. Both bounds must be given, ordered, and parseable; the resolved
		// pair must also be usable, which is what the identical-spec cases below check.
		{"window both bounds ordered", redactCmd{Query: q, Start: "now-7d", End: "now-6d"}, false},
		{"window absolute bounds ordered", redactCmd{Query: q, Start: "2026-01-01T00:00:00Z", End: "2026-01-02T00:00:00Z"}, false},
		{"window start only", redactCmd{Query: q, Start: "now-7d"}, true},
		{"window end only", redactCmd{Query: q, End: "now-6d"}, true},
		{"window transposed", redactCmd{Query: q, Start: "now-6d", End: "now-7d"}, true},
		{"window unparseable", redactCmd{Query: q, Start: "yesterday", End: "now"}, true},

		// Identical specs must resolve to identical instants and be refused as a zero-width
		// window. Resolving each bound against its own time.Now() makes start fall a few
		// nanoseconds before end, so this sails through the ordering check and submits a window
		// nothing can match -- under-deletion reported as success.
		{"window identical relative specs", redactCmd{Query: q, Start: "now-7d", End: "now-7d"}, true},
		{"window identical now", redactCmd{Query: q, Start: "now", End: "now"}, true},
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

// TestRedactCmdCarriesResolvedWindow covers the two assignments that put the resolved window on the
// wire. Without them the CLI accepts --start/--end, reports success, and submits an unbounded
// request -- a whole-tenant redaction instead of the slice the operator asked for. Nothing else in
// the suite exercises validate() and submit() together, so nothing else can catch that.
func TestRedactCmdCarriesResolvedWindow(t *testing.T) {
	const q = `{resource.service_name = "x"}`

	t.Run("a resolved window reaches the request", func(t *testing.T) {
		cmd := &redactCmd{TenantID: "test-tenant", Query: q, Start: "now-7d", End: "now-6d"}
		require.NoError(t, cmd.validate())

		mock := &mockSchedulerClient{}
		_, err := cmd.submit(context.Background(), mock, nil)
		require.NoError(t, err)

		require.Equal(t, cmd.startNano, mock.capturedReq.StartTimeUnixNano, "resolved start must reach the wire")
		require.Equal(t, cmd.endNano, mock.capturedReq.EndTimeUnixNano, "resolved end must reach the wire")
		require.NotZero(t, mock.capturedReq.StartTimeUnixNano, "a submitted window must not be the unbounded sentinel")
		require.NotZero(t, mock.capturedReq.EndTimeUnixNano)
		require.InDelta(t, float64(24*time.Hour),
			float64(mock.capturedReq.EndTimeUnixNano-mock.capturedReq.StartTimeUnixNano),
			float64(2*time.Minute), "now-7d..now-6d is a one-day window")
	})

	t.Run("no window submits the unbounded sentinel", func(t *testing.T) {
		cmd := &redactCmd{TenantID: "test-tenant", Query: q}
		require.NoError(t, cmd.validate())

		mock := &mockSchedulerClient{}
		_, err := cmd.submit(context.Background(), mock, nil)
		require.NoError(t, err)

		require.Zero(t, mock.capturedReq.StartTimeUnixNano)
		require.Zero(t, mock.capturedReq.EndTimeUnixNano)
	})
}
