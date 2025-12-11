package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	configAllInOneLocal = "../deployments/config-all-in-one-local.yaml"

	tempoPort = 3200

	blockFlushTimeout = 6 * time.Second
)

type batchTmpl struct {
	spanCount                  int
	name                       string
	resourceAttVal, spanAttVal string
	resourceAttr, SpanAttr     string
}

func TestTagEndpoints(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components: util.ComponentRecentDataQuerying | util.ComponentsBackendQuerying | util.ComponentsObjectStorage,
	}, func(h *util.TempoHarness) {

		batches := []batchTmpl{
			{spanCount: 2, name: "foo", resourceAttr: "firstRes", resourceAttVal: "bar", SpanAttr: "firstSpan", spanAttVal: "bar"},
			{spanCount: 2, name: "baz", resourceAttr: "secondRes", resourceAttVal: "qux", SpanAttr: "secondSpan", spanAttVal: "qux"},
			{spanCount: 2, name: "foo", resourceAttr: "twoRes", resourceAttVal: "bar", SpanAttr: "twoSpan", spanAttVal: "bar"},
			{spanCount: 2, name: "baz", resourceAttr: "twoRes", resourceAttVal: "qux", SpanAttr: "twoSpan", spanAttVal: "qux"},
		}

		for _, b := range batches {
			batch := util.MakeThriftBatchWithSpanCountAttributeAndName(b.spanCount, b.name, b.resourceAttVal, b.spanAttVal, b.resourceAttr, b.SpanAttr)
			require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch))
		}

		// wait for the 2 traces to be written to the WAL
		liveStoreZoneA := h.Services[util.ServiceLiveStoreZoneA]
		require.NoError(t, liveStoreZoneA.WaitSumMetricsWithOptions(e2e.Equals(4), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

		tagsTestCases := buildSearchTagsV2TestCases(batches)
		tagValuesTestCases := buildSearchTagValuesV2TestCases(batches)

		queryFrontend := h.Services[util.ServiceQueryFrontend]
		for _, tc := range tagsTestCases {
			t.Run("tags_wal_"+tc.name, func(t *testing.T) {
				callSearchTagsV2AndAssert(t, queryFrontend, tc.scope, tc.query, tc.expected, 0, 0)
			})
		}

		for _, tc := range tagValuesTestCases {
			t.Run("values_wal_"+tc.name, func(t *testing.T) {
				callSearchTagValuesV2AndAssert(t, queryFrontend, tc.tagName, tc.query, tc.expected, 0, 0)
			})
		}

		// wait for 4 objects to be written to the backend
		require.NoError(t, queryFrontend.WaitSumMetricsWithOptions(e2e.Equals(4), []string{"tempodb_backend_objects_total"}, e2e.WaitMissingMetrics)) // jpe - use as the gate for restarting for backend reads

		frontend := h.Services[util.ServiceQueryFrontend]
		require.NoError(t, h.RestartServiceWithConfigOverlay(t, frontend, "../util/config-query-backend.yaml"))

		// Assert tags on storage backend
		now := time.Now()
		start := now.Add(-2 * time.Hour)
		end := now.Add(2 * time.Hour)

		for _, tc := range tagsTestCases {
			t.Run("tags_backend_"+tc.name, func(t *testing.T) {
				callSearchTagsV2AndAssert(t, queryFrontend, tc.scope, tc.query, tc.expected, start.Unix(), end.Unix())
			})
		}

		for _, tc := range tagValuesTestCases {
			t.Run("values_backend_"+tc.name, func(t *testing.T) {
				callSearchTagValuesV2AndAssert(t, queryFrontend, tc.tagName, tc.query, tc.expected, start.Unix(), end.Unix())
			})
		}
	})
}

