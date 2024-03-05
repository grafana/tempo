package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/e2e"
	util "github.com/grafana/tempo/integration"
	"github.com/stretchr/testify/require"
)

const (
	spanX     = "span.x"
	resourceX = "resource.xx"
)

func TestSearchTagValuesV2(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configAllInOneLocal, "config.yaml"))
	tempo := util.NewTempoAllInOne("-autocomplete-filtering.enabled=true")
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

	batch := makeThriftBatchWithSpanCountAttributeAndName(firstBatch.spanCount, firstBatch.name, firstBatch.resourceAttVal, firstBatch.spanAttVal)
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountAttributeAndName(secondBatch.spanCount, secondBatch.name, secondBatch.resourceAttVal, secondBatch.spanAttVal)
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
	callSearchTagsAndAssert(t, tempo, searchTagsResponse{}, 0, 0)

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
	callSearchTagValuesAndAssert(t, tempo, "service.name", searchTagValuesResponse{}, 0, 0)
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
