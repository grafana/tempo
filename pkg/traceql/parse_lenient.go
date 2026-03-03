package traceql

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// ParseLenient attempts to parse a query string. If parsing succeeds, the result
// is returned as-is. If parsing fails (e.g. due to incomplete matchers like `.foo=`),
// it removes incomplete matchers from the token stream while preserving the original
// query structure (ORs, ANDs, pipes, structural operators, etc.) and parses the
// cleaned-up query.
func ParseLenient(s string) (*RootExpr, error) {
	expr, err := Parse(s)
	if err == nil {
		return expr, nil
	}

	// Remove incomplete matchers and try again.
	cleaned := removeIncompleteMatchers(s)
	if cleaned == "" {
		return nil, err
	}

	result, cleanedErr := Parse(cleaned)
	if cleanedErr != nil {
		return nil, err // return original parse error
	}
	return result, nil
}

// token represents a lexed token with its type and string value.
type token struct {
	typ int
	str string
}

// removeIncompleteMatchers tokenizes the input, removes incomplete matchers
// (attribute + comparison operator with no following value), cleans up dangling
// connectors, and rebuilds the query string.
func removeIncompleteMatchers(s string) string {
	tokens := tokenize(s)
	if len(tokens) == 0 {
		return ""
	}

	remove := make([]bool, len(tokens))
	markIncompleteMatchers(tokens, remove)
	cleanDanglingConnectors(tokens, remove)

	return rebuildQuery(tokens, remove)
}

// tokenize lexes the input into tokens, skipping END_ATTRIBUTE markers.
func tokenize(s string) []token {
	l := lenientLexer{}
	l.Init(strings.NewReader(s))
	l.Scanner.Error = func(*scanner.Scanner, string) {} // Suppress scanner errors

	var tokens []token
	for {
		var lval yySymType
		tok := l.Lex(&lval)
		if tok == 0 {
			break
		}
		if tok == END_ATTRIBUTE {
			continue
		}

		t := token{typ: tok}
		switch tok {
		case STRING, IDENTIFIER:
			t.str = lval.staticStr
		case INTEGER:
			t.str = fmt.Sprintf("%d", lval.staticInt)
		case FLOAT:
			t.str = fmt.Sprintf("%f", lval.staticFloat)
		case DURATION:
			t.str = lval.staticDuration.String()
		case TRUE:
			t.str = "true"
		case FALSE:
			t.str = "false"
		default:
			t.str = l.TokenText()
		}
		tokens = append(tokens, t)
	}
	return tokens
}

// markIncompleteMatchers finds attribute + comparison operator sequences with no
// following value token and marks them for removal.
func markIncompleteMatchers(tokens []token, remove []bool) {
	i := 0
	for i < len(tokens) {
		if !isAttributeToken(tokens[i].typ) {
			i++
			continue
		}

		attrStart := i
		skipAttribute(tokens, &i)

		// Complete matcher: attr + op + value → keep
		if i+2 < len(tokens) && isComparisonOperator(tokens[i+1].typ) && isValueToken(tokens[i+2].typ) {
			i += 3
			continue
		}

		// Incomplete matcher: attr + op but no value → mark for removal
		if i+1 < len(tokens) && isComparisonOperator(tokens[i+1].typ) {
			for j := attrStart; j <= i+1; j++ {
				remove[j] = true
			}
			i += 2
			continue
		}

		// Not a matcher pattern (just an attribute reference) → leave it
		i = attrStart + 1
	}
}

// skipAttribute advances idx past attribute tokens (scope prefix + name),
// mirroring the logic of buildAttributeString without building a string.
func skipAttribute(tokens []token, idx *int) {
	i := *idx
	switch tokens[i].typ {
	case PARENT_DOT:
		i++
		if i < len(tokens) && (tokens[i].typ == SPAN_DOT || tokens[i].typ == RESOURCE_DOT) {
			i++
		}
	case RESOURCE_DOT, SPAN_DOT, EVENT_DOT, EVENT_COLON, LINK_DOT, LINK_COLON,
		TRACE_COLON, SPAN_COLON, INSTRUMENTATION_DOT, INSTRUMENTATION_COLON:
		i++
	case DOT:
		i++
	}
	// i now points to the attribute name (IDENTIFIER or intrinsic)
	*idx = i
}

// cleanDanglingConnectors removes AND/OR tokens left dangling after incomplete
// matcher removal (e.g. adjacent to braces or other connectors).
func cleanDanglingConnectors(tokens []token, remove []bool) {
	changed := true
	for changed {
		changed = false
		for i := range tokens {
			if remove[i] || !isConnector(tokens[i].typ) {
				continue
			}
			prev := findAdjacentToken(tokens, remove, i, -1)
			next := findAdjacentToken(tokens, remove, i, 1)

			if prev == -1 || tokens[prev].typ == OPEN_BRACE ||
				next == -1 || tokens[next].typ == CLOSE_BRACE ||
				isConnector(tokens[prev].typ) || isConnector(tokens[next].typ) {
				remove[i] = true
				changed = true
			}
		}
	}
}

func isConnector(typ int) bool {
	return typ == AND || typ == OR
}

// findAdjacentToken returns the index of the nearest non-removed token in the
// given direction (-1 for previous, +1 for next). Returns -1 if none found.
func findAdjacentToken(tokens []token, remove []bool, from, dir int) int {
	for j := from + dir; j >= 0 && j < len(tokens); j += dir {
		if !remove[j] {
			return j
		}
	}
	return -1
}

