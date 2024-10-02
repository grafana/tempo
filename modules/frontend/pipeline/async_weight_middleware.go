package pipeline

import (
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/pkg/traceql"
)

type RequestType int

type WeightRequest interface {
	SetWeight(int)
	Weight() int
}

const (
	Default RequestType = iota
	TraceByID
	TraceQLSearch
	TraceQLMetrics
)

const (
	DefaultWeight       int = 1
	TraceQLSearchWeight int = 1
	TraceByIDWeight     int = 2

	maxTraceQLConditions int = 4
	maxRegexConditions   int = 1
)

type weightRequestWare struct {
	requestType RequestType
	enabled     bool
	next        AsyncRoundTripper[combiner.PipelineResponse]
}

// It increments the weight of a retriyed request
func IncrementRetriedRequestWeight(r WeightRequest) {
	r.SetWeight(r.Weight() + 1)
}

// It returns a new weight request middleware
func NewWeightRequestWare(rt RequestType, enabled bool) AsyncMiddleware[combiner.PipelineResponse] {
	return AsyncMiddlewareFunc[combiner.PipelineResponse](func(next AsyncRoundTripper[combiner.PipelineResponse]) AsyncRoundTripper[combiner.PipelineResponse] {
		return &weightRequestWare{
			requestType: rt,
			enabled:     enabled,
			next:        next,
		}
	})
}

func (c weightRequestWare) RoundTrip(req Request) (Responses[combiner.PipelineResponse], error) {
	c.setWeight(req)
	return c.next.RoundTrip(req)
}

func (c weightRequestWare) setWeight(req Request) {
	if !c.enabled {
		req.SetWeight(DefaultWeight)
		return
	}
	switch c.requestType {
	case TraceByID:
		req.SetWeight(TraceByIDWeight)
	case TraceQLSearch, TraceQLMetrics:
		setTraceQLWeight(req)
	default:
		req.SetWeight(DefaultWeight)
	}
}

func setTraceQLWeight(req Request) {
	var traceQLQuery string
	query := req.HTTPRequest().URL.Query()
	if query.Has("q") {
		traceQLQuery = query.Get("q")
	}
	if query.Has("query") {
		traceQLQuery = query.Get("query")
	}

	req.SetWeight(TraceQLSearchWeight)

	if traceQLQuery == "" {
		return
	}

	_, _, _, spanRequest, err := traceql.Compile(traceQLQuery)
	if err != nil || spanRequest == nil {
		return
	}

	conditions := 0
	regexConditions := 0

	for _, c := range spanRequest.Conditions {
		if c.Op != traceql.OpNone {
			conditions++
		}
		if c.Op == traceql.OpRegex || c.Op == traceql.OpNotRegex {
			regexConditions++
		}
	}
	complexQuery := regexConditions >= maxRegexConditions || conditions >= maxTraceQLConditions
	if complexQuery {
		req.SetWeight(TraceQLSearchWeight + 1)
	}
}
