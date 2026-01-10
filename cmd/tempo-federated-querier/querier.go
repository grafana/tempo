package main

import (
	"context"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/cmd/tempo-federated-querier/combiner"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// tempoInstanceClient represents a configured Tempo instance with its httpclient
type tempoInstanceClient struct {
	Name   string
	Client *httpclient.Client
}

// FederatedQuerier coordinates queries across multiple Tempo instances.
type FederatedQuerier struct {
	cfg       Config
	instances []tempoInstanceClient
	logger    log.Logger
}

// NewFederatedQuerier creates a new federated querier.
func NewFederatedQuerier(cfg Config, logger log.Logger) (*FederatedQuerier, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	instances := make([]tempoInstanceClient, len(cfg.Instances))
	for i, inst := range cfg.Instances {
		// Create the shared httpclient with compression support
		client := httpclient.NewWithCompression(inst.Endpoint, inst.OrgID)

		// Set custom headers
		for k, v := range inst.Headers {
			client.SetHeader(k, v)
		}

		instances[i] = tempoInstanceClient{
			Name:   inst.Name,
			Client: client,
		}
	}

	return &FederatedQuerier{
		cfg:       cfg,
		instances: instances,
		logger:    logger,
	}, nil
}

// QueryTraces queries all Tempo instances for a trace by ID
func (q *FederatedQuerier) QueryTraces(ctx context.Context, traceID string) []combiner.TraceResult {
	var wg sync.WaitGroup
	results := make([]combiner.TraceResult, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.TraceResult{
				Instance: instance.Name,
			}

			trace, err := instance.Client.QueryTrace(traceID)
			if err != nil {
				if err == util.ErrTraceNotFound {
					result.NotFound = true
				} else {
					result.Error = err
					level.Warn(q.logger).Log("msg", "trace query failed", "instance", instance.Name, "err", err)
				}
			} else {
				result.Response = trace
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// QueryTracesV2 queries all Tempo instances for a trace by ID using v2 API
func (q *FederatedQuerier) QueryTracesV2(ctx context.Context, traceID string) []combiner.TraceByIDResult {
	var wg sync.WaitGroup
	results := make([]combiner.TraceByIDResult, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.TraceByIDResult{
				Instance: instance.Name,
			}

			resp, err := instance.Client.QueryTraceV2(traceID)
			if err != nil {
				if err == util.ErrTraceNotFound {
					result.NotFound = true
				} else {
					result.Error = err
					level.Warn(q.logger).Log("msg", "trace v2 query failed", "instance", instance.Name, "err", err)
				}
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// Search queries all Tempo instances with a TraceQL search
func (q *FederatedQuerier) Search(ctx context.Context, query string, start, end int64) []combiner.SearchResult {
	var wg sync.WaitGroup
	results := make([]combiner.SearchResult, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.SearchResult{
				Instance: instance.Name,
			}

			resp, err := instance.Client.SearchTraceQLWithRange(query, start, end)
			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "search query failed", "instance", instance.Name, "err", err)
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// SearchTags queries all Tempo instances for available tags
func (q *FederatedQuerier) SearchTags(ctx context.Context, start, end int64) []combiner.SearchTagsResult {
	var wg sync.WaitGroup
	results := make([]combiner.SearchTagsResult, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.SearchTagsResult{
				Instance: instance.Name,
			}

			var resp *tempopb.SearchTagsResponse
			var err error
			if start > 0 && end > 0 {
				resp, err = instance.Client.SearchTagsWithRange(start, end)
			} else {
				resp, err = instance.Client.SearchTags()
			}

			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "tags query failed", "instance", instance.Name, "err", err)
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// SearchTagsV2 queries all Tempo instances for available tags using v2 API
func (q *FederatedQuerier) SearchTagsV2(ctx context.Context, start, end int64) []combiner.SearchTagsV2Result {
	var wg sync.WaitGroup
	results := make([]combiner.SearchTagsV2Result, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.SearchTagsV2Result{
				Instance: instance.Name,
			}

			var resp *tempopb.SearchTagsV2Response
			var err error
			if start > 0 && end > 0 {
				resp, err = instance.Client.SearchTagsV2WithRange(start, end)
			} else {
				resp, err = instance.Client.SearchTagsV2()
			}

			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "tags v2 query failed", "instance", instance.Name, "err", err)
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// SearchTagValues queries all Tempo instances for values of a specific tag
func (q *FederatedQuerier) SearchTagValues(ctx context.Context, tagName string) []combiner.SearchTagValuesResult {
	var wg sync.WaitGroup
	results := make([]combiner.SearchTagValuesResult, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.SearchTagValuesResult{
				Instance: instance.Name,
			}

			resp, err := instance.Client.SearchTagValues(tagName)
			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "tag values query failed", "instance", instance.Name, "err", err)
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// SearchTagValuesV2 queries all Tempo instances for values of a specific tag using v2 API
func (q *FederatedQuerier) SearchTagValuesV2(ctx context.Context, tagName string, query string, start, end int64) []combiner.SearchTagValuesV2Result {
	var wg sync.WaitGroup
	results := make([]combiner.SearchTagValuesV2Result, len(q.instances))

	for i, inst := range q.instances {
		wg.Add(1)
		go func(idx int, instance tempoInstanceClient) {
			defer wg.Done()

			result := combiner.SearchTagValuesV2Result{
				Instance: instance.Name,
			}

			var resp *tempopb.SearchTagValuesV2Response
			var err error
			if start > 0 && end > 0 {
				resp, err = instance.Client.SearchTagValuesV2WithRange(tagName, start, end)
			} else {
				resp, err = instance.Client.SearchTagValuesV2(tagName, query)
			}

			if err != nil {
				result.Error = err
				level.Warn(q.logger).Log("msg", "tag values v2 query failed", "instance", instance.Name, "err", err)
			} else {
				result.Response = resp
			}

			results[idx] = result
		}(i, inst)
	}

	wg.Wait()
	return results
}

// Instances returns the list of configured instances.
func (q *FederatedQuerier) Instances() []string {
	names := make([]string, len(q.instances))
	for i, inst := range q.instances {
		names[i] = inst.Name
	}
	return names
}