// rebuildQuery reconstructs a query string from remaining (non-removed) tokens,
// balancing any unclosed braces.
func rebuildQuery(tokens []token, remove []bool) string {
	var b strings.Builder
	prevIdx := -1
	braceDepth := 0
	for i, t := range tokens {
		if remove[i] {
			continue
		}
		if prevIdx >= 0 && !isScopeToken(tokens[prevIdx].typ) {
			b.WriteString(" ")
		}
		b.WriteString(tokenRepr(t))
		prevIdx = i
		switch t.typ {
		case OPEN_BRACE:
			braceDepth++
		case CLOSE_BRACE:
			braceDepth--
		}
	}
	// Close any unclosed braces (e.g. input was missing closing `}`)
	for braceDepth > 0 {
		b.WriteString(" }")
		braceDepth--
	}
	return b.String()
}

func isScopeToken(typ int) bool {
	return typ == DOT || typ == SPAN_DOT || typ == RESOURCE_DOT ||
		typ == EVENT_DOT || typ == LINK_DOT || typ == INSTRUMENTATION_DOT ||
		typ == PARENT_DOT || typ == EVENT_COLON || typ == LINK_COLON ||
		typ == TRACE_COLON || typ == SPAN_COLON || typ == INSTRUMENTATION_COLON
}

// tokenRepr returns the string representation of a token for query rebuilding.
// Explicit mappings are needed for multi-character tokens where scanner.TokenText()
// only returns the last scanned character (e.g. "&&" → "&").
func tokenRepr(t token) string {
	switch t.typ {
	case STRING:
		return strconv.Quote(t.str)
	// Connectors
	case AND:
		return "&&"
	case OR:
		return "||"
	// Comparison operators
	case EQ:
		return "="
	case NEQ:
		return "!="
	case LT:
		return "<"
	case LTE:
		return "<="
	case GT:
		return ">"
	case GTE:
		return ">="
	case RE:
		return "=~"
	case NRE:
		return "!~"
	// Structural
	case OPEN_BRACE:
		return "{"
	case CLOSE_BRACE:
		return "}"
	case OPEN_PARENS:
		return "("
	case CLOSE_PARENS:
		return ")"
	case PIPE:
		return "|"
	// Scope tokens
	case DOT:
		return "."
	case SPAN_DOT:
		return "span."
	case RESOURCE_DOT:
		return "resource."
	case EVENT_DOT:
		return "event."
	case LINK_DOT:
		return "link."
	case INSTRUMENTATION_DOT:
		return "instrumentation."
	case PARENT_DOT:
		return "parent."
	case EVENT_COLON:
		return "event:"
	case LINK_COLON:
		return "link:"
	case TRACE_COLON:
		return "trace:"
	case SPAN_COLON:
		return "span:"
	case INSTRUMENTATION_COLON:
		return "instrumentation:"
	// Arithmetic
	case ADD:
		return "+"
	case SUB:
		return "-"
	case MUL:
		return "*"
	case DIV:
		return "/"
	case MOD:
		return "%"
	case POW:
		return "^"
	case NOT:
		return "!"
	// Spanset structural operators
	case DESC:
		return ">>"
	case ANCE:
		return "<<"
	case SIBL:
		return "~"
	case NOT_CHILD:
		return "!>"
	case NOT_PARENT:
		return "!<"
	case NOT_DESC:
		return "!>>"
	case NOT_ANCE:
		return "!<<"
	case UNION_CHILD:
		return "&>"
	case UNION_PARENT:
		return "&<"
	case UNION_DESC:
		return "&>>"
	case UNION_ANCE:
		return "&<<"
	case UNION_SIBL:
		return "&~"
	default:
		return t.str
	}
}

func isAttributeToken(typ int) bool {
	return typ == DOT || typ == IDENTIFIER || isIntrinsicToken(typ) ||
		typ == PARENT_DOT || typ == RESOURCE_DOT || typ == SPAN_DOT ||
		typ == EVENT_DOT || typ == LINK_DOT || typ == INSTRUMENTATION_DOT ||
		typ == EVENT_COLON || typ == LINK_COLON || typ == TRACE_COLON ||
		typ == SPAN_COLON || typ == INSTRUMENTATION_COLON
}

func isIntrinsicToken(typ int) bool {
	return typ == IDURATION || typ == CHILDCOUNT || typ == NAME ||
		typ == STATUS || typ == STATUS_MESSAGE || typ == KIND ||
		typ == ROOTNAME || typ == ROOTSERVICENAME || typ == ROOTSERVICE ||
		typ == TRACEDURATION || typ == NESTEDSETLEFT || typ == NESTEDSETRIGHT ||
		typ == NESTEDSETPARENT || typ == ID || typ == TRACE_ID ||
		typ == SPAN_ID || typ == PARENT_ID || typ == TIMESINCESTART ||
		typ == VERSION || typ == PARENT
}

func isComparisonOperator(typ int) bool {
	return typ == EQ || typ == NEQ || typ == LT || typ == LTE ||
		typ == GT || typ == GTE || typ == RE || typ == NRE
}

func isValueToken(typ int) bool {
	return typ == STRING || typ == INTEGER || typ == FLOAT ||
		typ == DURATION || typ == TRUE || typ == FALSE || typ == NIL ||
		typ == STATUS_OK || typ == STATUS_ERROR || typ == STATUS_UNSET ||
		typ == KIND_UNSPECIFIED || typ == KIND_INTERNAL || typ == KIND_SERVER ||
		typ == KIND_CLIENT || typ == KIND_PRODUCER || typ == KIND_CONSUMER ||
		typ == IDENTIFIER
}

// lenientLexer is a lexer that doesn't stop on errors.
type lenientLexer struct {
	lexer
}
