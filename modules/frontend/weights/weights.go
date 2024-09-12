package weights

import "github.com/grafana/tempo/pkg/traceql"

// PRTODO: test weight calculation
type WeightRequest interface {
	SetWeight(int)
	Weight() int
}

func TraceByID() int {
	return 2
}

func FetchSpans(req *traceql.FetchSpansRequest) int {
	weight := 1

	if req == nil {
		return weight
	}

	if !req.AllConditions {
		weight++
	}

	conditions := 0
	regexConditions := 0

	for _, c := range req.Conditions {
		if c.Op != traceql.OpNone {
			conditions++
		}
		if c.Op == traceql.OpRegex || c.Op == traceql.OpNotRegex {
			regexConditions++
		}
	}

	if conditions > 4 { // yay, magic!
		weight++
	}
	if regexConditions > 0 {
		weight++
	}

	return weight
}

func RetryRequest(r WeightRequest) {
	r.SetWeight(r.Weight() + 1)
}
