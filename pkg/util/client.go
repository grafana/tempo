package util

import (
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

const orgIDHeader = "X-Scope-OrgID"

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

func (c *Client) getFor(url string, m proto.Message) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if len(c.OrgID) > 0 {
		req.Header.Set(orgIDHeader, c.OrgID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error searching tempo for tag %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return ErrTraceNotFound
	}

	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(resp.Body, m)
	if err != nil {
		return fmt.Errorf("error decoding %T json, err: %v", m, err)
	}

	return nil
}

func (c *Client) SearchTag(key, value string) (*tempopb.SearchResponse, error) {
	m := &tempopb.SearchResponse{}

	err := c.getFor(c.BaseURL+"/api/search?"+key+"="+value, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (c *Client) QueryTrace(id string) (*tempopb.Trace, error) {
	m := &tempopb.Trace{}
	err := c.getFor(c.BaseURL+"/api/traces/"+id, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
