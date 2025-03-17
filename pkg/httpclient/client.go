package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb" //nolint:all
	"github.com/golang/protobuf/proto"  //nolint:all
	"github.com/klauspost/compress/gzhttp"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/grafana/tempo/pkg/util"
)

const (
	orgIDHeader = "X-Scope-OrgID"

	QueryTraceEndpoint   = "/api/traces"
	QueryTraceV2Endpoint = "/api/v2/traces"

	acceptHeader        = "Accept"
	applicationProtobuf = "application/protobuf"
	applicationJSON     = "application/json"
)

type TempoHTTPClient interface {
	WithTransport(t http.RoundTripper)
	Do(req *http.Request) (*http.Response, error)
	SearchTags() (*tempopb.SearchTagsResponse, error)
	SearchTagsV2() (*tempopb.SearchTagsV2Response, error)
	SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error)
	SearchTagsV2WithRange(start int64, end int64) (*tempopb.SearchTagsV2Response, error)
	SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error)
	SearchTagValuesV2(key, query string) (*tempopb.SearchTagValuesV2Response, error)
	SearchTagValuesV2WithRange(tag string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error)
	Search(tags string) (*tempopb.SearchResponse, error)
	SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error)
	QueryTrace(id string) (*tempopb.Trace, error)
	QueryTraceWithRange(id string, start int64, end int64) (*tempopb.Trace, error)
	SearchTraceQL(query string) (*tempopb.SearchResponse, error)
	SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error)
	SearchTraceQLWithRangeAndLimit(query string, start int64, end int64, limit int64, spss int64) (*tempopb.SearchResponse, error)
	MetricsSummary(query string, groupBy string, start int64, end int64) (*tempopb.SpanMetricsSummaryResponse, error)
	MetricsQueryRange(query string, start, end int, step string, exemplars int) (*tempopb.QueryRangeResponse, error)
	GetOverrides() (*userconfigurableoverrides.Limits, string, error)
	SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error)
	PatchOverrides(limits *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error)
	DeleteOverrides(version string) error
}

var ErrNotFound = errors.New("resource not found")

// Client is client to the Tempo API.
type Client struct {
	BaseURL     string
	OrgID       string
	client      *http.Client
	headers     map[string]string
	queryParams map[string]string
}

func New(baseURL, orgID string) *Client {
	return &Client{
		BaseURL: baseURL,
		OrgID:   orgID,
		client:  http.DefaultClient,
	}
}

func NewWithCompression(baseURL, orgID string) *Client {
	c := New(baseURL, orgID)
	c.WithTransport(gzhttp.Transport(http.DefaultTransport))
	return c
}

func (c *Client) SetHeader(key string, value string) {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[key] = value
}

func (c *Client) SetQueryParam(key, value string) {
	if c.queryParams == nil {
		c.queryParams = make(map[string]string)
	}
	c.queryParams[key] = value
}

func (c *Client) WithTransport(t http.RoundTripper) {
	c.client.Transport = t
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// getFor sends a GET request and attempts to unmarshal the response.
func (c *Client) getFor(url string, m proto.Message) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	marshallingFormat := applicationJSON
	if strings.Contains(url, QueryTraceEndpoint) || strings.Contains(url, QueryTraceV2Endpoint) {
		marshallingFormat = applicationProtobuf
	}
	// Set 'Accept' header to 'application/protobuf'.
	// This is required for the /api/traces and /api/v2/traces endpoint to return a protobuf response.
	// JSON lost backwards compatibility with the upgrade to `opentelemetry-proto` v0.18.0.
	req.Header.Set(acceptHeader, marshallingFormat)

	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	switch marshallingFormat {
	case applicationJSON:
		if err = jsonpb.UnmarshalString(string(body), m); err != nil {
			return resp, fmt.Errorf("error decoding %T json, err: %v body: %s", m, err, string(body))
		}
	default:
		if err = proto.Unmarshal(body, m); err != nil {
			return resp, fmt.Errorf("error decoding %T proto, err: %w body: %s", m, err, string(body))
		}
	}

	return resp, nil
}

