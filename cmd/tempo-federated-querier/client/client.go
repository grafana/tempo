package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// TempoClient defines the interface for querying a Tempo instance.
type TempoClient interface {
	Name() string
	GetTraceByID(ctx context.Context, traceID string) (*http.Response, error)
	GetTraceByIDV2(ctx context.Context, traceID string) (*http.Response, error)
	Search(ctx context.Context, queryParams string) (*http.Response, error)
	SearchTags(ctx context.Context, queryParams string) (*http.Response, error)
	SearchTagsV2(ctx context.Context, queryParams string) (*http.Response, error)
	SearchTagValues(ctx context.Context, tagName, queryParams string) (*http.Response, error)
	SearchTagValuesV2(ctx context.Context, tagName, queryParams string) (*http.Response, error)
}

// Client is an HTTP client for communicating with a single Tempo instance.
type Client struct {
	name       string
	endpoint   string
	orgID      string
	timeout    time.Duration
	headers    map[string]string
	httpClient *http.Client
	logger     log.Logger
}

// Config holds the configuration for a single Tempo client.
type Config struct {
	// Name is a friendly name for this instance
	Name string
	// Endpoint is the base URL for this Tempo instance (e.g., "http://tempo-1:3200")
	Endpoint string
	// OrgID is the tenant ID to use for this instance (optional)
	OrgID string
	// Timeout is the request timeout for this instance
	Timeout time.Duration
	// Headers are additional headers to send with requests
	Headers map[string]string
}

// New creates a new Tempo client with the given configuration.
func New(cfg Config, logger log.Logger) *Client {
	return &Client{
		name:     cfg.Name,
		endpoint: cfg.Endpoint,
		orgID:    cfg.OrgID,
		timeout:  cfg.Timeout,
		headers:  cfg.Headers,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: log.With(logger, "instance", cfg.Name),
	}
}

// Name returns the friendly name of this Tempo instance.
func (c *Client) Name() string {
	return c.name
}

// GetTraceByID retrieves a trace by its ID.
func (c *Client) GetTraceByID(ctx context.Context, traceID string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/traces/%s", traceID), nil)
}

// GetTraceByIDV2 retrieves a trace by its ID using the v2 API.
func (c *Client) GetTraceByIDV2(ctx context.Context, traceID string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v2/traces/%s", traceID), nil)
}

// Search performs a TraceQL search query.
func (c *Client) Search(ctx context.Context, queryParams string) (*http.Response, error) {
	path := "/api/search"
	if queryParams != "" {
		path = fmt.Sprintf("%s?%s", path, queryParams)
	}
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// SearchTags retrieves available tag names.
func (c *Client) SearchTags(ctx context.Context, queryParams string) (*http.Response, error) {
	path := "/api/search/tags"
	if queryParams != "" {
		path = fmt.Sprintf("%s?%s", path, queryParams)
	}
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// SearchTagsV2 retrieves available tag names using the v2 API.
func (c *Client) SearchTagsV2(ctx context.Context, queryParams string) (*http.Response, error) {
	path := "/api/v2/search/tags"
	if queryParams != "" {
		path = fmt.Sprintf("%s?%s", path, queryParams)
	}
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// SearchTagValues retrieves values for a specific tag.
func (c *Client) SearchTagValues(ctx context.Context, tagName, queryParams string) (*http.Response, error) {
	path := fmt.Sprintf("/api/search/tag/%s/values", tagName)
	if queryParams != "" {
		path = fmt.Sprintf("%s?%s", path, queryParams)
	}
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// SearchTagValuesV2 retrieves values for a specific tag using the v2 API.
func (c *Client) SearchTagValuesV2(ctx context.Context, tagName, queryParams string) (*http.Response, error) {
	path := fmt.Sprintf("/api/v2/search/tag/%s/values", tagName)
	if queryParams != "" {
		path = fmt.Sprintf("%s?%s", path, queryParams)
	}
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// doRequest executes an HTTP request to the Tempo instance.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.endpoint + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set org ID header if configured
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	// Set custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	level.Debug(c.logger).Log("msg", "sending request", "method", method, "url", url)
	return c.httpClient.Do(req)
}
