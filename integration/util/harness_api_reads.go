package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// BaseURL returns the base URL for the Tempo queryAPI. Some tests construct requests manually if
// they need to do something such as set custom headers or bad requests bodies. If you don't have
// a special need, use the client returned from APIClientHTTP or APIClientGRPC instead.
func (h *TempoHarness) BaseURL() string {
	return "http://" + h.Services[ServiceQueryFrontend].Endpoint(3200)
}

func (h *TempoHarness) APIClientHTTP(tenant string) *httpclient.Client {
	return httpclient.New(h.BaseURL(), tenant)
}

func (h *TempoHarness) APIClientGRPC(tenant string) (tempopb.StreamingQuerierClient, context.Context, error) {
	endpoint := h.Services[ServiceQueryFrontend].Endpoint(3200)

	ctx := context.Background()

	if tenant != "" {
		ctx = user.InjectOrgID(ctx, tenant)
		var err error
		ctx, err = user.InjectIntoGRPCRequest(ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	client, err := NewSearchGRPCClient(endpoint, insecure.NewCredentials())
	if err != nil {
		return nil, nil, err
	}
	return client, ctx, nil
}

func NewSearchGRPCClient(endpoint string, creds credentials.TransportCredentials) (tempopb.StreamingQuerierClient, error) {
	clientConn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return tempopb.NewStreamingQuerierClient(clientConn), nil
}

func SearchTraceQLAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	resp, err := client.SearchTraceQL(query)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

func SearchTraceQLAndAssertTraceWithRange(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	resp, err := client.SearchTraceQLWithRange(query, start, end)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

// SearchStreamAndAssertTrace will search and assert that the trace is present in the streamed results.
// nolint: revive
func SearchStreamAndAssertTrace(t *testing.T, ctx context.Context, client tempopb.StreamingQuerierClient, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	// -- assert search
	resp, err := client.Search(ctx, &tempopb.SearchRequest{
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	})
	require.NoError(t, err)

	// drain the stream until everything is returned while watching for the trace in question
	found := false
	for {
		resp, err := resp.Recv()
		if resp != nil {
			found = traceIDInResults(t, info.HexID(), resp)
			if found {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.True(t, found)
}

func traceIDInResults(t *testing.T, hexID string, resp *tempopb.SearchResponse) bool {
	for _, s := range resp.Traces {
		equal, err := tempoUtil.EqualHexStringTraceIDs(s.TraceID, hexID)
		require.NoError(t, err)
		if equal {
			return true
		}
	}

	return false
}

func QueryAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	// v2
	respV2, err := client.QueryTraceV2(info.HexID())
	require.NoError(t, err)

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	AssertEqualTrace(t, respV2.Trace, expected)

	// v1
	respV1, err := client.QueryTrace(info.HexID())
	require.NoError(t, err)

	AssertEqualTrace(t, respV1, expected)
}

func AssertEqualTrace(t *testing.T, a, b *tempopb.Trace) {
	t.Helper()
	trace.SortTraceAndAttributes(a)
	trace.SortTraceAndAttributes(b)

	assert.Equal(t, a, b)
}
