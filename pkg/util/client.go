package util

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"go.uber.org/zap"
)

const orgIDHeader = "X-Scope-OrgID"

// Client is client to the Tempo API.
type Client struct {
	BaseURL string
	OrgID   string
	client  *http.Client
	logger  *zap.Logger
}

func NewClient(baseURL, orgID string, log *zap.Logger) *Client {
	return &Client{
		BaseURL: baseURL,
		OrgID:   orgID,
		client:  http.DefaultClient,
		logger:  log,
	}
}

func (c *Client) getFor(url string, m proto.Message) error {
	log := c.logger.With(
		zap.String("query_url", url),
		zap.String("message", reflect.TypeOf(m).String()),
	)

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
		if err := resp.Body.Close(); err != nil {
			log.Error("error closing body ", zap.Error(err))
		}
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
