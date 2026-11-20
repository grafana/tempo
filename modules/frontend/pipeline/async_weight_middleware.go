package pipeline

import (
	"context"
	"sync/atomic"

	"github.com/grafana/tempo/v3/modules/frontend/combiner"
	"github.com/grafana/tempo/v3/pkg/traceql"
)

type RequestType int

type WeightRequest interface {
	SetWeight(int)
	Weight() int
}

// Query type values surfaced as `query_type` on logs and spans.
const (
	QueryTypeTraces   = "traces"
	QueryTypeSearch   = "search"
	QueryTypeMetrics  = "metrics"
	QueryTypeMetadata = "metadata"
)

// QueryShape captures the structural shape of a query; populated by the
// async-weight middleware, propagated via the Request and via context.
type QueryShape struct {
	Type            string
	Weight          int
	SubQueries      int
	Conditions      int
	RegexConditions int
	HasOr           bool
	NeedsFullTrace  bool
	SelectAll       bool
}

type queryShapeCtxKey struct{}

// queryShapeCell is the mutable per-query cell on the handler's outer context. The middleware writes
// the shape into it so the handler can read it after RoundTrip returns. The shape has a single
// writer (middleware) and a single reader (handler), serialized by the pipeline; no mutex.
// The estimated bytes of the jobs are added by the combiners and have their own atomic.
type queryShapeCell struct {
	qs *QueryShape

	// estimatedBytes is the sum of the estimated bytes of the jobs of the query. The combiners add to it as
	// job responses arrive, which can be concurrent.
	estimatedBytes atomic.Uint64
}

// WithQueryShapeCell installs an empty cell on ctx that the weight middleware
// populates during RoundTrip. The cell also sums up the estimated bytes of the jobs of the query.
func WithQueryShapeCell(ctx context.Context) context.Context {
	return context.WithValue(ctx, queryShapeCtxKey{}, &queryShapeCell{})
}

// QueryShapeFromContext returns the shape stamped on ctx, or ok=false when none.
func QueryShapeFromContext(ctx context.Context) (QueryShape, bool) {
	cell := queryShapeCellFromContext(ctx)
	if cell == nil || cell.qs == nil {
		return QueryShape{}, false
	}
	return *cell.qs, true
}

// EstimatedBytesAdder returns a function that adds to the estimated bytes of the jobs of the query whose cell
// is installed on ctx. Without a cell the function does nothing. It is meant to be called once per query.
func EstimatedBytesAdder(ctx context.Context) func(bytes uint64) {
	cell := queryShapeCellFromContext(ctx)
	if cell == nil {
		return func(uint64) {}
	}
	return func(bytes uint64) { cell.estimatedBytes.Add(bytes) }
}

// EstimatedBytesFromContext returns the sum of the estimated bytes of the jobs of the query added so far,
// or ok=false when ctx has no cell.
func EstimatedBytesFromContext(ctx context.Context) (bytes uint64, ok bool) {
	cell := queryShapeCellFromContext(ctx)
	if cell == nil {
		return 0, false
	}
	return cell.estimatedBytes.Load(), true
}

func setQueryShapeOnContext(ctx context.Context, qs QueryShape) {
	cell := queryShapeCellFromContext(ctx)
	if cell == nil {
		return
	}
	snapshot := qs
	cell.qs = &snapshot
}

// queryShapeCellFromContext returns the cell installed on ctx, or nil if there is none.
func queryShapeCellFromContext(ctx context.Context) *queryShapeCell {
	if ctx == nil {
		return nil
	}
	cell, _ := ctx.Value(queryShapeCtxKey{}).(*queryShapeCell)
	return cell
}

func (rt RequestType) queryTypeLabel() string {
	switch rt {
	case TraceByID:
		return QueryTypeTraces
	case TraceQLSearch:
		return QueryTypeSearch
	case TraceQLMetrics:
		return QueryTypeMetrics
	default:
		return QueryTypeMetadata
	}
}

type WeightsConfig struct {
	RequestWithWeights   bool `yaml:"request_with_weights,omitempty"`
	RetryWithWeights     bool `yaml:"retry_with_weights,omitempty"`
	MaxTraceQLConditions int  `yaml:"max_traceql_conditions,omitempty"`
	MaxRegexConditions   int  `yaml:"max_regex_conditions,omitempty"`
}

type Weights struct {
	DefaultWeight        int
	TraceQLSearchWeight  int
	TraceByIDWeight      int
	MaxTraceQLConditions int
	MaxRegexConditions   int
}

const (
	Default RequestType = iota
	TraceByID
	TraceQLSearch
	TraceQLMetrics
)

type weightRequestWare struct {
	requestType RequestType
	enabled     bool
	next        AsyncRoundTripper[combiner.PipelineResponse]

	weights Weights
}

