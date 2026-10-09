package benchmark

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/grafana/tempo/v3/pkg/benchmark/profile"
	"github.com/grafana/tempo/v3/pkg/benchmark/profile/attributes"
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
}

// phase1Cases is the fixed query set: no generated queries, no attribute profiling.
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
		})
	}

	return cases
}

// selectivityBand names the band a selectivity falls in, at cuts where a
// predicate's cost changes character: high reads about everything it scans,
// low is rejected early, medium filters as it goes.
func selectivityBand(s float64) string {
	switch {
	case s >= 0.5:
		return "high"
	case s >= 0.1:
		return "medium"
	default:
		return "low"
	}
}

// phase2Cases generates a search per selectivity band: high, medium, low.
// The selectivity indicates a predicate's cost.
func phase2Cases(prof *profile.BlockProfile) []benchCase {
	if prof.Attributes == nil {
		return nil
	}

	byBand := map[string]string{}
	for _, a := range prof.Attributes.Ranked {
		if len(a.Values) == 0 {
			continue // quantiles stand for `>` predicates, a later phase
		}
		ref, ok := attributeReference(a.Scope, a.Name)
		if !ok {
			continue
		}
		if band := selectivityBand(a.Values[0].Selectivity); byBand[band] == "" {
			byBand[band] = fmt.Sprintf("{ %s = %s }", ref, valueLiteral(a.Type, a.Values[0].Value))
		}
	}

	var cases []benchCase
	for _, band := range []string{"high", "medium", "low"} {
		query := byBand[band]
		if query == "" {
			continue // a block can lack a band entirely
		}
		cases = append(cases, benchCase{
			id:    "search/attr/" + band,
			api:   apiSearch,
			query: query,
			executions: func(_ *profile.BlockProfile, shards []Shard, opts RunOptions) ([]execution, error) {
				return searchExecutions(query, shards, prof.Block, opts.searchOptions()), nil
			},
		})
	}
	return cases
}

// attributeReference writes a profile attribute as a TraceQL reference:
// resource and span attributes carry their scope's prefix, intrinsics are bare.
func attributeReference(scope, name string) (string, bool) {
	switch scope {
	case attributes.ScopeResource:
		return "resource." + attributeIdentifier(name), true
	case attributes.ScopeSpan:
		return "span." + attributeIdentifier(name), true
	case attributes.ScopeIntrinsic:
		return attributeIdentifier(name), true
	}
	return "", false
}

// attributeIdentifier quotes a name the TraceQL lexer would not read as one token:
// whitespace and the structural characters end an attribute,
// and a quote starts a quoted part, where a quote and a backslash escape.
func attributeIdentifier(name string) string {
	if !strings.ContainsFunc(name, endsAttribute) {
		return name
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name) + `"`
}

func endsAttribute(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(`{}()=~!<>&|^,"`, r)
}

// valueLiteral quotes a string value; every other profile type is already TraceQL literal syntax.
func valueLiteral(typ, value string) string {
	if typ == attributes.TypeString {
		return strconv.Quote(value)
	}
	return value
}
