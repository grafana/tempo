package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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
		expected searchTagValuesResponse
	}{
		{
			name:    "no filtering",
			query:   "",
			tagName: spanX,
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}, {Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:    "first batch - name",
			query:   fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
			tagName: spanX,
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}},
			},
		},
		{
			name:    "second batch with incomplete query - name",
			query:   fmt.Sprintf(`{ name="%s" && span.x = }`, secondBatch.name),
			tagName: spanX,
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:    "first batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, firstBatch.resourceAttVal),
			tagName: spanX,
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: firstBatch.spanAttVal}},
			},
		},
		{
			name:    "second batch only - resource attribute",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, secondBatch.resourceAttVal),
			tagName: spanX,
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: secondBatch.spanAttVal}},
			},
		},
		{
			name:     "too restrictive query",
			query:    fmt.Sprintf(`{ %s="%s" && resource.y="%s" }`, resourceX, firstBatch.resourceAttVal, secondBatch.resourceAttVal),
			tagName:  spanX,
			expected: searchTagValuesResponse{},
		},
		{
			name:     "unscoped attribute", // TODO: Not supported, should return only the first batch
			query:    fmt.Sprintf(`{ .x="%s" }`, firstBatch.spanAttVal),
			tagName:  spanX,
			expected: searchTagValuesResponse{},
		},
		{
			name:    "first batch - name and resource attribute",
			query:   fmt.Sprintf(`{ name="%s" }`, firstBatch.name),
			tagName: "resource.service.name",
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: "my-service"}},
			},
		},
		{
			name:    "both batches - name and resource attribute",
			query:   `{ resource.service.name="my-service"}`,
			tagName: "name",
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: secondBatch.name}, {Type: "string", Value: firstBatch.name}},
			},
		},
		{
			name:    "only resource attributes",
			query:   fmt.Sprintf(`{ %s="%s" }`, resourceX, firstBatch.resourceAttVal),
			tagName: "resource.service.name",
			expected: searchTagValuesResponse{
				TagValues: []TagValue{{Type: "string", Value: "my-service"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callSearchTagValuesAndAssert(t, tempo, tc.tagName, tc.query, tc.expected)
		})
	}
}

func callSearchTagValuesAndAssert(t *testing.T, svc *e2e.HTTPService, tagName, query string, expected searchTagValuesResponse) {
	urlPath := fmt.Sprintf(`/api/v2/search/tag/%s/values?q=%s`, tagName, url.QueryEscape(query))

	// search for tag values
	req, err := http.NewRequest(http.MethodGet, "http://"+svc.Endpoint(3200)+urlPath, nil)
	require.NoError(t, err)

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
	sort.Slice(response.TagValues, func(i, j int) bool { return response.TagValues[i].Value < response.TagValues[j].Value })
	require.Equal(t, expected, response)
}

type searchTagValuesResponse struct {
	TagValues []TagValue `json:"tagValues"`
}

type TagValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
