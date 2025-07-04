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
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	configAllInOneLocal = "../deployments/config-all-in-one-local.yaml"
	spanX               = "span.x"
	resourceX           = "resource.xx"

	tempoPort = 3200

	queryableTimeout    = 5 * time.Second        // timeout for waiting for traces to be queryable
	queryableCheckEvery = 100 * time.Millisecond // check every 100ms for traces to be queryable

	// Wait to block flushed to backend, 5 seconds is the complete_block_timeout configuration on all in one, we add
	// 1s for security.
	blockFlushTimeout = 6 * time.Second
)

func TestSearchTagsV2(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	type batchTmpl struct {
		spanCount                  int
		name                       string
		resourceAttVal, spanAttVal string
		resourceAttr, SpanAttr     string
	}

	firstBatch := batchTmpl{spanCount: 2, name: "foo", resourceAttVal: "bar", spanAttVal: "bar", resourceAttr: "firstRes", SpanAttr: "firstSpan"}
	secondBatch := batchTmpl{spanCount: 2, name: "baz", resourceAttVal: "qux", spanAttVal: "qux", resourceAttr: "secondRes", SpanAttr: "secondSpan"}

	batch := util.MakeThriftBatchWithSpanCountAttributeAndName(firstBatch.spanCount, firstBatch.name, firstBatch.resourceAttVal, firstBatch.spanAttVal, firstBatch.resourceAttr, firstBatch.SpanAttr)
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = util.MakeThriftBatchWithSpanCountAttributeAndName(secondBatch.spanCount, secondBatch.name, secondBatch.resourceAttVal, secondBatch.spanAttVal, secondBatch.resourceAttr, secondBatch.SpanAttr)
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL and searchable
	require.Eventually(t, func() bool {
		ok, err := isQueryable(tempo.Endpoint(tempoPort))
		if err != nil {
			return false
		}
		return ok
	}, queryableTimeout, queryableCheckEvery, "traces were not queryable within timeout")

	// wait for the 2 traces to be written to the WAL
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(2), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	testCases := []struct {
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
						Tags: []string{firstBatch.SpanAttr, secondBatch.SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, secondBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "first batch - resource",
			query: fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
			scope: "resource",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch with incomplete query - span",
			query: fmt.Sprintf(`{ name="%s" && span.x = }`, secondBatch.name),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{secondBatch.SpanAttr},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, firstBatch.resourceAttr, firstBatch.resourceAttVal),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{firstBatch.SpanAttr},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - resource",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, firstBatch.resourceAttr, firstBatch.resourceAttVal),
			scope: "resource",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch - resource attribute - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, secondBatch.resourceAttr, secondBatch.resourceAttVal),
			scope: "span",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{secondBatch.SpanAttr},
					},
				},
			},
		},
		{
			name:  "too restrictive query",
			query: fmt.Sprintf(`{ resource.%s="%s" && resource.y="%s" }`, firstBatch.resourceAttr, firstBatch.resourceAttVal, secondBatch.resourceAttVal),
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
			query: fmt.Sprintf(`{ .x="%s" }`, firstBatch.spanAttVal),
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{firstBatch.SpanAttr, secondBatch.SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, secondBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "unscoped res attribute",
			query: fmt.Sprintf(`{ .xx="%s" }`, firstBatch.resourceAttVal),
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{firstBatch.SpanAttr, secondBatch.SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, secondBatch.resourceAttr, "service.name"},
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
						Tags: []string{firstBatch.SpanAttr, secondBatch.SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, secondBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
		{
			name:  "bad query - unfiltered results",
			query: fmt.Sprintf("%s = bar", spanX), // bad query, missing quotes
			scope: "none",
			expected: searchTagsV2Response{
				Scopes: []ScopedTags{
					{
						Name: "span",
						Tags: []string{firstBatch.SpanAttr, secondBatch.SpanAttr},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceAttr, secondBatch.resourceAttr, "service.name"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagsV2AndAssert(t, tempo, tc.scope, tc.query, tc.expected, 0, 0)
		})
	}

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	// wait for 2 objects to be written to the backend
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(2), []string{"tempodb_backend_objects_total"}, e2e.WaitMissingMetrics))

	// Assert tags on storage backend
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(2 * time.Hour)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagsV2AndAssert(t, tempo, tc.scope, tc.query, tc.expected, start.Unix(), end.Unix())
		})
	}
}

