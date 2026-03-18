package traceql

import (
	"strings"
)

const emptyQuery = "{}"

// ExtractFetchRequest extracts filter conditions from a query string.
// It parses the query using the lenient parser (which handles incomplete matchers like `.foo=`)
// and walks the AST to extract conditions. Conditions with OpNone (from incomplete matchers) are filtered out.
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

	// Walk the full AST to extract conditions. AllConditions is set to false
	// by extractConditions when OR operators or structural operators are present.
	req := &FetchSpansRequest{AllConditions: true}
	expr.extractConditions(req)

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

// CanonicalQuery parses a query string using the lenient parser and returns
// a normalized string representation. Used for cache key generation.
func CanonicalQuery(query string) string {
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
