package external

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/stretchr/testify/require"
)

func TestClient_TraceByID(t *testing.T) {
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	userID := "test-tenant"
	expectedPath := "/api/v2/traces/" + hex.EncodeToString(traceID)

	// Create a test trace to return
	testTrace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Spans: []*v1_trace.Span{
							{
								TraceId: traceID,
								SpanId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
								Name:    "test-span",
							},
						},
					},
				},
			},
		},
	}

	// Create httptest server that validates the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate path
		require.Equal(t, expectedPath, r.URL.Path, "path should match expected trace path")

		// Validate headers
		require.Equal(t, api.HeaderAcceptProtobuf, r.Header.Get(api.HeaderAccept), "Accept header should be protobuf")
		require.Equal(t, userID, r.Header.Get(user.OrgIDHeaderName), "X-Scope-OrgID header should match userID")

		// Validate method
		require.Equal(t, http.MethodGet, r.Method, "method should be GET")

		// Validate query parameters
		require.Equal(t, "123", r.URL.Query().Get("start"), "start query parameter should be 123")
		require.Equal(t, "456", r.URL.Query().Get("end"), "end query parameter should be 456")

		// Marshal and return the trace
		traceBytes, err := testTrace.Marshal()
		require.NoError(t, err)

		w.Header().Set("Content-Type", api.HeaderAcceptProtobuf)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(traceBytes)
		require.NoError(t, err)
	}))
	defer server.Close()

	// Create client
	client, err := NewClient(server.URL, 10*time.Second)
	require.NoError(t, err)

	// Call TraceByID
	ctx := context.Background()
	resp, err := client.TraceByID(ctx, userID, traceID, 123, 456)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Trace)
	require.Len(t, resp.Trace.ResourceSpans, 1)
	require.Len(t, resp.Trace.ResourceSpans[0].ScopeSpans, 1)
	require.Len(t, resp.Trace.ResourceSpans[0].ScopeSpans[0].Spans, 1)
	require.Equal(t, "test-span", resp.Trace.ResourceSpans[0].ScopeSpans[0].Spans[0].Name)
}
