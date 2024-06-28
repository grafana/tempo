package e2e

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
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

const (
	spanX     = "span.x"
	resourceX = "resource.xx"
)

func TestSearchTagsV2(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	type batchTmpl struct {
		spanCount                  int
		name                       string
		resourceAttVal, spanAttVal string
		resourceName, spanName     string
	}

	firstBatch := batchTmpl{spanCount: 2, name: "foo", resourceAttVal: "bar", spanAttVal: "bar", resourceName: "firstRes", spanName: "firstSpan"}
	secondBatch := batchTmpl{spanCount: 2, name: "baz", resourceAttVal: "qux", spanAttVal: "qux", resourceName: "secondRes", spanName: "secondSpan"}

	batch := makeThriftBatchWithSpanCountAttributeAndName(firstBatch.spanCount, firstBatch.name, firstBatch.resourceAttVal, firstBatch.spanAttVal, firstBatch.resourceName, firstBatch.spanName)
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountAttributeAndName(secondBatch.spanCount, secondBatch.name, secondBatch.resourceAttVal, secondBatch.spanAttVal, secondBatch.resourceName, secondBatch.spanName)
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	testCases := []struct {
		name     string
		query    string
		scope    string
		expected tempopb.SearchTagsV2Response
	}{
		{
			name:  "no filtering",
			query: "",
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName, secondBatch.spanName},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, secondBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "first batch - resource",
			query: fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
			scope: "resource",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch with incomplete query - span",
			query: fmt.Sprintf(`{ name="%s" && span.x = }`, secondBatch.name),
			scope: "span",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{secondBatch.spanName},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, firstBatch.resourceName, firstBatch.resourceAttVal),
			scope: "span",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName},
					},
				},
			},
		},
		{
			name:  "first batch - resource att - resource",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, firstBatch.resourceName, firstBatch.resourceAttVal),
			scope: "resource",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "second batch - resource attribute - span",
			query: fmt.Sprintf(`{ resource.%s="%s" }`, secondBatch.resourceName, secondBatch.resourceAttVal),
			scope: "span",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{secondBatch.spanName},
					},
				},
			},
		},
		{
			name:  "too restrictive query",
			query: fmt.Sprintf(`{ resource.%s="%s" && resource.y="%s" }`, firstBatch.resourceName, firstBatch.resourceAttVal, secondBatch.resourceAttVal),
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
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
			query: fmt.Sprintf(`{ .x="%s" }`, firstBatch.spanAttVal),
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName, secondBatch.spanName},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, secondBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "unscoped res attribute",
			query: fmt.Sprintf(`{ .xx="%s" }`, firstBatch.resourceAttVal),
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName, secondBatch.spanName},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, secondBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "both batches - name and resource attribute",
			query: `{ resource.service.name="my-service"}`,
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName, secondBatch.spanName},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, secondBatch.resourceName, "service.name"},
					},
				},
			},
		},
		{
			name:  "bad query - unfiltered results",
			query: fmt.Sprintf("%s = bar", spanX), // bad query, missing quotes
			scope: "none",
			expected: tempopb.SearchTagsV2Response{
				Scopes: []*tempopb.SearchTagsV2Scope{
					{
						Name: "span",
						Tags: []string{firstBatch.spanName, secondBatch.spanName},
					},
					{
						Name: "resource",
						Tags: []string{firstBatch.resourceName, secondBatch.resourceName, "service.name"},
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

	// Wait to block flushed to backend, 20 seconds is the complete_block_timeout configuration on all in one, we add
	// 2s for security.
	callFlush(t, tempo)
	time.Sleep(time.Second * 22)
	callFlush(t, tempo)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_cleared_total"))

	// Assert no more on the ingester
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagsV2AndAssert(t, tempo, tc.scope, tc.query, tempopb.SearchTagsV2Response{}, 0, 0)
		})
	}

	// Wait to blocklist_poll to be completed
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))

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

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	type batchTmpl struct {
		spanCount                  int
		name                       string
		resourceAttVal, spanAttVal string
	}

	firstBatch := batchTmpl{spanCount: 2, name: "foo", resourceAttVal: "bar", spanAttVal: "bar"}
	secondBatch := batchTmpl{spanCount: 2, name: "baz", resourceAttVal: "qux", spanAttVal: "qux"}

	batch := makeThriftBatchWithSpanCountAttributeAndName(firstBatch.spanCount, firstBatch.name, firstBatch.resourceAttVal, firstBatch.spanAttVal, "xx", "x")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountAttributeAndName(secondBatch.spanCount, secondBatch.name, secondBatch.resourceAttVal, secondBatch.spanAttVal, "xx", "x")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

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

	// Wait to block flushed to backend, 20 seconds is the complete_block_timeout configuration on all in one, we add
	// 2s for security.
	callFlush(t, tempo)
	time.Sleep(time.Second * 22)
	callFlush(t, tempo)

	// test metrics
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_flushed_total"))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetrics(e2e.Equals(1), "tempo_ingester_blocks_cleared_total"))

	// Assert no more on the ingester
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagValuesV2AndAssert(t, tempo, tc.tagName, tc.query, searchTagValuesV2Response{TagValues: []TagValue{}}, 0, 0)
		})
	}

	// Wait to blocklist_poll to be completed
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.Equals(1), []string{"tempodb_blocklist_length"}, e2e.WaitMissingMetrics))

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

