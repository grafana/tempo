package util

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/klauspost/compress/gzhttp"
)

const (
	orgIDHeader = "X-Scope-OrgID"

	QueryTraceEndpoint = "/api/traces"
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error searching tempo for tag %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		return resp, fmt.Errorf("GET request to %s failed with response: %d", req.URL.String(), resp.StatusCode)
	}

	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(resp.Body, m)
	if err != nil {
		return resp, fmt.Errorf("error decoding %T json, err: %v", m, err)
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

func (c *Client) QueryTrace(id string) (*tempopb.Trace, error) {
	m, _, err := c.QueryTraceWithResponse(id)
	return m, err
}

func (c *Client) QueryTraceWithResponse(id string) (*tempopb.Trace, *http.Response, error) {
	m := &tempopb.Trace{}
	resp, err := c.getFor(c.BaseURL+QueryTraceEndpoint+"/"+id, m)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, resp, ErrTraceNotFound
		}
		return nil, resp, err
	}

	return m, resp, nil
}
