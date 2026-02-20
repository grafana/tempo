package traceql

import (
	"strings"
	"unicode"
)

const emptyQuery = "{}"

// ExtractMatchers extracts complete matchers from a potentially incomplete
// TraceQL query string and returns a valid query that the storage layer can
// parse. Incomplete matchers (e.g., ".bar =") are silently dropped.
//
// This function uses token-based parsing instead of regex to robustly handle
// edge cases like whitespace in quoted values, special characters in attribute
// names, and incomplete expressions.
//
// Only single spanset filter queries are supported. Queries with OR conditions,
// multiple spanset filters, or pipeline operators return an empty query.
func ExtractMatchers(query string) string {
	query = strings.TrimSpace(query)

	if len(query) == 0 {
		return emptyQuery
	}

	// Reject queries with multiple spanset filters or OR/pipe operators
	if !isSingleFilter(query) {
		return emptyQuery
	}

	matchers := extractCompleteMatchers(query)
	if len(matchers) == 0 {
		return emptyQuery
	}

	var q strings.Builder
	q.WriteString("{")
	for i, m := range matchers {
		if i > 0 {
			q.WriteString(" && ")
		}
		q.WriteString(m)
	}
	q.WriteString("}")

	return q.String()
}

// isSingleFilter checks that the query represents a single spanset filter
// without OR conditions or multiple spanset groups.
func isSingleFilter(query string) bool {
	braceDepth := 0
	braceCount := 0
	inString := false
	escaped := false

	for i, r := range query {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch r {
		case '{':
			braceDepth++
			braceCount++
			if braceCount > 1 {
				return false // multiple spanset filters
			}
		case '}':
			braceDepth--
		case '|':
			// Check for || (OR) or | (pipe) - both disqualify
			if braceDepth > 0 {
				// Inside braces: || is OR
				if i+1 < len(query) && query[i+1] == '|' {
					return false
				}
				// For =~ and !~ operators, the | inside quotes is handled above
				// For regex patterns inside a single filter, we need to allow |
				// inside a regex value. Check if we're after =~ or !~
				if !isPipeInRegex(query, i) {
					// Could be a | inside a regex value, let it pass if within quotes
					// or it's a bare pipe which disqualifies
				}
			} else {
				// Outside braces: pipe operator
				return false
			}
		}
	}

	return braceCount <= 1
}

// isPipeInRegex checks if the pipe character at position i is within a regex
// pattern (after =~ or !~ operator inside a quoted string).
func isPipeInRegex(query string, i int) bool {
	// Look backwards for an unclosed quote
	quoteCount := 0
	for j := i - 1; j >= 0; j-- {
		if query[j] == '"' && (j == 0 || query[j-1] != '\\') {
			quoteCount++
		}
	}
	// If odd number of quotes before us, we're inside a string
	return quoteCount%2 == 1
}

// extractCompleteMatchers uses a token-based approach to extract complete
// matchers (attribute operator value) from a query string.
func extractCompleteMatchers(query string) []string {
	// Strip the outer braces if present
	inner := query
	if idx := strings.IndexByte(query, '{'); idx >= 0 {
		inner = query[idx+1:]
	}
	if idx := strings.LastIndexByte(inner, '}'); idx >= 0 {
		inner = inner[:idx]
	}

	// Split on && to get individual matcher candidates
	parts := splitOnAnd(inner)

	var matchers []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if m, ok := parseCompleteMatcher(part); ok {
			matchers = append(matchers, m)
		}
	}
	return matchers
}

