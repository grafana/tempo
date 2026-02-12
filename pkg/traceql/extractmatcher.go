package traceql

import (
	"strconv"
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
func ExtractConditions(query string) []Condition {
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

	if !req.AllConditions || len(req.Conditions) == 0 {
		return nil
	}

	// Filter out OpNone conditions (incomplete matchers)
	conditions := make([]Condition, 0, len(req.Conditions))
	for _, cond := range req.Conditions {
		if cond.Op != OpNone {
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return conditions
}

// ExtractMatchers extracts matchers from a query string and returns a string
// that can be parsed by the storage layer. It uses ExtractConditions internally.
func ExtractMatchers(query string) string {
	conditions := ExtractConditions(query)
	if len(conditions) == 0 {
		return emptyQuery
	}

	var q strings.Builder
	q.WriteString("{")

	for i, cond := range conditions {
		if i > 0 {
			q.WriteString(" && ")
		}

		q.WriteString(formatAttribute(cond.Attribute))
		q.WriteString(" ")
		q.WriteString(cond.Op.String())
		q.WriteString(" ")

		if len(cond.Operands) > 0 {
			q.WriteString(formatOperand(cond.Operands[0]))
		}
	}

	q.WriteString("}")

	return q.String()
}

// formatAttribute returns the string representation of an attribute,
// quoting the name if it contains special characters.
func formatAttribute(attr Attribute) string {
	scopes := []string{}
	if attr.Parent {
		scopes = append(scopes, "parent")
	}

	if attr.Scope != AttributeScopeNone {
		scopes = append(scopes, attr.Scope.String())
	}

	att := attr.Name
	if attr.Intrinsic != IntrinsicNone {
		att = attr.Intrinsic.String()
	} else if needsQuoting(att) {
		att = strconv.Quote(att)
	}

	scope := ""
	if len(scopes) > 0 {
		scope = strings.Join(scopes, ".") + "."
	}

	// Top-level attributes get a "." but top-level intrinsics don't
	if scope == "" && attr.Intrinsic == IntrinsicNone && len(att) > 0 {
		scope += "."
	}

	return scope + att
}

// needsQuoting returns true if an attribute name contains characters
// that require quoting (spaces or special characters).
func needsQuoting(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, r := range name {
		if !isAttributeRune(r) {
			return true
		}
	}
	return false
}

// formatOperand returns the string representation of an operand,
// using double quotes for string values instead of backticks.
func formatOperand(operand Static) string {
	if operand.Type == TypeString {
		s := operand.EncodeToString(false)
		return strconv.Quote(s)
	}
	return operand.String()
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