// doRequest sends the given request, it injects X-Scope-OrgID and handles bad status codes.
func (c *Client) doRequest(req *http.Request) (*http.Response, []byte, error) {
	if len(c.OrgID) > 0 {
		req.Header.Set(orgIDHeader, c.OrgID)
	}

	if c.headers != nil {
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error querying Tempo %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		body, _ := io.ReadAll(resp.Body)
		return resp, body, fmt.Errorf("%s request to %s failed with response: %d body: %s", req.Method, req.URL.String(), resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading response body: %w", err)
	}

	return resp, body, nil
}

func (c *Client) SearchTags() (*tempopb.SearchTagsResponse, error) {
	m := &tempopb.SearchTagsResponse{}
	_, err := c.getFor(c.BaseURL+api.PathSearchTags, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagsV2() (*tempopb.SearchTagsV2Response, error) {
	m := &tempopb.SearchTagsV2Response{}
	_, err := c.getFor(c.BaseURL+api.PathSearchTagsV2, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error) {
	m := &tempopb.SearchTagsResponse{}
	_, err := c.getFor(c.buildTagsQueryURL(start, end), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagsV2WithRange(start int64, end int64) (*tempopb.SearchTagsV2Response, error) {
	m := &tempopb.SearchTagsV2Response{}
	_, err := c.getFor(c.buildTagsV2QueryURL(start, end), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error) {
	m := &tempopb.SearchTagValuesResponse{}
	_, err := c.getFor(c.BaseURL+"/api/search/tag/"+key+"/values", m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagValuesV2(key, query string) (*tempopb.SearchTagValuesV2Response, error) {
	m := &tempopb.SearchTagValuesV2Response{}
	urlPath := fmt.Sprintf(`/api/v2/search/tag/%s/values?q=%s`, key, url.QueryEscape(query))

	_, err := c.getFor(c.BaseURL+urlPath, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTagValuesV2WithRange(tag string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error) {
	m := &tempopb.SearchTagValuesV2Response{}
	_, err := c.getFor(c.buildTagValuesV2QueryURL(tag, start, end), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Search Tempo. tags must be in logfmt format, that is "key1=value1 key2=value2"
func (c *Client) Search(tags string) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.buildSearchQueryURL("tags", tags, 0, 0, 0, 0, c.queryParams), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// SearchWithRange calls the /api/search endpoint. tags is expected to be in logfmt format and start/end are unix
// epoch timestamps in seconds.
func (c *Client) SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.buildSearchQueryURL("tags", tags, start, end, 0, 0, c.queryParams), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) QueryTrace(id string) (*tempopb.Trace, error) {
	m := &tempopb.Trace{}
	resp, err := c.getFor(c.BaseURL+QueryTraceEndpoint+"/"+id, m)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, util.ErrTraceNotFound
		}
		return nil, err
	}
	return m, nil
}

func (c *Client) QueryTraceV2(id string) (*tempopb.TraceByIDResponse, error) {
	m := &tempopb.TraceByIDResponse{}
	resp, err := c.getFor(c.BaseURL+QueryTraceV2Endpoint+"/"+id, m)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, util.ErrTraceNotFound
		}
		return nil, err
	}
	return m, nil
}

func (c *Client) QueryTraceWithRange(id string, start int64, end int64) (*tempopb.Trace, error) {
	m := &tempopb.Trace{}
	if start > end {
		return nil, errors.New("start time can not be greater than end time")
	}
	queryParams := mergeMaps(c.queryParams, map[string]string{
		"start": strconv.FormatInt(start, 10),
		"end":   strconv.FormatInt(end, 10),
	})
	url := c.getURLWithQueryParams(QueryTraceEndpoint+"/"+id, queryParams)
	resp, err := c.getFor(url, m)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, util.ErrTraceNotFound
		}
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTraceQL(query string) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.buildSearchQueryURL("q", query, 0, 0, 0, 0, c.queryParams), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.buildSearchQueryURL("q", query, start, end, 0, 0, c.queryParams), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) SearchTraceQLWithRangeAndLimit(query string, start int64, end int64, limit int64, spss int64) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.buildSearchQueryURL("q", query, start, end, limit, spss, c.queryParams), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) MetricsSummary(query string, groupBy string, start int64, end int64) (*tempopb.SpanMetricsSummaryResponse, error) {
	joinURL, _ := url.Parse(c.BaseURL + api.PathSpanMetricsSummary + "?")
	q := joinURL.Query()
	if start != 0 && end != 0 {
		q.Set("start", strconv.FormatInt(start, 10))
		q.Set("end", strconv.FormatInt(end, 10))
	}
	q.Set("q", query)
	q.Set("groupBy", groupBy)
	joinURL.RawQuery = q.Encode()

	m := &tempopb.SpanMetricsSummaryResponse{}
	_, err := c.getFor(fmt.Sprint(joinURL), m)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (c *Client) buildSearchQueryURL(queryType string, query string, start int64, end int64, limit int64, spss int64, params map[string]string) string {
	joinURL, _ := url.Parse(c.BaseURL + "/api/search?")
	q := joinURL.Query()
	if start != 0 && end != 0 {
		q.Set("start", strconv.FormatInt(start, 10))
		q.Set("end", strconv.FormatInt(end, 10))
	}
	if limit != 0 {
		q.Set("limit", strconv.FormatInt(limit, 10))
	}
	if spss != 0 {
		q.Set("spss", strconv.FormatInt(spss, 10))
	}
	q.Set(queryType, query)
	for k, v := range params {
		q.Set(k, v)
	}
	joinURL.RawQuery = q.Encode()

	return fmt.Sprint(joinURL)
}

func (c *Client) MetricsQueryRange(query string, start, end int, step string, exemplars int) (*tempopb.QueryRangeResponse, error) {
	joinURL, _ := url.Parse(c.BaseURL + api.PathMetricsQueryRange + "?")
	q := joinURL.Query()
	if exemplars != 0 {
		q.Set("exemplars", strconv.Itoa(exemplars))
	}
	if start != 0 && end != 0 {
		q.Set("start", strconv.Itoa(start))
		q.Set("end", strconv.Itoa(end))
	}
	if step != "" {
		q.Set("step", step)
	}
	q.Set("q", query)
	joinURL.RawQuery = q.Encode()

	m := &tempopb.QueryRangeResponse{}
	_, err := c.getFor(fmt.Sprint(joinURL), m)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (c *Client) buildTagsQueryURL(start int64, end int64) string {
	joinURL, _ := url.Parse(c.BaseURL + api.PathSearchTags + "?")
	q := joinURL.Query()
	if start != 0 && end != 0 {
		q.Set("start", strconv.FormatInt(start, 10))
		q.Set("end", strconv.FormatInt(end, 10))
	}
	joinURL.RawQuery = q.Encode()

	return fmt.Sprint(joinURL)
}

func (c *Client) buildTagsV2QueryURL(start int64, end int64) string {
	joinURL, _ := url.Parse(c.BaseURL + api.PathSearchTagsV2 + "?")
	q := joinURL.Query()
	if start != 0 && end != 0 {
		q.Set("start", strconv.FormatInt(start, 10))
		q.Set("end", strconv.FormatInt(end, 10))
	}
	joinURL.RawQuery = q.Encode()

	return fmt.Sprint(joinURL)
}

func (c *Client) buildTagValuesV2QueryURL(key string, start int64, end int64) string {
	urlPath := fmt.Sprintf(`/api/v2/search/tag/%s/values`, key)
	joinURL, _ := url.Parse(c.BaseURL + urlPath + "?")
	q := joinURL.Query()
	if start != 0 && end != 0 {
		q.Set("start", strconv.FormatInt(start, 10))
		q.Set("end", strconv.FormatInt(end, 10))
	}
	joinURL.RawQuery = q.Encode()

	return fmt.Sprint(joinURL)
}

func (c *Client) GetOverrides() (*userconfigurableoverrides.Limits, string, error) {
	req, err := http.NewRequest("GET", c.BaseURL+api.PathOverrides, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set(acceptHeader, applicationJSON)

	resp, body, err := c.doRequest(req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	limits := &userconfigurableoverrides.Limits{}
	if err = json.Unmarshal(body, limits); err != nil {
		return nil, "", fmt.Errorf("error decoding overrides, err: %v body: %s", err, string(body))
	}
	return limits, resp.Header.Get("Etag"), err
}

func (c *Client) SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error) {
	b, err := json.Marshal(limits)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL+api.PathOverrides, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("If-Match", version)

	resp, _, err := c.doRequest(req)
	return resp.Header.Get("Etag"), err
}

func (c *Client) PatchOverrides(limits *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error) {
	b, err := json.Marshal(limits)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest("PATCH", c.BaseURL+api.PathOverrides, bytes.NewBuffer(b))
	if err != nil {
		return nil, "", err
	}

	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, "", err
	}

	patchedLimits := &userconfigurableoverrides.Limits{}
	if err = json.Unmarshal(body, patchedLimits); err != nil {
		return nil, "", fmt.Errorf("error decoding overrides, err: %v body: %s", err, string(body))
	}
	return patchedLimits, resp.Header.Get("Etag"), err
}

func (c *Client) DeleteOverrides(version string) error {
	req, err := http.NewRequest("DELETE", c.BaseURL+api.PathOverrides, nil)
	if err != nil {
		return err
	}
	req.Header.Set("If-Match", version)

	_, _, err = c.doRequest(req)
	return err
}

func (c *Client) getURLWithQueryParams(endpoint string, queryParams map[string]string) string {
	joinURL, _ := url.Parse(c.BaseURL + endpoint + "?")
	q := joinURL.Query()

	for k, v := range queryParams {
		q.Set(k, v)
	}
	joinURL.RawQuery = q.Encode()

	return fmt.Sprint(joinURL)
}

// merge combines two maps of the same key and value types into a new map.
// Values from the second map overwrite duplicates.
func mergeMaps[K comparable, V any](a, b map[K]V) map[K]V {
	newMap := make(map[K]V, len(a)+len(b))
	for k, v := range a {
		newMap[k] = v
	}
	for k, v := range b {
		newMap[k] = v
	}
	return newMap
}
