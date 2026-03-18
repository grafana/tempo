package traceql

import (
	"strings"
)

const emptyQuery = "{}"

// ExtractConditions extracts filter conditions from a query string.
// It parses the query using the lenient parser (which handles incomplete matchers like `.foo=`)
// and walks the AST to extract conditions. Conditions with OpNone (from incomplete matchers) are filtered out.
//
// Returns nil if:
//   - The query is empty
//   - Parsing fails completely
//   - The query contains structural operators (multiple spansets)
//
// The caller should check AllConditions to determine whether the conditions
// can be used as filters (true = all AND, false = contains OR).
func ExtractConditions(query string) *FetchSpansRequest {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil
	}

	// Find the first SpansetFilter in the pipeline.
	// Returns nil for structural operators (SpansetOperation) indicating multiple spansets.
	filter := findSpansetFilter(expr.Pipeline)
	if filter == nil {
		return nil
	}

	// Extract conditions from the AST. AllConditions is set to false by
	// extractConditions when OR operators are present.
	req := &FetchSpansRequest{AllConditions: true}
	filter.Expression.extractConditions(req)

	// Filter out OpNone conditions (incomplete matchers)
	conditions := make([]Condition, 0, len(req.Conditions))
	for _, cond := range req.Conditions {
		if cond.Op != OpNone {
			conditions = append(conditions, cond)
		}
	}
	req.Conditions = conditions

	return req
}

// ExtractMatchers extracts matchers from a query string and returns a string
// that can be parsed by the storage layer.
func ExtractMatchers(query string) string {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return emptyQuery
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return emptyQuery
	}

	return expr.String()
}

// findSpansetFilter returns the first SpansetFilter in the pipeline.
// Returns nil if the pipeline contains structural operators (SpansetOperation)
// which indicate multiple spansets.
func findSpansetFilter(p Pipeline) *SpansetFilter {
	if len(p.Elements) == 0 {
		return nil
	}

	switch e := p.Elements[0].(type) {
	case *SpansetFilter:
		return e
	case Pipeline:
		return findSpansetFilter(e)
	default:
		// SpansetOperation, ScalarFilter, etc. - not a simple spanset filter
		return nil
	}
}
