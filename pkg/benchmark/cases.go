package benchmark

import "errors"

// API names, as recorded in a result.
const (
	apiTraceByID = "traceByID"
	apiSearch    = "search"
)

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
	return []benchCase{
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
}
