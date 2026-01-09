package external

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/dskit/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
)

type Client struct {
	httpClient  *http.Client
	externalURL *url.URL
}

func NewClient(endpoint string, timeout time.Duration) (*Client, error) {
	externalURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid external endpoint URL: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		externalURL: externalURL,
	}, nil
}

// TraceByID forwards a trace-by-ID request to the external endpoint
// traceID is the trace ID to query
// startTime and endTime are Unix timestamps in seconds (0 means not specified)
func (c *Client) TraceByID(ctx context.Context, userID string, traceID []byte, startTime, endTime int64) (*tempopb.TraceByIDResponse, error) {
	path := c.externalURL.JoinPath(strings.Replace(api.PathTracesV2, "{traceID}", hex.EncodeToString(traceID), 1))

	// Add query parameters for start/end times
	q := path.Query()
	if startTime != 0 {
		q.Set("start", strconv.FormatInt(startTime, 10))
	}
	if endTime != 0 {
		q.Set("end", strconv.FormatInt(endTime, 10))
	}
	path.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, path.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create external request: %w", err)
	}

	httpReq.Header.Set(api.HeaderAccept, api.HeaderAcceptProtobuf)
	httpReq.Header.Set(user.OrgIDHeaderName, userID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("external endpoint request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read external response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var trace tempopb.Trace
	err = trace.Unmarshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal external response: %w", err)
	}

	return &tempopb.TraceByIDResponse{
		Trace: &trace,
	}, nil
}
