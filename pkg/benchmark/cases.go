package benchmark

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/v3/pkg/traceql"
)

// API names, as recorded in a result.
const (
	apiTraceByID = "traceByID"
	apiSearch    = "search"
	apiMetrics   = "metrics"
	apiMetadata  = "metadata"
)

// basicMetricsQueries are the phase 1 metrics queries: no generated queries and
// no attribute profiling, so the grouped one groups by a well-known attribute.
var basicMetricsQueries = []struct {
	id    string
	query string
}{
	{"rate", "{} | rate()"},
	{"rate-by-service", "{} | rate() by (resource.service.name)"},
}

// tagNameScopes are the scopes a tag-name lookup is measured over. None is the
// unscoped case, which reads every scope.
var tagNameScopes = []traceql.AttributeScope{
	traceql.AttributeScopeNone,
	traceql.AttributeScopeResource,
	traceql.AttributeScopeSpan,
	traceql.AttributeScopeEvent,
	traceql.AttributeScopeLink,
	traceql.AttributeScopeInstrumentation,
}

// benchCase is one query shape. It expands into the executions that make up a
// pass over it: a trace-by-ID case is one execution per ID, a search is one per
// row-group shard, mirroring how the frontend splits the work.
type benchCase struct {
	id    string
	api   string
	query string

	executions func(*BlockProfile, []Shard, RunOptions) ([]execution, error)
}

// phase1Cases is the fixed query set: no generated queries, no attribute
// profiling.
func phase1Cases() []benchCase {
	cases := []benchCase{
		{
			id:  "traceid/present",
			api: apiTraceByID,
			executions: func(p *BlockProfile, _ []Shard, o RunOptions) ([]execution, error) {
				if len(p.TraceIDs.Present) == 0 {
					return nil, errors.New("profile has no present trace IDs")
				}
				return traceByIDExecutions(p.TraceIDs.Present, o.searchOptions())
			},
		},
		{
			// The miss path is settled by the bloom filter and the row-group
			// index rather than by reading a trace, so it is measured apart.
			id:  "traceid/absent",
			api: apiTraceByID,
			executions: func(p *BlockProfile, _ []Shard, o RunOptions) ([]execution, error) {
				if len(p.TraceIDs.Absent) == 0 {
					return nil, errors.New("profile has no absent trace IDs")
				}
				return traceByIDExecutions(p.TraceIDs.Absent, o.searchOptions())
			},
		},
		{
			// No predicate to prune by, so this is the ceiling on fetch cost.
			id:    "search/nopredicate",
			api:   apiSearch,
			query: "{}",
			executions: func(p *BlockProfile, shards []Shard, o RunOptions) ([]execution, error) {
				return searchExecutions("{}", shards, p.Block, o.searchOptions()), nil
			},
		},
	}

	// Range and instant are measured apart: they differ only in the step, but
	// that decides how many intervals the aggregator keeps, so their cost is
	// not the same.
	for _, q := range basicMetricsQueries {
		for _, variant := range []struct {
			suffix  string
			instant bool
		}{
			{"", false},
			{"/instant", true},
		} {
			cases = append(cases, benchCase{
				id:    "metrics/" + q.id + variant.suffix,
				api:   apiMetrics,
				query: q.query,
				executions: func(p *BlockProfile, shards []Shard, o RunOptions) ([]execution, error) {
					if !p.Block.EndTime.After(p.Block.StartTime) {
						return nil, fmt.Errorf("block time range is %s to %s, which is not a window a metrics query can step over",
							p.Block.StartTime, p.Block.EndTime)
					}
					return metricsExecutions(q.query, variant.instant, shards, p.Block, o.searchOptions()), nil
				},
			})
		}
	}

	// Tag names, one case per scope, with no TraceQL to filter by.
	for _, scope := range tagNameScopes {
		cases = append(cases, benchCase{
			id:  "metadata/tagnames/" + scope.String(),
			api: apiMetadata,
			executions: func(_ *BlockProfile, shards []Shard, o RunOptions) ([]execution, error) {
				return tagNamesExecutions(scope, shards, o.searchOptions()), nil
			},
		})
	}

	return cases
}
