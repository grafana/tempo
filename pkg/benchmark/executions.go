package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// execOutput is what one execution produced.
type execOutput struct {
	matched int64
	// keys are the result elements this execution produced: trace IDs for a
	// search, series keys for a metrics query, tag names for a metadata one.
	// The case's merge turns one pass's keys into the logical result count,
	// as the query frontend merges job responses. Nil when the executions are
	// the logical results already, as a trace lookup's are.
	keys []string
	// metrics is what the API reported, flattened. Nil when it reported
	// nothing, which a trace-by-ID miss does.
	metrics map[string]int64
}

type execution func(context.Context, common.BackendBlock, RunOptions) (execOutput, error)

// mergeSum counts a pass's matches as the plain sum of its executions', for a
// case whose executions are logical results already.
func mergeSum(outputs []execOutput, _ RunOptions) int64 {
	var n int64
	for _, o := range outputs {
		n += o.matched
	}
	return n
}

// mergeSearch counts a pass's matches the way the query frontend reports
// them: the distinct traces over every shard's response, capped at the search
// limit.
func mergeSearch(outputs []execOutput, opts RunOptions) int64 {
	seen := make(map[string]struct{})
	for _, o := range outputs {
		for _, id := range o.keys {
			seen[id] = struct{}{}
		}
	}
	return min(int64(len(seen)), int64(opts.SearchLimit))
}

// mergeDistinct counts a pass's matches as the distinct result elements over
// every shard's response, as the frontend unions tag names and combines
// metrics series.
func mergeDistinct(outputs []execOutput, _ RunOptions) int64 {
	seen := make(map[string]struct{})
	for _, o := range outputs {
		for _, k := range o.keys {
			seen[k] = struct{}{}
		}
	}
	return int64(len(seen))
}

// fetcherFor wraps a block's fetch methods for the TraceQL engine.
func fetcherFor(block common.BackendBlock, readOpts common.SearchOptions) traceql.SpansetFetcher {
	return traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return block.Fetch(ctx, req, readOpts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return block.FetchSpans(ctx, req, readOpts)
		},
	)
}

// shardOptions narrows read options to one shard's row groups.
func shardOptions(baseOpts common.SearchOptions, shard Shard) common.SearchOptions {
	baseOpts.StartPage, baseOpts.TotalPages = shard.StartPage, shard.TotalPages
	return baseOpts
}

// traceByIDExecutions looks up one trace per ID.
func traceByIDExecutions(hexIDs []string, readOpts common.SearchOptions) ([]execution, error) {
	executions := make([]execution, 0, len(hexIDs))
	for _, hexID := range hexIDs {
		id, err := util.HexStringToTraceID(hexID)
		if err != nil {
			return nil, fmt.Errorf("decoding trace ID %q: %w", hexID, err)
		}

		executions = append(executions, func(ctx context.Context, block common.BackendBlock, _ RunOptions) (execOutput, error) {
			resp, err := block.FindTraceByID(ctx, id, readOpts)
			if err != nil {
				return execOutput{}, err
			}

			out := execOutput{}
			if resp == nil {
				// A bloom miss returns before a response is built, so the
				// backend counter is the only signal for this execution.
				return out, nil
			}
			if resp.Trace != nil {
				out.matched = 1
			}
			if resp.Metrics != nil {
				if out.metrics, err = metrics.FromResponse(resp.Metrics); err != nil {
					return execOutput{}, err
				}
			}
			return out, nil
		})
	}
	return executions, nil
}

