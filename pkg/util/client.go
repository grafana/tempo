package util

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb" //nolint:all
	"github.com/golang/protobuf/proto"  //nolint:all
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/klauspost/compress/gzhttp"
)

const (
	orgIDHeader = "X-Scope-OrgID"

	QueryTraceEndpoint = "/api/traces"

	AcceptHeader        = "Accept"
	ApplicationProtobuf = "application/protobuf"
	ApplicationJSON     = "application/json"
)

// Client is client to the Tempo API.
type Client struct {
	BaseURL string
	OrgID   string
	client  *http.Client
}

func NewClient(baseURL, orgID string) *Client {
	return &Client{
		BaseURL: baseURL,
		OrgID:   orgID,
		client:  http.DefaultClient,
	}
}

func NewClientWithCompression(baseURL, orgID string) *Client {
	c := NewClient(baseURL, orgID)
	c.WithTransport(gzhttp.Transport(http.DefaultTransport))
	return c
}

func (c *Client) WithTransport(t http.RoundTripper) {
	c.client.Transport = t
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *Client) getFor(url string, m proto.Message) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if len(c.OrgID) > 0 {
		req.Header.Set(orgIDHeader, c.OrgID)
	}

	marshallingFormat := ApplicationJSON
	if strings.Contains(url, QueryTraceEndpoint) {
		marshallingFormat = ApplicationProtobuf
	}
	req.Header.Set(AcceptHeader, marshallingFormat)

	// // Set 'Accept' header to 'application/protobuf'.
	// // This is required for the /api/traces endpoint to return a protobuf response.
	// // JSON lost backwards compatibility with the upgrade to `opentelemetry-proto` v0.18.0.
	// req.Header.Set("Accept", "application/protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error searching tempo for tag %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		body, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("GET request to %s failed with response: %d body: %s", req.URL.String(), resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch marshallingFormat {
	case ApplicationJSON:
		if err = jsonpb.UnmarshalString(string(body), m); err != nil {
			return resp, fmt.Errorf("error decoding %T json, err: %v body: %s", m, err, string(body))
		}
	case ApplicationProtobuf:

		if err = proto.Unmarshal(body, m); err != nil {
			return nil, fmt.Errorf("error decoding %T proto, err: %w body: %s", m, err, string(body))
		}
	}

	return resp, nil
}

func (c *Client) SearchTags() (*tempopb.SearchTagsResponse, error) {
	m := &tempopb.SearchTagsResponse{}
	_, err := c.getFor(c.BaseURL+"/api/search/tags", m)
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

// Search Tempo. tags must be in logfmt format, that is "key1=value1 key2=value2"
func (c *Client) Search(tags string) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.BaseURL+"/api/search?tags="+url.QueryEscape(tags), m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// SearchWithRange calls the /api/search endpoint. tags is expected to be in logfmt format and start/end are unix
// epoch timestamps in seconds.
func (c *Client) SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}
	_, err := c.getFor(c.BaseURL+"/api/search?tags="+url.QueryEscape(tags)+"&start="+strconv.FormatInt(start, 10)+"&end="+strconv.FormatInt(end, 10), m)
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
			return nil, ErrTraceNotFound
		}
		return nil, err
	}

	return m, nil
}
