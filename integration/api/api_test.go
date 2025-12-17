package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/collector"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type batchTmpl struct {
	spanCount                  int
	name                       string
	resourceAttVal, spanAttVal string
	resourceAttr, SpanAttr     string
}

func TestTagEndpoints(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		batches := []batchTmpl{
			{spanCount: 2, name: "foo", resourceAttr: "firstRes", resourceAttVal: "bar", SpanAttr: "firstSpan", spanAttVal: "bar"},
			{spanCount: 2, name: "baz", resourceAttr: "secondRes", resourceAttVal: "qux", SpanAttr: "secondSpan", spanAttVal: "qux"},
			{spanCount: 2, name: "foo", resourceAttr: "twoRes", resourceAttVal: "bar", SpanAttr: "twoSpan", spanAttVal: "bar"},
			{spanCount: 2, name: "baz", resourceAttr: "twoRes", resourceAttVal: "qux", SpanAttr: "twoSpan", spanAttVal: "qux"},
		}

		for _, b := range batches {
			batch := util.MakeThriftBatchWithSpanCountAttributeAndName(b.spanCount, b.name, b.resourceAttVal, b.spanAttVal, b.resourceAttr, b.SpanAttr)
			require.NoError(t, h.WriteJaegerBatch(batch, ""))
		}

		// wait for the 2 traces to be written to the WAL
		h.WaitTracesQueryable(t, 4)

		tagsTestCases := buildSearchTagsV2TestCases(batches)
		tagValuesTestCases := buildSearchTagValuesV2TestCases(batches)

		for _, tc := range tagsTestCases {
			t.Run("tags_wal_"+tc.name, func(t *testing.T) {
				callSearchTagsV2AndAssert(t, h, tc.scope, tc.query, tc.expected, 0, 0)
			})
		}

		for _, tc := range tagValuesTestCases {
			t.Run("values_wal_"+tc.name, func(t *testing.T) {
				callSearchTagValuesV2AndAssert(t, h, tc.tagName, tc.query, tc.expected, 0, 0)
			})
		}

		// wait for 4 objects to be written to the backend
		h.WaitTracesWrittenToBackend(t, 4)
		h.ForceBackendQuerying(t)

		// Assert tags on storage backend
		now := time.Now()
		start := now.Add(-2 * time.Hour)
		end := now.Add(2 * time.Hour)

		for _, tc := range tagsTestCases {
			t.Run("tags_backend_"+tc.name, func(t *testing.T) {
				callSearchTagsV2AndAssert(t, h, tc.scope, tc.query, tc.expected, start.Unix(), end.Unix())
			})
		}

		for _, tc := range tagValuesTestCases {
			t.Run("values_backend_"+tc.name, func(t *testing.T) {
				callSearchTagValuesV2AndAssert(t, h, tc.tagName, tc.query, tc.expected, start.Unix(), end.Unix())
			})
		}
	})
}

func buildSearchTagsV2TestCases(batches []batchTmpl) []struct {
	name     string
	query    string
	scope    string
	expected *tempopb.SearchTagsV2Response
} {
	tcs := []struct {
		name     string
		query    string
		scope    string
		expected *tempopb.SearchTagsV2Response
	}{
		{
			name:  "no filtering",
			query: "",
			scope: "none",
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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
			expected: &tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
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

	for _, tc := range tcs {
		// Expected will not have the intrinsic results to make the tests simpler,
		// they are added here based on the scope.
		if tc.scope == "" || tc.scope == "none" || tc.scope == "intrinsic" {
			tc.expected.Scopes = append(tc.expected.Scopes, &tempopb.SearchTagsV2Scope{
				Name: "intrinsic",
				Tags: search.GetVirtualIntrinsicValues(),
			})
		}
		sort.Slice(tc.expected.Scopes, func(i, j int) bool { return tc.expected.Scopes[i].Name < tc.expected.Scopes[j].Name })
		for _, scope := range tc.expected.Scopes {
			slices.Sort(scope.Tags)
		}
	}

	return tcs
}

func buildSearchTagValuesV2TestCases(batches []batchTmpl) []struct {
	name     string
	query    string
	tagName  string
	expected *tempopb.SearchTagValuesV2Response
} {
	return []struct {
		name     string
		query    string
		tagName  string
		expected *tempopb.SearchTagValuesV2Response
	}{
		{
			name:    "no filtering",
			query:   "",
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "first batch - name",
			query:   fmt.Sprintf(`{ name="%s" }`, batches[2].name),
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].spanAttVal}},
			},
		},
		{
			name:    "second batch with incomplete query - name",
			query:   fmt.Sprintf(`{ name="%s" && span.twoSpan = }`, batches[3].name),
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "first batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[2].resourceAttVal),
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].spanAttVal}},
			},
		},
		{
			name:    "second batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[3].resourceAttVal),
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:     "too restrictive query",
			query:    fmt.Sprintf(`{ %s="%s" && resource.y="%s" }`, "resource.twoRes", batches[2].resourceAttVal, batches[3].resourceAttVal),
			tagName:  "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{},
		},
		// Unscoped not supported, unfiltered results.
		{
			name:    "unscoped span attribute",
			query:   fmt.Sprintf(`{ .twoSpan="%s" }`, batches[2].spanAttVal),
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
		{
			name:    "unscoped resource attribute",
			query:   fmt.Sprintf(`{ .twoRes="%s" }`, batches[2].spanAttVal),
			tagName: "resource.twoRes",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].resourceAttVal}, {Type: "string", Value: batches[3].resourceAttVal}},
			},
		},
		{
			name:    "first batch - name and resource attribute",
			query:   fmt.Sprintf(`{ name="%s" }`, batches[2].name),
			tagName: "resource.service.name",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "both batches - name and resource attribute",
			query:   `{ resource.service.name="my-service"}`,
			tagName: "name",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[3].name}, {Type: "string", Value: batches[2].name}},
			},
		},
		{
			name:    "only resource attributes",
			query:   fmt.Sprintf(`{ %s="%s" }`, "resource.twoRes", batches[2].resourceAttVal),
			tagName: "resource.service.name",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "bad query - unfiltered results",
			query:   fmt.Sprintf("%s = bar", "span.twoSpan"), // bad query, missing quotes
			tagName: "span.twoSpan",
			expected: &tempopb.SearchTagValuesV2Response{
				TagValues: []*tempopb.TagValue{{Type: "string", Value: batches[2].spanAttVal}, {Type: "string", Value: batches[3].spanAttVal}},
			},
		},
	}
}

