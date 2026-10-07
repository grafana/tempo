package benchmark

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

// API names, as recorded in a result.
const (
	apiTraceByID = "traceByID"
	apiSearch    = "search"
	apiMetrics   = "metrics"
	apiMetadata  = "metadata"
)

// basicMetricsQueries are the phase 1 metrics queries. Phase 1 does no
// attribute profiling, so the grouped query groups by a well-known attribute.
var basicMetricsQueries = []struct {
	id    string
	query string
}{
	{"rate", "{} | rate()"},
	{"rate-by-service", "{} | rate() by (resource.service.name)"},
}

// tagNameScopes are the scopes a tag-name lookup is measured over. None means
// unscoped, which reads them all.
var tagNameScopes = []traceql.AttributeScope{
	traceql.AttributeScopeNone,
	traceql.AttributeScopeResource,
	traceql.AttributeScopeSpan,
	traceql.AttributeScopeEvent,
	traceql.AttributeScopeLink,
	traceql.AttributeScopeInstrumentation,
}

// benchCase is one query shape. It expands into the executions that make up a
// pass over it: one per trace ID, or one per row-group shard, mirroring how the
// frontend splits the work.
type benchCase struct {
	id    string
	api   string
	query string

	executions func(*profile.BlockProfile, []Shard, RunOptions) ([]execution, error)
	// merge turns one pass's execution outputs into the case's match count,
	// the way the query frontend merges job responses into the logical
	// result. Nil means the executions are the logical results already, and
	// their matches are simply summed.
	merge func(outputs []execOutput, opts RunOptions) int64
}

// phase1Cases is the fixed query set: no generated queries, no attribute
// profiling.
func phase1Cases() []benchCase {
	cases := []benchCase{
		{
			id:  "traceid/present",
			api: apiTraceByID,
			executions: func(prof *profile.BlockProfile, _ []Shard, opts RunOptions) ([]execution, error) {
				if len(prof.TraceIDs.Present) == 0 {
					return nil, errors.New("profile has no present trace IDs")
				}
				return traceByIDExecutions(prof.TraceIDs.Present, opts.searchOptions())
			},
		},
		{
			// The miss path is settled by the bloom filter and the row-group
			// index rather than by reading a trace, so it is measured apart.
			id:  "traceid/absent",
			api: apiTraceByID,
			executions: func(prof *profile.BlockProfile, _ []Shard, opts RunOptions) ([]execution, error) {
				if len(prof.TraceIDs.Absent) == 0 {
					return nil, errors.New("profile has no absent trace IDs")
				}
				return traceByIDExecutions(prof.TraceIDs.Absent, opts.searchOptions())
			},
		},
		{
			// The fetch path with nothing to prune by.
			id:    "search/nopredicate",
			api:   apiSearch,
			query: "{}",
			executions: func(prof *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				return searchExecutions("{}", shards, prof.Block, opts.searchOptions()), nil
			},
			merge: mergeSearch,
		},
	}

	for _, q := range basicMetricsQueries {
		cases = append(cases, benchCase{
			id:    "metrics/" + q.id,
			api:   apiMetrics,
			query: q.query,
			executions: func(prof *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				if !prof.Block.EndTime.After(prof.Block.StartTime) {
					return nil, fmt.Errorf("a metrics query needs a non-empty time window, but the block's is %s to %s",
						prof.Block.StartTime, prof.Block.EndTime)
				}
				return metricsExecutions(q.query, shards, prof.Block, opts.searchOptions()), nil
			},
			merge: mergeDistinct,
		})
	}

	// Tag names, one case per scope, with no TraceQL to filter by.
	for _, scope := range tagNameScopes {
		cases = append(cases, benchCase{
			id:  "metadata/tagnames/" + scope.String(),
			api: apiMetadata,
			executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				return tagNamesExecutions(scope, shards, opts.searchOptions()), nil
			},
			merge: mergeDistinct,
		})
	}

	return cases
}
