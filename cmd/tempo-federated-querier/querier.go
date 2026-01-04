package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// TempoClient is an HTTP client for querying a Tempo instance
type TempoClient struct {
	name       string
	endpoint   string
	orgID      string
	timeout    time.Duration
	headers    map[string]string
	httpClient *http.Client
	logger     log.Logger
}

// NewTempoClient creates a new client for a Tempo instance
func NewTempoClient(instance TempoInstance, logger log.Logger) *TempoClient {
	return &TempoClient{
		name:     instance.Name,
		endpoint: instance.Endpoint,
		orgID:    instance.OrgID,
		timeout:  instance.Timeout,
		headers:  instance.Headers,
		httpClient: &http.Client{
			Timeout: instance.Timeout,
		},
		logger: log.With(logger, "instance", instance.Name),
	}
}

// Name returns the instance name
func (c *TempoClient) Name() string {
	return c.name
}

// GetTraceByID fetches a trace by its ID from this Tempo instance
func (c *TempoClient) GetTraceByID(ctx context.Context, traceID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/traces/%s", c.endpoint, traceID)
	return c.doRequest(ctx, http.MethodGet, url)
}

// GetTraceByIDV2 fetches a trace by its ID using the v2 API
func (c *TempoClient) GetTraceByIDV2(ctx context.Context, traceID string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/v2/traces/%s", c.endpoint, traceID)
	return c.doRequest(ctx, http.MethodGet, url)
}

// doRequest performs an HTTP request with appropriate headers
func (c *TempoClient) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - request JSON format for easier parsing
	req.Header.Set("Accept", "application/json")
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// FederatedQuerier coordinates queries across multiple Tempo instances
type FederatedQuerier struct {
	cfg     Config
	clients []*TempoClient
	logger  log.Logger
}

// NewFederatedQuerier creates a new federated querier
func NewFederatedQuerier(cfg Config, logger log.Logger) (*FederatedQuerier, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	clients := make([]*TempoClient, len(cfg.Instances))
	for i, inst := range cfg.Instances {
		clients[i] = NewTempoClient(inst, logger)
	}

	return &FederatedQuerier{
		cfg:     cfg,
		clients: clients,
		logger:  logger,
	}, nil
}

// QueryResult holds the result from a single Tempo instance
type QueryResult struct {
	Instance string
	Response *http.Response
	Body     []byte
	Error    error
}

// QueryAllInstances queries all Tempo instances in parallel and returns results
func (q *FederatedQuerier) QueryAllInstances(ctx context.Context, queryFn func(ctx context.Context, client *TempoClient) (*http.Response, error)) []QueryResult {
	var wg sync.WaitGroup
	results := make([]QueryResult, len(q.clients))

	for i, client := range q.clients {
		wg.Add(1)
		go func(idx int, c *TempoClient) {
			defer wg.Done()

			result := QueryResult{
				Instance: c.Name(),
			}

			resp, err := queryFn(ctx, c)
			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "query failed", "instance", c.Name(), "err", err)
			} else {
				result.Response = resp
				// Read body immediately to allow connection reuse
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					result.Error = fmt.Errorf("failed to read response body: %w", err)
				} else {
					result.Body = body
				}
			}

			results[idx] = result
		}(i, client)
	}

	wg.Wait()
	return results
}

// Instances returns the list of configured instances
func (q *FederatedQuerier) Instances() []string {
	names := make([]string, len(q.clients))
	for i, c := range q.clients {
		names[i] = c.Name()
	}
	return names
}
