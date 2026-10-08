package querier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/user"
	livestore_client "github.com/grafana/tempo/v3/modules/livestore/client"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/modules/querier/worker"
	"github.com/grafana/tempo/v3/modules/storage"
	"github.com/grafana/tempo/v3/pkg/api"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestVirtualTagsDoesntHitBackend(t *testing.T) {
	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	q, err := New(Config{}, nil, livestore_client.Config{}, nil, false, nil, o)
	require.NoError(t, err)

	ctx := user.InjectOrgID(context.Background(), "blerg")

	// duration should return nothing
	resp, err := q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "duration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{Metrics: &tempopb.MetadataMetrics{}}, resp)

	// traceDuration should return nothing
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "traceDuration",
	})
	require.NoError(t, err)
	require.Equal(t, &tempopb.SearchTagValuesV2Response{Metrics: &tempopb.MetadataMetrics{}}, resp)

	// status should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "status",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "error",
			},
			{
				Type:  "keyword",
				Value: "ok",
			},
			{
				Type:  "keyword",
				Value: "unset",
			},
		},
		Metrics: &tempopb.MetadataMetrics{},
	}, resp)

	// kind should return a static list
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: "kind",
	})
	require.NoError(t, err)
	sort.Slice(resp.TagValues, func(i, j int) bool { return resp.TagValues[i].Value < resp.TagValues[j].Value })
	require.Equal(t, &tempopb.SearchTagValuesV2Response{
		TagValues: []*tempopb.TagValue{
			{
				Type:  "keyword",
				Value: "client",
			},
			{
				Type:  "keyword",
				Value: "consumer",
			},
			{
				Type:  "keyword",
				Value: "internal",
			},
			{
				Type:  "keyword",
				Value: "producer",
			},
			{
				Type:  "keyword",
				Value: "server",
			},
			{
				Type:  "keyword",
				Value: "unspecified",
			},
		},
		Metrics: &tempopb.MetadataMetrics{},
	}, resp)

	// this should error b/c it will attempt to hit the un-configured backend
	resp, err = q.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
		TagName: ".foo",
	})
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestPostProcessIngesterSearchResultsMergesReadMetrics(t *testing.T) {
	q := &Querier{}

	resp := q.postProcessIngesterSearchResults(&tempopb.SearchRequest{}, []any{
		&tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 1,
				InspectedBytes:  2,
				InspectedSpans:  3,
				BackendReads:    4,
				BackendBytes:    5,
				AdditionalMetrics: map[string]int64{
					tempopb.AdditionalMetricCacheHits: 6,
				},
			},
		},
		&tempopb.SearchResponse{
			Metrics: &tempopb.SearchMetrics{
				InspectedTraces: 7,
				InspectedBytes:  8,
				InspectedSpans:  9,
				BackendReads:    10,
				BackendBytes:    11,
				AdditionalMetrics: map[string]int64{
					tempopb.AdditionalMetricCacheHits:   12,
					tempopb.AdditionalMetricCacheMisses: 13,
				},
			},
		},
	})

	require.Equal(t, &tempopb.SearchMetrics{
		InspectedTraces: 1 + 7,
		InspectedBytes:  2 + 8,
		InspectedSpans:  3 + 9,
		BackendReads:    4 + 10,
		BackendBytes:    5 + 11,
		AdditionalMetrics: map[string]int64{
			tempopb.AdditionalMetricCacheHits:   6 + 12,
			tempopb.AdditionalMetricCacheMisses: 13,
		},
	}, resp.Metrics)
}