func buildSearchTagsV2TestCases(batches []batchTmpl) []struct {
	name     string
	query    string
	scope    string
	expected searchTagsV2Response
} {
	return []struct {
		name     string
		query    string
		scope    string
		expected searchTagsV2Response
	}{
		{
			name:  "no filtering",
			query: "",
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "invalid query",
			query: ` { a="test" } `,
			scope: "none",
			// same results as no filtering
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "first batch - resource",
			query: fmt.Sprintf(`{ name="%s" }`, batches[0].name),
			scope: "resource",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch with incomplete query - span",
			query: fmt.Sprintf(`{ name="%s" && span.twoSpan = }`, batches[1].name),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[1].SpanAttr, batches[3].SpanAttr},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, batches[0].resourceAttr, batches[0].resourceAttVal),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - resource",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, batches[0].resourceAttr, batches[0].resourceAttVal),
			scope: "resource",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch - resource attribute - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, batches[1].resourceAttr, batches[1].resourceAttVal),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[1].SpanAttr},
					},
				},
			},
		},
		{
			name:  "too restrictive query",
			query: fmt.Sprintf(`{ resource.%s="%s" && resource.y="%s" }`, batches[0].resourceAttr, batches[0].resourceAttVal, batches[1].resourceAttVal),
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "resource",
						Tags: []string{"service.name"}, // well known column so included
					},
				},
			},
		},
		// Unscoped not supported, unfiltered results.
		{
			name:  "unscoped span attribute",
			query: fmt.Sprintf(`{ .twoSpan="%s" }`, batches[0].spanAttVal),
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "unscoped res attribute",
			query: fmt.Sprintf(`{ .twoRes="%s" }`, batches[0].resourceAttVal),
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "both batches - name and resource attribute",
			query: `{ resource.service.name="my-service"}`,
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "bad query - unfiltered results",
			query: fmt.Sprintf("%s = bar", "span.twoSpan"), // bad query, missing quotes
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{batches[0].SpanAttr, batches[1].SpanAttr, batches[2].SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{batches[0].resourceAttr, batches[1].resourceAttr, batches[2].resourceAttr, "service.name"},
					},
				},
			},
		},
	}
}

func buildSearchTagValuesV2TestCases(batches []batchTmpl) []struct {
	name     string
	query    string
	tagName  string
	expected searchTagValuesV2Response
} {
	return []struct {
		name     string
		query    string
		tagName  string
		expected searchTagValuesV2Response
	}{
		{
			name:    "no filtering",
			query:   "",
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "first batch - name",
			query:   fmt.Sprintf(`{ name="%s" }`, batches[2].name),
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].spanAttVal}},
			},
		},
		{
			name:    "second batch with incomplete query - name",
			query:   fmt.Sprintf(`{ name="%s" && span.twoSpan = }`, batches[3].name),
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "first batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[2].resourceAttVal),
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].spanAttVal}},
			},
		},
		{
			name:    "second batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[3].resourceAttVal),
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:     "too restrictive query",
			query:    fmt.Sprintf(`{ %s="%s" && resource.y="%s" }`, "resource.twoRes", batches[2].resourceAttVal, batches[3].resourceAttVal),
			tagName:  "span.twoSpan",
			expected: searchTagValuesV2Response{TagValues: []TagValue{}},
		},
		// Unscoped not supported, unfiltered results.
		{
			name:    "unscoped span attribute",
			query:   fmt.Sprintf(`{ .twoSpan="%s" }`, batches[2].spanAttVal),
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "unscoped resource attribute",
			query:   fmt.Sprintf(`{ .twoRes="%s" }`, batches[2].spanAttVal),
			tagName: "resource.twoRes",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].resourceAttVal}, {Type: "string", Value: batches[3].resourceAttVal}},
			},
		},
		{
			name:    "first batch - name and resource attribute",
			query:   fmt.Sprintf(`{ name="%s" }`, batches[2].name),
			tagName: "resource.service.name",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "both batches - name and resource attribute",
			query:   `{ resource.service.name="my-service"}`,
			tagName: "name",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[3].name}, {Type: "string", Value: batches[2].name}},
			},
		},
		{
			name:    "only resource attributes",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[2].resourceAttVal),
			tagName: "resource.service.name",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "bad query - unfiltered results",
			query:   fmt.Sprintf("%s = bar", "span.twoSpan"), // bad query, missing quotes
			tagName: "span.twoSpan",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
	}
}

