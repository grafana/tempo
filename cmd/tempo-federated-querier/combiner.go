package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

// TraceMetrics contains metrics about the trace query
type TraceMetrics struct {
	InstancesQueried   int  `json:"instancesQueried"`
	InstancesResponded int  `json:"instancesResponded"`
	InstancesWithTrace int  `json:"instancesWithTrace"`
	InstancesNotFound  int  `json:"instancesNotFound"`
	InstancesFailed    int  `json:"instancesFailed"`
	TotalSpans         int  `json:"totalSpans"`
	PartialResponse    bool `json:"partialResponse"`
}

// CombineMetadata contains metadata about the combine operation
type CombineMetadata struct {
	InstancesQueried   int
	InstancesResponded int
	InstancesWithTrace int
	InstancesNotFound  int
	InstancesFailed    int
	TotalSpans         int
	PartialResponse    bool
	Errors             []string
}

// TraceCombiner combines traces from multiple Tempo instances
type TraceCombiner struct {
	maxSizeBytes int
	logger       log.Logger
}

// NewTraceCombiner creates a new trace combiner
func NewTraceCombiner(maxSizeBytes int, logger log.Logger) *TraceCombiner {
	return &TraceCombiner{
		maxSizeBytes: maxSizeBytes,
		logger:       logger,
	}
}

// CombineTraceResults combines trace results from multiple instances (v1 API - direct trace)
// Returns a tempopb.Trace which can be properly marshaled to both protobuf and JSON
func (c *TraceCombiner) CombineTraceResults(results []QueryResult) (*tempopb.Trace, *CombineMetadata, error) {
	metadata := &CombineMetadata{
		InstancesQueried: len(results),
	}

	// Use the existing trace combiner from pkg/model/trace
	combiner := trace.NewCombiner(c.maxSizeBytes, true)

	for _, result := range results {
		if result.Error != nil {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: %v", result.Instance, result.Error))
			level.Warn(c.logger).Log("msg", "instance query failed", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response != nil && result.Response.StatusCode == http.StatusNotFound {
			// 404 is a valid response - trace simply not found in this instance
			metadata.InstancesResponded++
			metadata.InstancesNotFound++
			level.Debug(c.logger).Log("msg", "trace not found in instance", "instance", result.Instance)
			continue
		}

		if result.Response != nil && result.Response.StatusCode != http.StatusOK {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: status %d", result.Instance, result.Response.StatusCode))
			level.Warn(c.logger).Log("msg", "instance returned error status", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		metadata.InstancesResponded++

		// Parse the trace using tempopb's UnmarshalFromJSONV1 which handles the batches->resourceSpans conversion
		tr := &tempopb.Trace{}
		if err := tempopb.UnmarshalFromJSONV1(result.Body, tr); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal trace", "instance", result.Instance, "err", err)
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: unmarshal error: %v", result.Instance, err))
			continue
		}

		// Check if trace has any spans
		if len(tr.ResourceSpans) == 0 {
			level.Debug(c.logger).Log("msg", "trace response has no spans", "instance", result.Instance)
			metadata.InstancesNotFound++
			continue
		}

		metadata.InstancesWithTrace++

		// Consume the trace into the combiner
		spanCount, err := combiner.Consume(tr)
		if err != nil {
			level.Warn(c.logger).Log("msg", "error consuming trace", "instance", result.Instance, "err", err)
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: consume error: %v", result.Instance, err))
		} else {
			metadata.TotalSpans += spanCount
		}

		level.Debug(c.logger).Log("msg", "consumed trace from instance", "instance", result.Instance, "spans", spanCount)
	}

	metadata.PartialResponse = metadata.InstancesFailed > 0

	// Get the combined result
	combinedTrace, spanCount := combiner.Result()
	if spanCount > 0 {
		metadata.TotalSpans = spanCount
	}

	if combinedTrace == nil || len(combinedTrace.ResourceSpans) == 0 {
		return nil, metadata, nil
	}

	// Sort the trace
	trace.SortTrace(combinedTrace)

	return combinedTrace, metadata, nil
}

// traceByIDV2Response is a helper struct to parse v2 API responses
// which wrap the trace in {"trace": {...}, "metrics": {...}}
type traceByIDV2Response struct {
	Trace   json.RawMessage `json:"trace"`
	Metrics json.RawMessage `json:"metrics,omitempty"`
}

// CombineTraceResultsV2 combines trace results from multiple instances (v2 API - wrapped response)
// Returns a tempopb.Trace which can be properly marshaled to both protobuf and JSON
func (c *TraceCombiner) CombineTraceResultsV2(results []QueryResult) (*tempopb.Trace, *CombineMetadata, error) {
	metadata := &CombineMetadata{
		InstancesQueried: len(results),
	}

	// Use the existing trace combiner from pkg/model/trace
	combiner := trace.NewCombiner(c.maxSizeBytes, true)

	for _, result := range results {
		if result.Error != nil {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: %v", result.Instance, result.Error))
			level.Warn(c.logger).Log("msg", "instance query failed", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response != nil && result.Response.StatusCode == http.StatusNotFound {
			// 404 is a valid response - trace simply not found in this instance
			metadata.InstancesResponded++
			metadata.InstancesNotFound++
			level.Debug(c.logger).Log("msg", "trace not found in instance", "instance", result.Instance)
			continue
		}

		if result.Response != nil && result.Response.StatusCode != http.StatusOK {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: status %d", result.Instance, result.Response.StatusCode))
			level.Warn(c.logger).Log("msg", "instance returned error status", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		metadata.InstancesResponded++

		// v2 API returns wrapped response: {"trace": {...}, "metrics": {...}}
		var v2Resp traceByIDV2Response
		if err := json.Unmarshal(result.Body, &v2Resp); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal v2 response wrapper", "instance", result.Instance, "err", err)
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: v2 wrapper unmarshal error: %v", result.Instance, err))
			continue
		}

		// Check if we got a trace
		if len(v2Resp.Trace) == 0 {
			level.Debug(c.logger).Log("msg", "v2 response has no trace", "instance", result.Instance)
			metadata.InstancesNotFound++
			continue
		}

		// Parse the trace from the wrapper using tempopb's UnmarshalFromJSONV1
		tr := &tempopb.Trace{}
		if err := tempopb.UnmarshalFromJSONV1(v2Resp.Trace, tr); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal trace from v2 response", "instance", result.Instance, "err", err)
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: trace unmarshal error: %v", result.Instance, err))
			continue
		}

		// Check if trace has any spans
		if len(tr.ResourceSpans) == 0 {
			level.Debug(c.logger).Log("msg", "trace response has no spans", "instance", result.Instance)
			metadata.InstancesNotFound++
			continue
		}

		metadata.InstancesWithTrace++

		// Consume the trace into the combiner
		spanCount, err := combiner.Consume(tr)
		if err != nil {
			level.Warn(c.logger).Log("msg", "error consuming trace", "instance", result.Instance, "err", err)
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: consume error: %v", result.Instance, err))
		} else {
			metadata.TotalSpans += spanCount
		}

		level.Debug(c.logger).Log("msg", "consumed trace from instance", "instance", result.Instance, "spans", spanCount)
	}

	metadata.PartialResponse = metadata.InstancesFailed > 0

	// Get the combined result
	combinedTrace, spanCount := combiner.Result()
	if spanCount > 0 {
		metadata.TotalSpans = spanCount
	}

	if combinedTrace == nil || len(combinedTrace.ResourceSpans) == 0 {
		return nil, metadata, nil
	}

	// Sort the trace
	trace.SortTrace(combinedTrace)

	return combinedTrace, metadata, nil
}
