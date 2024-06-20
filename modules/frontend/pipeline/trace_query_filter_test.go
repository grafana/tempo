package pipeline

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTraceQueryFilterWareMetrics(t *testing.T) {

}

func TestTraceQueryFilterWareSearch(t *testing.T) {

}

func TestTraceQueryFilterWare(t *testing.T) {

	tests := []struct {
		name         string
		query        string
		denyList     []string
		expectedResp *http.Response
	}{
		{
			name:  "no query",
			query: "",
			denyList: []string{
				"GET",
				"POST",
			},
			expectedResp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
		},
		{
			name:  "query matches regex",
			query: "span.http.method='GET'",
			denyList: []string{
				"GET",
			},
			expectedResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader("Query is temporarily blocked by your administrator.")),
			},
		},
		{
			name:  "query does not match regex",
			query: "span.http.method='GET'",
			denyList: []string{
				"status",
				"start",
			},
			expectedResp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
		},
		{
			name:     "empty deny list",
			query:    "service.name=cart",
			denyList: []string{},
			expectedResp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader("foo")),
			},
		},
		{
			name:  "query matches multiple patterns",
			query: "span.http.method='GET'&&span.http.status_code>=200",
			denyList: []string{
				"GET",
				"span",
				"200",
			},
			expectedResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     http.StatusText(http.StatusBadRequest),
				Body:       io.NopCloser(strings.NewReader("Query is temporarily blocked by your administrator.")),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := buildSearchTagValuesQueryUrl("service.name", tc.query)
			req := httptest.NewRequest("GET", u, nil)
			test := NewTraceQueryFilterWareWithDenyList(tc.denyList, ParseSearchRequestQuery)
			next := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Status:     http.StatusText(http.StatusOK),
					Body:       io.NopCloser(strings.NewReader("foo")),
				}
				return resp, nil
			})

			rt, err := test.Wrap(next).RoundTrip(req)

			require.NoError(t, err)

			require.Equal(t, tc.expectedResp, rt)

		})

	}
}

func TestTraceQueryFilterWareNoDenyList(t *testing.T) {

}

func buildSearchTagsQueryUrl(query string) string {
	joinURL, _ := url.Parse("http://localhost:3200/api/v2/search/tags")
	q := joinURL.Query()
	q.Set("q", query)
	joinURL.RawQuery = q.Encode()
	return fmt.Sprint(joinURL)
}

func buildSearchTagValuesQueryUrl(key string, query string) string {
	urlPath := fmt.Sprintf("/api/v2/search/tag/%s/values", key)
	joinURL, _ := url.Parse("http://localhost:3200" + urlPath + "?")
	q := joinURL.Query()
	q.Set("q", query)
	joinURL.RawQuery = q.Encode()
	return fmt.Sprint(joinURL)
}

func buildMetricQueryUrl(query string) string {
	urlPath := "/api/metrics/query_range"
	joinURL, _ := url.Parse("http://localhost:3200" + urlPath + "?")
	q := joinURL.Query()
	q.Set("q", query)
	joinURL.RawQuery = q.Encode()
	return fmt.Sprint(joinURL)
}