// todo: add search tags v2
func TestSearchTags(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_tags")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := makeThriftBatch()
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)
	callSearchTagsAndAssert(t, tempo, searchTagsResponse{TagNames: []string{"service.name", "x", "xx"}}, 0, 0)

	callFlush(t, tempo)
	time.Sleep(time.Second * 30)
	callFlush(t, tempo)

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

	jaegerClient, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	batch := makeThriftBatch()
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)
	callSearchTagValuesAndAssert(t, tempo, "service.name", searchTagValuesResponse{TagValues: []string{"my-service"}}, 0, 0)

	callFlush(t, tempo)
	time.Sleep(time.Second * 22)
	callFlush(t, tempo)

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

func callSearchTagValuesV2AndAssert(t *testing.T, svc *e2e.HTTPService, tagName, query string, expected searchTagValuesV2Response, start, end int64) {
	urlPath := fmt.Sprintf(`/api/v2/search/tag/%s/values?q=%s`, tagName, url.QueryEscape(query))

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(3200)+urlPath, nil)
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
	require.Equal(t, expected, response)

	// streaming
	grpcReq := &tempopb.SearchTagValuesRequest{
		TagName: tagName,
		Query:   query,
		Start:   uint32(start),
		End:     uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(3200))
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
	require.Equal(t, expected, actualGrpcResp)
}

func callSearchTagsV2AndAssert(t *testing.T, svc *e2e.HTTPService, scope, query string, expected tempopb.SearchTagsV2Response, start, end int64) {
	urlPath := fmt.Sprintf(`/api/v2/search/tags?scope=%s&q=%s`, scope, url.QueryEscape(query))

	// expected will not have the intrinsic scope since it's the same every time, add it here.
	if scope == "none" || scope == "" || scope == "intrinsic" {
		expected.Scopes = append(expected.Scopes, &tempopb.SearchTagsV2Scope{
			Name: "intrinsic",
			Tags: []string{"duration", "event:name", "kind", "name", "rootName", "rootServiceName", "span:duration", "span:kind", "span:name", "span:status", "span:statusMessage", "status", "statusMessage", "trace:duration", "trace:rootName", "trace:rootService", "traceDuration"},
		})
	}
	sort.Slice(expected.Scopes, func(i, j int) bool { return expected.Scopes[i].Name < expected.Scopes[j].Name })
	for _, scope := range expected.Scopes {
		slices.Sort(scope.Tags)
	}

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(3200)+urlPath, nil)
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
	var response tempopb.SearchTagsV2Response
	require.NoError(t, json.Unmarshal(body, &response))

	prepTagsResponse(&response)
	require.Equal(t, expected, response)

	// streaming
	grpcReq := &tempopb.SearchTagsRequest{
		Scope: scope,
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(3200))
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

	prepTagsResponse(&response)
	require.Equal(t, expected, response)
}

func prepTagsResponse(resp *tempopb.SearchTagsV2Response) {
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
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(3200)+urlPath, nil)
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
	sort.Strings(response.TagNames)
	sort.Strings(expected.TagNames)
	require.Equal(t, expected, response)

	// streaming
	grpcReq := &tempopb.SearchTagsRequest{
		Start: uint32(start),
		End:   uint32(end),
	}

	grpcClient, err := util.NewSearchGRPCClient(context.Background(), svc.Endpoint(3200))
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
	sort.Slice(grpcResp.TagNames, func(i, j int) bool { return grpcResp.TagNames[i] < grpcResp.TagNames[j] })
	require.Equal(t, expected.TagNames, grpcResp.TagNames)
}

func callSearchTagValuesAndAssert(t *testing.T, svc *e2e.HTTPService, tagName string, expected searchTagValuesResponse, start, end int64) {
	urlPath := fmt.Sprintf(`/api/search/tag/%s/values`, tagName)
	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(3200)+urlPath, nil)
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

	require.Equal(t, expected, response)
}

type searchTagValuesV2Response struct {
	TagValues []TagValue `json:"tagValues"`
}

type searchTagValuesResponse struct {
	TagValues []string `json:"tagValues"`
}

type searchTagsResponse struct {
	TagNames []string `json:"tagNames"`
}

type TagValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