func TestSearchTagValuesV2(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	type batchTmpl struct {
		spanCount                  int
		name                       string
		resourceAttVal, spanAttVal string
	}

	firstBatch := batchTmpl{spanCount: 2, name: "foo", resourceAttVal: "bar", spanAttVal: "bar"}
	secondBatch := batchTmpl{spanCount: 2, name: "baz", resourceAttVal: "qux", spanAttVal: "qux"}

	batch := util.MakeThriftBatchWithSpanCountAttributeAndName(firstBatch.spanCount, firstBatch.name, firstBatch.resourceAttVal, firstBatch.spanAttVal, "xx", "x")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = util.MakeThriftBatchWithSpanCountAttributeAndName(secondBatch.spanCount, secondBatch.name, secondBatch.resourceAttVal, secondBatch.spanAttVal, "xx", "x")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL and searchable
	require.Eventually(t, func() bool {
		ok, err := isQueryable(tempo.Endpoint(tempoPort))
		if err != nil {
			return false
		}
		return ok
	}, queryableTimeout, queryableCheckEvery, "traces were not queryable within timeout")

	// wait for the 2 traces to be written to the WAL
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(2), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

	testCases := []struct {
		name     string
		query    string
		tagName  string
		expected searchTagValuesV2Response
	}{
		{
			name:    "no filtering",
			query:   "",
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}, {Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:    "first batch - name",
			query:   fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}},
			},
		},
		{
			name:    "second batch with incomplete query - name",
			query:   fmt.Sprintf(`{ name="%s" && span.x = }`, secondBatch.name),
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:    "first batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, firstBatch.resourceAttVal),
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}},
			},
		},
		{
			name:    "second batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, secondBatch.resourceAttVal),
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:     "too restrictive query",
			query:    fmt.Sprintf(`{ %s="%s" && resource.y="%s" }`, resourceX, firstBatch.resourceAttVal, secondBatch.resourceAttVal),
			tagName:  spanX,
			expected: searchTagValuesV2Response{TagValues: []TagValue{}},
		},
		// Unscoped not supported, unfiltered results.
		{
			name:    "unscoped span attribute",
			query:   fmt.Sprintf(`{ .x="%s" }`, firstBatch.spanAttVal),
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}, {Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:    "unscoped resource attribute",
			query:   fmt.Sprintf(`{ .xx="%s" }`, firstBatch.spanAttVal),
			tagName: resourceX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.resourceAttVal}, {Type: "string", Value: secondBatch.resourceAttVal}},
			},
		},
		{
			name:    "first batch - name and resource attribute",
			query:   fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
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
				TagValues: []TagValue{{Type: "string", Value: secondBatch.name}, {Type: "string", Value: firstBatch.name}},
			},
		},
		{
			name:    "only resource attributes",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, firstBatch.resourceAttVal),
			tagName: "resource.service.name",
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "bad query - unfiltered results",
			query:   fmt.Sprintf("%s = bar", spanX), // bad query, missing quotes
			tagName: spanX,
			expected: searchTagValuesV2Response{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}, {Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagValuesV2AndAssert(t, tempo, tc.tagName, tc.query, tc.expected, 0, 0)
		})
	}

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	// wait for 2 objects to be written to the backend
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(2), []string{"tempodb_backend_objects_total"}, e2e.WaitMissingMetrics))

	// Assert tags on storage backend
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(2 * time.Hour)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagValuesV2AndAssert(t, tempo, tc.tagName, tc.query, tc.expected, start.Unix(), end.Unix())
		})
	}
}

func TestSearchTags(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_tags")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := util.MakeThriftBatch()
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)
	callSearchTagsAndAssert(t, tempo, searchTagsResponse{TagNames: []string{"service.name", "x", "xx"}}, 0, 0)

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_cleared_total"))

	// Assert no more on the ingester
	callSearchTagsAndAssert(t, tempo, searchTagsResponse{TagNames: []string{}}, 0, 0)

	// Wait to blocklist_poll to be completed
	time.Sleep(time.Second * 2)
	// Assert tags on storage backend
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(2 * time.Hour)
	callSearchTagsAndAssert(t, tempo, searchTagsResponse{TagNames: []string{"service.name", "x", "xx"}}, start.Unix(), end.Unix())
}

func TestSearchTagValues(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_tags")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := util.MakeThriftBatch()
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)
	callSearchTagValuesAndAssert(t, tempo, "service.name", searchTagValuesResponse{TagValues: []string{"my-service"}}, 0, 0)

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_cleared_total"))

	// Assert no more on the ingester
	callSearchTagValuesAndAssert(t, tempo, "service.name", searchTagValuesResponse{TagValues: []string{}}, 0, 0)
	// Wait to blocklist_poll to be completed
	time.Sleep(time.Second * 2)
	// Assert tags on storage backen
	now := time.Now()
	start := now.Add(-2 * time.Hour)
	end := now.Add(2 * time.Hour)
	callSearchTagValuesAndAssert(t, tempo, "service.name", searchTagValuesResponse{TagValues: []string{"my-service"}}, start.Unix(), end.Unix())
}

func TestStreamingSearch_badRequest(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_tags")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := util.MakeThriftBatch()
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	// Create gRPC client
	c, err := util.NewSearchGRPCClient(context.Background(), tempo.Endpoint(tempoPort))
	require.NoError(t, err)

	res, err := c.Search(context.Background(), &tempopb.SearchRequest{
		Query: "{resource.service.name=article}",
	})
	require.NoError(t, err)

	_, err = res.Recv()
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
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