func TestTraceByIDandTraceQL(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		Components: util.ComponentsRecentDataQuerying | util.ComponentsBackendQuerying,
		Backends:   util.BackendObjectStorageAll, // runs basic querying against all 3 object storage backends. no need to replicate for every test.
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		countTraces := 10
		infos := make([]*tempoUtil.TraceInfo, 0, countTraces)

		for range countTraces {
			time.Sleep(time.Millisecond) // force a new seed
			info := tempoUtil.NewTraceInfo(time.Now(), "")
			infos = append(infos, info)
			require.NoError(t, h.WriteTraceInfo(info, ""))
		}

		h.WaitTracesQueryable(t, countTraces)

		grpcClient, ctx, err := h.APIClientGRPC("")
		require.NoError(t, err)
		apiClient := h.APIClientHTTP("")

		now := time.Now()
		for _, i := range infos {
			util.QueryAndAssertTrace(t, apiClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, apiClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
			util.SearchStreamAndAssertTrace(t, ctx, grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}

		h.WaitTracesWrittenToBackend(t, countTraces)
		h.ForceBackendQuerying(t)

		grpcClient, ctx, err = h.APIClientGRPC("")
		require.NoError(t, err)
		apiClient = h.APIClientHTTP("")

		// Assert tags on storage backend
		for _, i := range infos {
			util.QueryAndAssertTrace(t, apiClient, i)
			util.SearchTraceQLAndAssertTraceWithRange(t, apiClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
			util.SearchStreamAndAssertTrace(t, ctx, grpcClient, i, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
		}
	})
}

func TestStreamingSearch_badRequest(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// Send a batch of traces
		batch := util.MakeThriftBatch()
		require.NoError(t, h.WriteJaegerBatch(batch, "")) // jpe - failed here with? rpc error: code = DeadlineExceeded desc = context deadline exceeded? should i increase a timeout somewhere?

		// Wait for the traces to be written to the WAL
		h.WaitTracesQueryable(t, 1)

		// Create gRPC client
		grpcClient, ctx, err := h.APIClientGRPC("")
		require.NoError(t, err)

		// Send invalid search query (missing operator)
		res, err := grpcClient.Search(ctx, &tempopb.SearchRequest{
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
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		// Test HTTP endpoint returns 400 for invalid tagName
		invalidTagName := "app.user.id" // not a valid scoped attribute
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v2/search/tag/%s/values", h.BaseURL(), invalidTagName), nil)
		require.NoError(t, err)

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Contains(t, string(body), "tag name is not valid intrinsic or scoped attribute")

		// Test gRPC endpoint returns InvalidArgument for invalid tagName
		grpcClient, ctx, err := h.APIClientGRPC("")
		require.NoError(t, err)

		stream, err := grpcClient.SearchTagValuesV2(ctx, &tempopb.SearchTagValuesRequest{
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

func callSearchTagValuesV2AndAssert(t *testing.T, h *util.TempoHarness, tagName, query string, expected *tempopb.SearchTagValuesV2Response, start, end int64) {
	apiClient := h.APIClientHTTP("")
	response, err := apiClient.SearchTagValuesV2WithRange(tagName, query, start, end)
	require.NoError(t, err)

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

	grpcClient, ctx, err := h.APIClientGRPC("")
	require.NoError(t, err)

	respTagsValuesV2, err := grpcClient.SearchTagValuesV2(ctx, grpcReq)
	require.NoError(t, err)
	finalResponse := &tempopb.SearchTagValuesV2Response{
		Metrics: &tempopb.MetadataMetrics{},
	}
	for {
		resp, err := respTagsValuesV2.Recv()
		if resp != nil {
			naiveTagValuesV2Combine(resp, finalResponse)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NotNil(t, finalResponse)
	sort.Slice(finalResponse.TagValues, func(i, j int) bool { return finalResponse.TagValues[i].Value < finalResponse.TagValues[j].Value })
	require.Equal(t, expected.TagValues, finalResponse.TagValues)
	// assert metrics, and make sure it's non-zero when response is non-empty
	if len(finalResponse.TagValues) > 0 {
		require.Greater(t, finalResponse.Metrics.InspectedBytes, uint64(0))
	}
}

func naiveTagValuesV2Combine(rNew, rInto *tempopb.SearchTagValuesV2Response) {
	rIntoTagValues := map[string]*tempopb.TagValue{}
	for _, val := range rInto.GetTagValues() {
		rIntoTagValues[val.Type+"="+val.Value] = val
	}

	for _, newVal := range rNew.GetTagValues() {
		if _, ok := rIntoTagValues[newVal.Type+"="+newVal.Value]; ok {
			continue
		}

		rInto.TagValues = append(rInto.TagValues, newVal)
	}

	rInto.Metrics.InspectedBytes += rNew.Metrics.InspectedBytes
	rInto.Metrics.CompletedJobs += rNew.Metrics.CompletedJobs
}

func callSearchTagsV2AndAssert(t *testing.T, h *util.TempoHarness, scope, query string, expected *tempopb.SearchTagsV2Response, start, end int64) {
	// search for tag values
	apiClient := h.APIClientHTTP("")

	response, err := apiClient.SearchTagsV2WithRange(scope, query, start, end)
	require.NoError(t, err)

	// parse response
	prepTagsResponse(response)
	require.Equal(t, expected.Scopes, response.Scopes)
	assertMetrics(t, response.Metrics, lenWithoutIntrinsic(response))

	// streaming
	grpcReq := &tempopb.SearchTagsRequest{
		Scope: scope,
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	}

	grpcClient, ctx, err := h.APIClientGRPC("")
	require.NoError(t, err)

	respTagsValuesV2, err := grpcClient.SearchTagsV2(ctx, grpcReq)
	require.NoError(t, err)
	finalResponse := &tempopb.SearchTagsV2Response{
		Metrics: &tempopb.MetadataMetrics{},
	}
	for {
		resp, err := respTagsValuesV2.Recv()
		if resp != nil {
			naiveTagsV2Combine(resp, finalResponse)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NotNil(t, finalResponse)
	require.NotNil(t, finalResponse.Metrics)

	prepTagsResponse(finalResponse)
	require.Equal(t, expected.Scopes, finalResponse.Scopes)
	// assert metrics, and make sure it's non-zero when response is non-empty
	if lenWithoutIntrinsic(response) > 0 {
		require.Greater(t, finalResponse.Metrics.InspectedBytes, uint64(100))
	}
}

func naiveTagsV2Combine(rNew, rInto *tempopb.SearchTagsV2Response) {
	distinctVals := collector.NewScopedDistinctString(0, 0, 0)
	for _, scope := range rNew.GetScopes() {
		for _, tag := range scope.GetTags() {
			distinctVals.Collect(scope.GetName(), tag)
		}
	}

	for _, scope := range rInto.GetScopes() {
		for _, tag := range scope.GetTags() {
			distinctVals.Collect(scope.GetName(), tag)
		}
	}

	scopeToVals := distinctVals.Strings()
	rInto.Scopes = make([]*tempopb.SearchTagsV2Scope, 0, len(scopeToVals))
	for scope, vals := range scopeToVals {
		rInto.Scopes = append(rInto.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scope,
			Tags: vals,
		})
	}

	rInto.Metrics.InspectedBytes += rNew.Metrics.InspectedBytes
	rInto.Metrics.CompletedJobs += rNew.Metrics.CompletedJobs
}

func prepTagsResponse(resp *tempopb.SearchTagsV2Response) {
	sort.Slice(resp.Scopes, func(i, j int) bool { return resp.Scopes[i].Name < resp.Scopes[j].Name })
	for _, scope := range resp.Scopes {
		if len(scope.Tags) == 0 {
			scope.Tags = nil
		}

		slices.Sort(scope.Tags)
	}
}

func assertMetrics(t *testing.T, metrics *tempopb.MetadataMetrics, respLen int) {
	// metrics are not present when response is empty, so return
	if respLen == 0 {
		return
	}

	require.NotNil(t, metrics)
	// if response len is empty, then the inspected bytes should be 0
	// assert metrics, and make sure it's non-zero
	require.Greater(t, metrics.InspectedBytes, uint64(300))
}

func lenWithoutIntrinsic(resp *tempopb.SearchTagsV2Response) int {
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
