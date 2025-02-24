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
	if !c.enabled {
		req.SetWeight(c.weights.DefaultWeight)
		return
	}
	switch c.requestType {
	case TraceByID:
		req.SetWeight(c.weights.TraceByIDWeight)
	case TraceQLSearch, TraceQLMetrics:
		c.setTraceQLWeight(req)
	default:
		req.SetWeight(c.weights.DefaultWeight)
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

	req.SetWeight(c.weights.TraceQLSearchWeight)

	if traceQLQuery == "" {
		return
	}

	_, _, _, _, spanRequest, err := traceql.Compile(traceQLQuery)
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
	complexQuery := regexConditions >= c.weights.MaxRegexConditions || conditions >= c.weights.MaxTraceQLConditions
	if complexQuery {
		req.SetWeight(c.weights.TraceQLSearchWeight + 1)
	}
}
