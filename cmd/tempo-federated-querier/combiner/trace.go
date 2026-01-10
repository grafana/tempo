package combiner

import (
	"fmt"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CombineTraceResults combines trace results from multiple instances
// Returns a tempopb.Trace which can be properly marshaled to both protobuf and JSON
func (c *Combiner) CombineTraceResults(results []TraceResult) (*tempopb.Trace, *CombineMetadata, error) {
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

		if result.NotFound {
			// 404 is a valid response - trace simply not found in this instance
			metadata.InstancesResponded++
			metadata.InstancesNotFound++
			level.Debug(c.logger).Log("msg", "trace not found in instance", "instance", result.Instance)
			continue
		}

		metadata.InstancesResponded++

		tr := result.Response
		if tr == nil || len(tr.ResourceSpans) == 0 {
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

// CombineTraceResultsV2 combines trace results from multiple instances (v2 API with metrics)
// Returns a tempopb.Trace which can be properly marshaled to both protobuf and JSON
func (c *Combiner) CombineTraceResultsV2(results []TraceByIDResult) (*tempopb.Trace, *CombineMetadata, error) {
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

		if result.NotFound {
			// 404 is a valid response - trace simply not found in this instance
			metadata.InstancesResponded++
			metadata.InstancesNotFound++
			level.Debug(c.logger).Log("msg", "trace not found in instance", "instance", result.Instance)
			continue
		}

		metadata.InstancesResponded++

		resp := result.Response
		if resp == nil || resp.Trace == nil || len(resp.Trace.ResourceSpans) == 0 {
			level.Debug(c.logger).Log("msg", "trace response has no spans", "instance", result.Instance)
			metadata.InstancesNotFound++
			continue
		}

		metadata.InstancesWithTrace++

		// Consume the trace into the combiner
		spanCount, err := combiner.Consume(resp.Trace)
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