func TestFindTraceByID_ExternalMode(t *testing.T) {
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	userID := "test-tenant"

	externalTrace := &tempopb.Trace{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Spans: []*v1_trace.Span{
							{
								TraceId: traceID,
								SpanId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
								Name:    "external-span",
							},
						},
					},
				},
			},
		},
	}

	startTime := int64(1000)
	endTime := int64(2000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that start and end query parameters are present
		require.Equal(t, strconv.FormatInt(startTime, 10), r.URL.Query().Get("start"))
		require.Equal(t, strconv.FormatInt(endTime, 10), r.URL.Query().Get("end"))

		traceBytes, err := externalTrace.Marshal()
		require.NoError(t, err)

		w.Header().Set("Content-Type", api.HeaderAcceptProtobuf)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(traceBytes)
		require.NoError(t, err)
	}))
	defer server.Close()

	cfg := Config{
		TraceByID: TraceByIDConfig{
			External: ExternalConfig{
				Endpoint: server.URL,
				Timeout:  10 * time.Second,
			},
		},
		Worker: worker.Config{
			GRPCClientConfig: grpcclient.Config{
				MaxSendMsgSize: 16 * 1024 * 1024,
			},
		},
	}

	o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.DefaultRegisterer)
	require.NoError(t, err)

	q, err := New(cfg, nil, livestore_client.Config{}, nil, true, nil, o)
	require.NoError(t, err)

	ctx := user.InjectOrgID(context.Background(), userID)

	resp, err := q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
		TraceID:   traceID,
		QueryMode: QueryModeExternal,
		Start:     time.Unix(startTime, 0),
		End:       time.Unix(endTime, 0),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Trace)
	require.Len(t, resp.Trace.ResourceSpans, 1)
	require.Len(t, resp.Trace.ResourceSpans[0].ScopeSpans, 1)
	require.Len(t, resp.Trace.ResourceSpans[0].ScopeSpans[0].Spans, 1)
	require.Equal(t, "external-span", resp.Trace.ResourceSpans[0].ScopeSpans[0].Spans[0].Name)
}

type mockTraceByIDStore struct {
	storage.Store
	findCalls    int
	findTenantID string
	findReq      *tempopb.TraceByIDRequest
}

func (m *mockTraceByIDStore) Find(_ context.Context, tenantID string, req *tempopb.TraceByIDRequest, _ common.SearchOptions) ([]*tempopb.TraceByIDResponse, []error, error) {
	m.findCalls++
	m.findTenantID = tenantID
	m.findReq = req
	return nil, nil, nil
}

func TestFindTraceByIDUsesFrontendBlocks(t *testing.T) {
	tests := []struct {
		name            string
		pollingDisabled bool
		blocks          *tempopb.TraceByIDBlocks
		expectErr       error
	}{
		{
			name:   "no blocks falls back to the polled blocklist",
			blocks: nil,
		},
		{
			name:            "no blocks without polling is rejected",
			pollingDisabled: true,
			blocks:          nil,
			expectErr:       ErrTraceByIDBlocksRequired,
		},
		{
			name:   "blocks are searched with polling",
			blocks: &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{{Version: "vParquet5"}}},
		},
		{
			name:            "blocks are searched without polling",
			pollingDisabled: true,
			blocks:          &tempopb.TraceByIDBlocks{Blocks: []*tempopb.TraceByIDBlock{{Version: "vParquet5"}}},
		},
		{
			name:            "empty blocks are searched, not rejected",
			pollingDisabled: true,
			blocks:          &tempopb.TraceByIDBlocks{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, err := overrides.NewOverrides(overrides.Config{}, nil, prometheus.NewRegistry())
			require.NoError(t, err)

			store := &mockTraceByIDStore{}
			q, err := New(Config{BlocklistPollingEnabled: !tc.pollingDisabled}, nil, livestore_client.Config{}, nil, false, store, o)
			require.NoError(t, err)

			ctx := user.InjectOrgID(context.Background(), "blerg")
			_, err = q.FindTraceByID(ctx, &tempopb.TraceByIDRequest{
				TraceID:   test.ValidTraceID(nil),
				QueryMode: QueryModeBlocks,
				Blocks:    tc.blocks,
			})
			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Equal(t, 0, store.findCalls)
				return
			}
			require.NoError(t, err)

			require.Equal(t, 1, store.findCalls)
			// tenant comes from the org id, never from the request payload
			require.Equal(t, "blerg", store.findTenantID)
			require.Equal(t, tc.blocks, store.findReq.Blocks)
		})
	}
}
