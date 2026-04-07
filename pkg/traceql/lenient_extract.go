package traceql

import (
	"strings"
)

const emptyQuery = "{}"

// ExtractFetchRequest parses a query string using the lenient parser and returns
// a FetchSpansRequest with the extracted conditions. Conditions with OpNone
// (from incomplete matchers) are filtered out.
//
// Returns nil if the query is empty or parsing fails completely.
//
// The caller should check AllConditions to determine whether the conditions
// can be used as filters (true = all AND, false = contains OR or structural operators).
func ExtractFetchRequest(query string) *FetchSpansRequest {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil
	}
	if err := expr.validate(); err != nil {
		return nil
	}

	// Walk the full AST to extract conditions. AllConditions is set to false
	// by extractConditions when OR operators or structural operators are present.
	requests := expr.extractConditions(FetchSpansRequest{AllConditions: true})

	// Use the first sub-query request.
	var req FetchSpansRequest
	for _, v := range requests {
		req = v
		break
	}

	// Filter out OpNone conditions — these are column-fetch hints (bare attributes,
	// structural intrinsics), not filterable conditions.
	conditions := make([]Condition, 0, len(req.Conditions))
	for _, cond := range req.Conditions {
		if cond.Op != OpNone {
			conditions = append(conditions, cond)
		}
	}
	req.Conditions = conditions

	return &req
}

// NormalizeQuery parses a query string using the lenient parser and returns
// a normalized string representation. Used for cache key generation.
// Match-all queries (e.g. "{}", "{ }", "{ true }") are normalized to "{}"
// to avoid cache fragmentation.
func NormalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return emptyQuery
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return emptyQuery
	}

	if err := expr.validate(); err != nil {
		return emptyQuery
	}

	s := expr.String()
	if s == "{ true }" {
		return emptyQuery
	}
	return s
}