// splitOnAnd splits a string on "&&" while respecting quoted strings.
func splitOnAnd(s string) []string {
	var parts []string
	var current strings.Builder
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		if escaped {
			escaped = false
			current.WriteByte(s[i])
			continue
		}
		if s[i] == '\\' {
			escaped = true
			current.WriteByte(s[i])
			continue
		}
		if s[i] == '"' {
			inString = !inString
			current.WriteByte(s[i])
			continue
		}
		if !inString && i+1 < len(s) && s[i] == '&' && s[i+1] == '&' {
			parts = append(parts, current.String())
			current.Reset()
			i++ // skip second &
			continue
		}
		current.WriteByte(s[i])
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// parseCompleteMatcher attempts to parse a complete matcher expression:
// attribute operator value. Returns the normalized matcher string and true
// if successful, or empty string and false if the matcher is incomplete.
func parseCompleteMatcher(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", false
	}

	// Phase 1: Extract the attribute name (LHS)
	lhs := extractLHS(expr)
	if lhs == "" {
		return "", false
	}

	// Find where the operator starts (after the LHS)
	remaining := strings.TrimSpace(expr[len(lhs):])
	lhs = strings.TrimSpace(lhs)

	// Phase 2: Extract the operator
	op, opLen := extractOperator(remaining)
	if op == "" {
		return "", false
	}

	// Phase 3: Extract the value (RHS)
	rhs := strings.TrimSpace(remaining[opLen:])
	if rhs == "" {
		return "", false
	}

	// Validate the value is complete
	value, ok := extractValue(rhs)
	if !ok || value == "" {
		return "", false
	}

	return lhs + " " + op + " " + value, true
}

// extractLHS extracts the left-hand side (attribute name) of a matcher.
func extractLHS(expr string) string {
	// The LHS can be:
	// - A scoped attribute: span.foo, resource.service.name, event:name
	// - An unscoped attribute: .foo, .http.status_code
	// - A quoted attribute: span."foo bar"
	// - An intrinsic: duration, name, status, kind, etc.
	var result strings.Builder
	i := 0
	inQuote := false

	for i < len(expr) {
		ch := expr[i]
		if ch == '"' {
			inQuote = !inQuote
			result.WriteByte(ch)
			i++
			continue
		}
		if inQuote {
			result.WriteByte(ch)
			i++
			continue
		}
		// Valid attribute characters
		if isAttributeChar(rune(ch)) {
			result.WriteByte(ch)
			i++
			continue
		}
		// Stop at whitespace or operator characters
		break
	}
	return result.String()
}

func isAttributeChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '.' || ch == '_' || ch == '-' || ch == ':'
}

// extractOperator extracts a comparison operator from the beginning of a string.
func extractOperator(s string) (string, int) {
	operators := []string{"=~", "!~", "!=", ">=", "<=", "=", ">", "<"}
	for _, op := range operators {
		if strings.HasPrefix(s, op) {
			return op, len(op)
		}
	}
	return "", 0
}

// extractValue extracts and validates a complete value from the RHS of a matcher.
func extractValue(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}

	// Quoted string value
	if s[0] == '"' {
		// Find the closing quote (handle escaped quotes)
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++ // skip escaped char
				continue
			}
			if s[i] == '"' {
				return s[:i+1], true
			}
		}
		return "", false // unclosed quote
	}

	// Unquoted value: number, duration, boolean, status, kind
	var val strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '.' || ch == '-' || ch == '+' {
			val.WriteByte(ch)
		} else {
			break
		}
	}
	result := val.String()
	if result == "" {
		return "", false
	}

	// Validate it looks like a valid value
	if isValidUnquotedValue(result) {
		return result, true
	}
	return "", false
}

// isValidUnquotedValue checks if a string is a valid unquoted TraceQL value.
func isValidUnquotedValue(s string) bool {
	// Boolean
	if s == "true" || s == "false" {
		return true
	}
	// Nil
	if s == "nil" {
		return true
	}
	// Status values
	if s == "ok" || s == "error" || s == "unset" {
		return true
	}
	// Kind values
	if s == "unspecified" || s == "internal" || s == "server" || s == "client" || s == "producer" || s == "consumer" {
		return true
	}
	// Number (int or float) possibly with duration suffix
	if len(s) > 0 && (s[0] >= '0' && s[0] <= '9' || s[0] == '-' || s[0] == '+') {
		return true
	}
	return false
}

func IsEmptyQuery(query string) bool {
	return query == emptyQuery || len(query) == 0
}
