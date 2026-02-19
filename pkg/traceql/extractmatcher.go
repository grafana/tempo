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
//   - The conditions use OR (AllConditions is false)
//   - No valid matchers can be extracted
func ExtractConditions(query string) ([]Condition, *SpansetFilter) {
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return nil, nil
	}

	expr, err := ParseLenient(query)
	if err != nil {
		return nil, nil
	}

	// Find the first SpansetFilter in the pipeline.
	// Returns nil for structural operators (SpansetOperation) indicating multiple spansets.
	filter := findSpansetFilter(expr.Pipeline)
	if filter == nil {
		return nil, nil
	}

	// Extract conditions from the AST. AllConditions is set to false by
	// extractConditions when OR operators are present.
	req := &FetchSpansRequest{AllConditions: true}
	filter.Expression.extractConditions(req)

	if !req.AllConditions || len(req.Conditions) == 0 {
		return nil, nil
	}

	// Filter out OpNone conditions (incomplete matchers)
	conditions := make([]Condition, 0, len(req.Conditions))
	for _, cond := range req.Conditions {
		if cond.Op != OpNone {
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return nil, nil
	}

	return conditions, filter
}

// ExtractMatchers extracts matchers from a query string and returns a string
// that can be parsed by the storage layer. It uses ExtractConditions internally.
func ExtractMatchers(query string) string {
	_, filter := ExtractConditions(query)
	if filter == nil {
		return emptyQuery
	}

	return filter.String()
}

func RemoveUnnecessaryParentheses(query string) string {
	findNextCloseParens := func(s string, start int) int {
		depth := 0
		for i := start; i < len(s); i++ {
			switch s[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					return i
				}
				depth--
			}
		}
		return -1
	}
	for char := 0; char < len(query); char++ {
		if query[char] == '(' {
			closeParensIdx := findNextCloseParens(query, char+1)
			if closeParensIdx != -1 && closeParensIdx != char+1 {
				// Check if the parentheses are around a simple matcher (e.g., (.foo = "bar"))
				inside := query[char+1 : closeParensIdx]
				if !strings.Contains(inside, "&&") && !strings.Contains(inside, "||") {
					if !strings.Contains(inside, "=") && !strings.Contains(inside, ">") &&
						!strings.Contains(inside, "<") && !strings.Contains(inside, "!") &&
						!strings.Contains(inside, "~") {
						continue
					}
					// Remove the parentheses
					query = query[:char] + inside + query[closeParensIdx+1:]
					// Move back the index to account for removed parentheses
					char -= 1
				}
			}
		}
	}
	return query
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
