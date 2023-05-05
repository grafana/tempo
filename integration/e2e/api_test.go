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

	batch := makeThriftBatchWithSpanCountAttributeAndName(2, "foo", "bar")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	batch = makeThriftBatchWithSpanCountAttributeAndName(2, "baz", "qux")
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), batch))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	// Search for tag values without filters
	expected := searchTagValuesResponse{
		TagValues: []TagValue{{Type: "string", Value: "bar"}, {Type: "string", Value: "qux"}},
	}
	callSearchTagValuesAndAssert(t, tempo, "span.x", "", expected)

	// Search for tag values for the first batch only
	expected = searchTagValuesResponse{TagValues: []TagValue{{Type: "string", Value: "bar"}}}
	callSearchTagValuesAndAssert(t, tempo, "span.x", `{ name="foo" }`, expected)

	// Search for tag values for the second batch only with incomplete query
	expected = searchTagValuesResponse{TagValues: []TagValue{{Type: "string", Value: "qux"}}}
	callSearchTagValuesAndAssert(t, tempo, "span.x", `{ name="baz" && span.x = }`, expected)
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
