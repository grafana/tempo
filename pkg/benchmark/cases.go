package benchmark

import (
	"errors"

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

const rateCaseID = "rate"

// basicMetricsQueries are the fixed metrics queries. The fixed set does no
// attribute profiling, so the grouped query groups by a well-known attribute.
var basicMetricsQueries = []struct {
	id    string
	query string
}{
	{rateCaseID, "{} | rate()"},
	{"rate-by-service", "{} | rate() by (resource.service.name)"},
}

// drilldownQueries mirror what Traces Drilldown runs when it opens.
var drilldownQueries = []struct {
	id    string
	query string
}{
	{rateCaseID, "{nestedSetParent<0 && true} | rate() with(sample=true)"},
	{"errors", "{nestedSetParent<0 && true && status=error} | rate() with(sample=true)"},
	{"duration", "{nestedSetParent<0 && true} | histogram_over_time(duration) with(sample=true)"},
	{"by-service", "{nestedSetParent<0 && true} | rate() by (resource.service.name)"},
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
}

// fixedCases is the fixed query set: no generated queries, no attribute profiling.
func fixedCases() []benchCase {
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
	}

	// The fetch path with nothing to prune by.
	cases = append(cases, searchCase("search/nopredicate", "{}"))

	for _, q := range basicMetricsQueries {
		cases = append(cases, metricsCase("metrics/"+q.id, q.query))
	}
	for _, q := range drilldownQueries {
		cases = append(cases, metricsCase("metrics/drilldown/"+q.id, q.query))
	}

	// Tag names, one case per scope, with no TraceQL to filter by.
	for _, scope := range tagNameScopes {
		cases = append(cases, benchCase{
			id:  "metadata/tagnames/" + scope.String(),
			api: apiMetadata,
			executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				return tagNamesExecutions(scope, shards, opts.searchOptions()), nil
			},
		})
	}

	return cases
}