// It increments the weight of a retriyed request
func IncrementRetriedRequestWeight(r WeightRequest) {
	r.SetWeight(r.Weight() + 1)
}

// It returns a new weight request middleware
func NewWeightRequestWare(rt RequestType, cfg WeightsConfig) AsyncMiddleware[combiner.PipelineResponse] {
	weights := Weights{
		DefaultWeight:        1,
		TraceQLSearchWeight:  1,
		TraceByIDWeight:      2,
		MaxTraceQLConditions: cfg.MaxTraceQLConditions,
		MaxRegexConditions:   cfg.MaxRegexConditions,
	}
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &weightRequestWare{
			requestType: rt,
			enabled:     cfg.RequestWithWeights,
			weights:     weights,
			next:        next,
		}
	})
}

func (c weightRequestWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	c.setWeight(req)
	return c.next.RoundTrip(req)
}

func (c weightRequestWare) setWeight(req Request) {
	qType := c.requestType.queryTypeLabel()
	if !c.enabled {
		req.SetWeight(c.weights.DefaultWeight)
		c.stampQueryShape(req, QueryShape{Type: qType, Weight: c.weights.DefaultWeight})
		return
	}
	switch c.requestType {
	case TraceByID:
		req.SetWeight(c.weights.TraceByIDWeight)
		c.stampQueryShape(req, QueryShape{Type: qType, Weight: c.weights.TraceByIDWeight})
	case TraceQLSearch, TraceQLMetrics:
		c.setTraceQLWeight(req)
	default:
		req.SetWeight(c.weights.DefaultWeight)
		c.stampQueryShape(req, QueryShape{Type: qType, Weight: c.weights.DefaultWeight})
	}
}

// stampQueryShape stores the shape both on the Request (for sharders) and via
// the mutable cell installed by WithQueryShapeCell (for handlers, which keep a
// reference to the outer *http.Request).
func (c weightRequestWare) stampQueryShape(req Request, qs QueryShape) {
	req.SetQueryShape(qs)
	if httpReq := req.HTTPRequest(); httpReq != nil {
		setQueryShapeOnContext(httpReq.Context(), qs)
	}
}

func (c weightRequestWare) setTraceQLWeight(req Request) {
	var traceQLQuery string
	query := req.HTTPRequest().URL.Query()
	if query.Has("q") {
		traceQLQuery = query.Get("q")
	}
	if query.Has("query") {
		traceQLQuery = query.Get("query")
	}

	weight := c.weights.TraceQLSearchWeight
	req.SetWeight(weight)

	qType := c.requestType.queryTypeLabel()
	// Stamp a base shape early so malformed/empty queries still produce a shape
	// with at least Type and Weight populated.
	shape := QueryShape{Type: qType, Weight: weight}
	defer func() {
		shape.Weight = weight
		c.stampQueryShape(req, shape)
	}()

	if traceQLQuery == "" {
		return
	}

	// Calculate the weight based on the optimized TraceQL query, unless an optimization is skipped via query hints.
	// This will deliver an accurate weight for most queries but also has two caveats:
	// - Does not respect the users unsafe_query_hints option (most users can't skip optimizations, but they can affect the weight calculation)
	// - Does not take query_frontend.skip_ast_transformations into account (which is likely not set anyway)
	rootExpr, subReqs, err := traceql.CompileFetchSpanRequests(traceQLQuery, traceql.WithUnsafeHints(true))
	if err != nil || len(subReqs) == 0 {
		return
	}

	for _, spanReq := range subReqs {
		conditions := 0
		regexConditions := 0
		for _, cond := range spanReq.Conditions {
			if cond.Op != traceql.OpNone {
				conditions++
			}
			if cond.Op == traceql.OpRegex || cond.Op == traceql.OpNotRegex {
				regexConditions++
			}
		}
		if regexConditions >= c.weights.MaxRegexConditions || conditions >= c.weights.MaxTraceQLConditions {
			weight++
		}
		if !spanReq.AllConditions {
			weight++
			shape.HasOr = true
		}
		if spanReq.SecondPassSelectAll {
			weight++
			shape.SelectAll = true
		}
		shape.Conditions += conditions
		shape.RegexConditions += regexConditions
	}

	// Metrics math queries (e.g. `({} | rate()) / ({...} | rate())`) fan out
	// into one storage sub-request per side; charge an extra unit per
	// additional sub-request so weight scales with backend fan-out.
	shape.SubQueries = len(subReqs)
	if len(subReqs) > 1 {
		weight += len(subReqs) - 1
	}

	// Query that requires full trace scanning, e.g. with structural operators
	for _, pipeline := range rootExpr.Pipeline {
		if traceql.NeedsFullTrace(pipeline) {
			weight++
			shape.NeedsFullTrace = true
		}
	}

	req.SetWeight(weight)
}