// searchExecutions runs a search, one execution per shard.
func searchExecutions(query string, shards []Shard, meta *backend.BlockMeta, baseOpts common.SearchOptions) []execution {
	executions := make([]execution, 0, len(shards))
	for _, shard := range shards {
		readOpts := shardOptions(baseOpts, shard)

		executions = append(executions, func(ctx context.Context, block common.BackendBlock, opts RunOptions) (execOutput, error) {
			resp, err := traceql.NewEngine().ExecuteSearch(ctx, &tempopb.SearchRequest{
				Query: query,
				Limit: uint32(opts.SearchLimit),
				Start: uint32(meta.StartTime.Unix()),
				// Round up: the field is whole seconds, so truncating down
				// would clip the block's last fractional second.
				End: uint32(meta.EndTime.Unix()) + 1,
			}, fetcherFor(block, readOpts))
			if err != nil {
				return execOutput{}, err
			}

			out := execOutput{}
			if resp == nil {
				return out, nil
			}
			out.matched = int64(len(resp.Traces))
			for _, tr := range resp.Traces {
				out.keys = append(out.keys, tr.TraceID)
			}
			if resp.Metrics != nil {
				if out.metrics, err = metrics.FromResponse(resp.Metrics); err != nil {
					return execOutput{}, err
				}
			}
			return out, nil
		})
	}
	return executions
}

// metricsStep is the step of a range query over the block.
func metricsStep(meta *backend.BlockMeta) time.Duration {
	return max(meta.EndTime.Sub(meta.StartTime)/metricsStepDivisor, minMetricsStep)
}

// metricsExecutions runs a TraceQL metrics range query over the block's whole
// time range, one execution per shard.
func metricsExecutions(query string, shards []Shard, meta *backend.BlockMeta, baseOpts common.SearchOptions) []execution {
	var (
		start = uint64(meta.StartTime.UnixNano())
		end   = uint64(meta.EndTime.UnixNano())
	)

	executions := make([]execution, 0, len(shards))
	for _, shard := range shards {
		readOpts := shardOptions(baseOpts, shard)

		executions = append(executions, func(ctx context.Context, block common.BackendBlock, opts RunOptions) (execOutput, error) {
			req := &tempopb.QueryRangeRequest{
				Query:     query,
				Start:     start,
				End:       end,
				Step:      uint64(metricsStep(meta)),
				MaxSeries: uint32(opts.MaxSeries),
				Exemplars: uint32(opts.Exemplars),
			}
			// Align to step boundaries like the query frontend does.
			traceql.AlignRequest(req)

			eval, err := traceql.NewEngine().CompileMetricsQueryRange(req,
				traceql.WithUnsafeHints(true),
				traceql.WithEngineBytesTracking(true),
			)
			if err != nil {
				return execOutput{}, fmt.Errorf("compiling %q: %w", query, err)
			}

			// Do takes the block's range, not the request's, as the querier
			// does: the request is the query window, Do bounds the fetch.
			if err := eval.Do(ctx, fetcherFor(block, readOpts), start, end, opts.MaxSeries); err != nil {
				return execOutput{}, err
			}

			// Results does the final series processing, so it is part of what
			// the query costs and has to run inside the measured window.
			results := eval.Results()

			out := execOutput{matched: int64(len(results))}
			for _, ts := range results {
				out.keys = append(out.keys, ts.Labels.String())
			}
			if out.metrics, err = metrics.FromEvaluator(eval.Metrics()); err != nil {
				return execOutput{}, err
			}
			return out, nil
		})
	}
	return executions
}

// tagNamesExecutions lists tag names in one scope, one execution per shard, as
// the frontend's tag jobs do.
//
// This API takes a callback for bytes read instead of returning a metrics
// message, so bytes is all it can report. The backend counter and the process
// metrics cover the rest.
func tagNamesExecutions(scope traceql.AttributeScope, shards []Shard, baseOpts common.SearchOptions) []execution {
	executions := make([]execution, 0, len(shards))
	for _, shard := range shards {
		readOpts := shardOptions(baseOpts, shard)

		executions = append(executions, func(ctx context.Context, block common.BackendBlock, _ RunOptions) (execOutput, error) {
			var names []string
			var bytesRead int64
			err := block.SearchTags(ctx, scope,
				func(name string, _ traceql.AttributeScope) { names = append(names, name) },
				func(b uint64) { bytesRead += int64(b) },
				readOpts,
			)
			if err != nil {
				return execOutput{}, err
			}

			out := execOutput{matched: int64(len(names)), keys: names}
			if out.metrics, err = metrics.BytesRead(bytesRead); err != nil {
				return execOutput{}, err
			}
			return out, nil
		})
	}
	return executions
}