func TestTraceByIDandTraceQL(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{
		Components: util.ComponentRecentDataQuerying | util.ComponentsBackendQuerying | util.ComponentsObjectStorage,
	}, func(h *util.TempoHarness) {
		countTraces := 10
		infos := make([]*tempoUtil.TraceInfo, 0, countTraces)

		for range countTraces {
			time.Sleep(time.Nanosecond) // force a new seed
			info := tempoUtil.NewTraceInfo(time.Now(), "")
			infos = append(infos, info)
			require.NoError(t, info.EmitAllBatches(h.JaegerExporter))
		}

		liveStoreZoneA := h.Services[util.ServiceLiveStoreZoneA]
		require.NoError(t, liveStoreZoneA.WaitSumMetricsWithOptions(e2e.Equals(float64(countTraces)), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

		grpcClient, err := util.NewSearchGRPCClient(context.Background(), h.QueryFrontendGRPCEndpoint)
		require.NoError(t, err)

		now := time.Now()
		for _, i := range infos {
			util.QueryAndAssertTrace(t, h.HTTPClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, h.HTTPClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix()) // jpe - keep? remove range
			util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}

		queryFrontend := h.Services[util.ServiceQueryFrontend]
		require.NoError(t, queryFrontend.WaitSumMetricsWithOptions(e2e.Equals(float64(countTraces)), []string{"tempodb_backend_objects_total"}, e2e.WaitMissingMetrics))
		require.NoError(t, h.RestartServiceWithConfigOverlay(t, queryFrontend, "../util/config-query-backend.yaml"))

		grpcClient, err = util.NewSearchGRPCClient(context.Background(), h.QueryFrontendGRPCEndpoint)
		require.NoError(t, err)

		// Assert tags on storage backend
		for _, i := range infos {
			util.QueryAndAssertTrace(t, h.HTTPClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, h.HTTPClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
			util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}
	})
}

func TestStreamingSearch_badRequest(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		// Send a batch of traces
		batch := util.MakeThriftBatch()
		require.NoError(t, h.JaegerExporter.EmitBatch(context.Background(), batch))

		// Wait for the traces to be written to the WAL
		liveStoreZoneA := h.Services[util.ServiceLiveStoreZoneA]
		require.NoError(t, liveStoreZoneA.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempo_live_store_traces_created_total"}, e2e.WaitMissingMetrics))

		// Create gRPC client
		c, err := util.NewSearchGRPCClient(context.Background(), h.QueryFrontendGRPCEndpoint)
		require.NoError(t, err)

		// Send invalid search query (missing operator)
		res, err := c.Search(context.Background(), &tempopb.SearchRequest{
			Query: "{resource.service.name=article}",
		})
		require.NoError(t, err)

		// Expect error on receive
		_, err = res.Recv()
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestSearchTagValuesV2_badRequest(t *testing.T) {
	util.WithTempoHarness(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		// Test HTTP endpoint returns 400 for invalid tagName
		invalidTagName := "app.user.id" // not a valid scoped attribute
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("http://%s/api/v2/search/tag/%s/values", h.QueryFrontendHTTPEndpoint, invalidTagName), nil)
		require.NoError(t, err)

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Contains(t, string(body), "tag name is not valid intrinsic or scoped attribute")

		// Test gRPC endpoint returns InvalidArgument for invalid tagName
		grpcClient, err := util.NewSearchGRPCClient(context.Background(), h.QueryFrontendGRPCEndpoint)
		require.NoError(t, err)

		stream, err := grpcClient.SearchTagValuesV2(context.Background(), &tempopb.SearchTagValuesRequest{
			TagName: invalidTagName,
		})
		require.NoError(t, err)

		_, err = stream.Recv()
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
		require.Contains(t, st.Message(), "tag name is not valid intrinsic or scoped attribute")
	})
}

func callSearchTagValuesV2AndAssert(t *testing.T, svc *e2e.HTTPService, tagName, query string, expected searchTagValuesV2Response, start, end int64) {
	urlPath := fmt.Sprintf(`/api/v2/search/tag/%s/values?q=%s`, tagName, url.QueryEscape(query))

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(tempoPort)+urlPath, nil)
	require.NoError(t, err)

	q := req.URL.Query()

	if start != 0 {
		q.Set("start", strconv.Itoa(int(start)))
	}

	if end != 0 {
		q.Set("end", strconv.Itoa(int(end)))
	}

	req.URL.RawQuery = q.Encode()

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response searchTagValuesV2Response
	require.NoError(t, json.Unmarshal(body, &response))
	sort.Slice(response.TagValues, func(i, j int) bool { return response.TagValues[i].Value < response.TagValues[j].Value })
	require.Equal(t, expected.TagValues, response.TagValues)
	assertMetrics(t, response.Metrics, len(expected.TagValues))

	// streaming
	grpcReq := &tempopb.SearchTagValuesRequest{
		TagName: tagName,
		Query:   query,
		Start:   uint32(start),
		End:     uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(tempoPort))
	require.NoError(t, err)

	respTagsValuesV2, err := grpcClient.SearchTagValuesV2(context.Background(), grpcReq)
	require.NoError(t, err)
	var grpcResp *tempopb.SearchTagValuesV2Response
	for {
		resp, err := respTagsValuesV2.Recv()
		if resp != nil {
			grpcResp = resp
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NotNil(t, grpcResp)
	actualGrpcResp := searchTagValuesV2Response{TagValues: []TagValue{}}
	for _, tagValue := range grpcResp.TagValues {
		actualGrpcResp.TagValues = append(actualGrpcResp.TagValues, TagValue{Type: tagValue.Type, Value: tagValue.Value})
	}
	sort.Slice(actualGrpcResp.TagValues, func(i, j int) bool { return grpcResp.TagValues[i].Value < grpcResp.TagValues[j].Value })
	require.Equal(t, expected.TagValues, actualGrpcResp.TagValues)
	// assert metrics, and make sure it's non-zero when response is non-empty
	if len(grpcResp.TagValues) > 0 {
		require.Greater(t, grpcResp.Metrics.InspectedBytes, uint64(0))
	}
}

func callSearchTagsV2AndAssert(t *testing.T, svc *e2e.HTTPService, scope, query string, expected searchTagsV2Response, start, end int64) {
	urlPath := fmt.Sprintf(`/api/v2/search/tags?scope=%s&q=%s`, scope, url.QueryEscape(query))

	// Expected will not have the intrinsic results to make the tests simpler,
	// they are added here based on the scope.
	if scope == "none" || scope == "" || scope == "intrinsic" {
		expected.Scopes = append(expected.Scopes, ScopedTags{
			Name: "intrinsic",
			Tags: search.GetVirtualIntrinsicValues(),
		})
	}
	sort.Slice(expected.Scopes, func(i, j int) bool { return expected.Scopes[i].Name < expected.Scopes[j].Name })
	for _, scope := range expected.Scopes {
		slices.Sort(scope.Tags)
	}

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(tempoPort)+urlPath, nil)
	require.NoError(t, err)

	q := req.URL.Query()

	if start != 0 {
		q.Set("start", strconv.Itoa(int(start)))
	}

	if end != 0 {
		q.Set("end", strconv.Itoa(int(end)))
	}

	req.URL.RawQuery = q.Encode()

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response searchTagsV2Response
	require.NoError(t, json.Unmarshal(body, &response))

	prepTagsResponse(&response)
	require.Equal(t, expected.Scopes, response.Scopes)
	assertMetrics(t, response.Metrics, lenWithoutIntrinsic(response))

	// streaming
	grpcReq := &tempopb.SearchTagsRequest{
		Scope: scope,
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(tempoPort))
	require.NoError(t, err)

	respTagsValuesV2, err := grpcClient.SearchTagsV2(context.Background(), grpcReq)
	require.NoError(t, err)
	var grpcResp *tempopb.SearchTagsV2Response
	for {
		resp, err := respTagsValuesV2.Recv()
		if resp != nil {
			grpcResp = resp
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NotNil(t, grpcResp)
	require.NotNil(t, grpcResp.Metrics)

	prepTagsResponse(&response)
	require.Equal(t, expected.Scopes, response.Scopes)
	// assert metrics, and make sure it's non-zero when response is non-empty
	if lenWithoutIntrinsic(response) > 0 {
		require.Greater(t, grpcResp.Metrics.InspectedBytes, uint64(100))
	}
}

func prepTagsResponse(resp *searchTagsV2Response) {
	if len(resp.Scopes) == 0 {
		resp.Scopes = nil
	}
	sort.Slice(resp.Scopes, func(i, j int) bool { return resp.Scopes[i].Name < resp.Scopes[j].Name })
	for _, scope := range resp.Scopes {
		if len(scope.Tags) == 0 {
			scope.Tags = nil
		}

		slices.Sort(scope.Tags)
	}
}

func callSearchTagsAndAssert(t *testing.T, svc *e2e.HTTPService, expected searchTagsResponse, start, end int64) {
	urlPath := "/api/search/tags"
	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(tempoPort)+urlPath, nil)
	require.NoError(t, err)

	q := req.URL.Query()

	if start != 0 {
		q.Set("start", strconv.Itoa(int(start)))
	}

	if end != 0 {
		q.Set("end", strconv.Itoa(int(end)))
	}

	req.URL.RawQuery = q.Encode()

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response searchTagsResponse
	require.NoError(t, json.Unmarshal(body, &response))
	require.ElementsMatch(t, expected.TagNames, response.TagNames)
	assertMetrics(t, response.Metrics, len(response.TagNames))

	// streaming
	grpcReq := &tempopb.SearchTagsRequest{
		Start: uint32(start),
		End:   uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(tempoPort))
	require.NoError(t, err)

	respTags, err := grpcClient.SearchTags(context.Background(), grpcReq)
	require.NoError(t, err)
	var grpcResp *tempopb.SearchTagsResponse
	for {
		resp, err := respTags.Recv()
		if resp != nil {
			grpcResp = resp
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NotNil(t, grpcResp)

	if grpcResp.TagNames == nil {
		grpcResp.TagNames = []string{}
	}

	require.ElementsMatch(t, expected.TagNames, grpcResp.TagNames)
	// assert metrics, and make sure it's non-zero when response is non-empty
	if len(grpcResp.TagNames) > 0 {
		require.Greater(t, grpcResp.Metrics.InspectedBytes, uint64(100))
	}
}

func callSearchTagValuesAndAssert(t *testing.T, svc *e2e.HTTPService, tagName string, expected searchTagValuesResponse, start, end int64) {
	urlPath := fmt.Sprintf(`/api/search/tag/%s/values`, tagName)
	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(tempoPort)+urlPath, nil)
	require.NoError(t, err)

	q := req.URL.Query()

	if start != 0 {
		q.Set("start", strconv.Itoa(int(start)))
	}

	if end != 0 {
		q.Set("end", strconv.Itoa(int(end)))
	}

	req.URL.RawQuery = q.Encode()

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()

	// parse response
	var response searchTagValuesResponse
	require.NoError(t, json.Unmarshal(body, &response))
	sort.Strings(response.TagValues)
	sort.Strings(expected.TagValues)

	require.Equal(t, expected.TagValues, response.TagValues)
	assertMetrics(t, response.Metrics, len(response.TagValues))
}

func assertMetrics(t *testing.T, metrics MetadataMetrics, respLen int) {
	// metrics are not present when response is empty, so return
	if respLen == 0 {
		return
	}

	require.NotNil(t, metrics)
	require.NotEmpty(t, metrics.InspectedBytes)
	inspectedBytes, err := strconv.ParseUint(metrics.InspectedBytes, 10, 64)
	require.NoError(t, err)
	// if response len is empty, then the inspected bytes should be 0
	// assert metrics, and make sure it's non-zero
	require.Greater(t, inspectedBytes, uint64(300))
}

type searchTagsV2Response struct {
	Scopes  []ScopedTags    `json:"scopes"`
	Metrics MetadataMetrics `json:"metrics"`
}

func lenWithoutIntrinsic(resp searchTagsV2Response) int {
	size := 0
	for _, scope := range resp.Scopes {
		// we don't count intrinsics as results for testing
		if scope.Name == "intrinsic" {
			continue
		}
		size += len(scope.Tags)
	}
	return size
}

// isQueryable returns true if the data is queryable and not empty
func isQueryable(host string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/search/tag/service.name/values", host))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status code is not ok: %d", resp.StatusCode)
	}

	var searchResp searchTagValuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return false, err
	}

	return len(searchResp.TagValues) > 0, nil
}

type ScopedTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type MetadataMetrics struct {
	InspectedBytes  string `json:"inspectedBytes"` // String to match JSON format
	TotalJobs       string `json:"totalJobs"`
	CompletedJobs   string `json:"completedJobs"`
	TotalBlocks     string `json:"totalBlocks"`
	TotalBlockBytes string `json:"totalBlockBytes"`
}

type searchTagValuesV2Response struct {
	TagValues []TagValue      `json:"tagValues"`
	Metrics   MetadataMetrics `json:"metrics"`
}

type searchTagValuesResponse struct {
	TagValues []string        `json:"tagValues"`
	Metrics   MetadataMetrics `json:"metrics"`
}

type searchTagsResponse struct {
	TagNames []string        `json:"tagNames"`
	Metrics  MetadataMetrics `json:"metrics"`
}

type TagValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
