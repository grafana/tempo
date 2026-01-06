package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/cmd/tempo-federated-querier/client"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/combiner"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/handler"
)

// FederatedQuerier coordinates queries across multiple Tempo instances.
type FederatedQuerier struct {
	cfg     Config
	clients []*client.Client
	logger  log.Logger
}

// Ensure FederatedQuerier implements handler.FederatedQuerier
var _ handler.FederatedQuerier = (*FederatedQuerier)(nil)

// NewFederatedQuerier creates a new federated querier.
func NewFederatedQuerier(cfg Config, logger log.Logger) (*FederatedQuerier, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	clients := make([]*client.Client, len(cfg.Instances))
	for i, inst := range cfg.Instances {
		clients[i] = client.New(client.Config{
			Name:     inst.Name,
			Endpoint: inst.Endpoint,
			OrgID:    inst.OrgID,
			Timeout:  inst.Timeout,
			Headers:  inst.Headers,
		}, logger)
	}

	return &FederatedQuerier{
		cfg:     cfg,
		clients: clients,
		logger:  logger,
	}, nil
}

// QueryAllInstances queries all Tempo instances in parallel and returns results.
func (q *FederatedQuerier) QueryAllInstances(ctx context.Context, queryFn func(ctx context.Context, c client.TempoClient) (*http.Response, error)) []combiner.QueryResult {
	var wg sync.WaitGroup
	results := make([]combiner.QueryResult, len(q.clients))

	for i, c := range q.clients {
		wg.Add(1)
		go func(idx int, tempoClient *client.Client) {
			defer wg.Done()

			result := combiner.QueryResult{
				Instance: tempoClient.Name(),
			}

			resp, err := queryFn(ctx, tempoClient)
			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "query failed", "instance", tempoClient.Name(), "err", err)
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
		}(i, c)
	}

	wg.Wait()
	return results
}

// Instances returns the list of configured instances.
func (q *FederatedQuerier) Instances() []string {
	names := make([]string, len(q.clients))
	for i, c := range q.clients {
		names[i] = c.Name()
	}
	return names
}
