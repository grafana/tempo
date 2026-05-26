package traceql

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// ParseLenient attempts to parse a query string. If parsing succeeds, the result
// is returned as-is. If parsing fails (e.g. due to incomplete matchers like `.foo=`),
// it removes the comparison operator from incomplete matchers, leaving just the
// attribute (e.g. `.foo`), while preserving the original query structure (ORs, ANDs,
// pipes, structural operators, etc.) and re-parses the cleaned-up query.
func ParseLenient(s string) (*RootExpr, error) {
	expr, err := ParseNoOptimizations(s)
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
	removeBareAttributes(result)
	return result, nil
}

// removeBareAttributes walks the AST and replaces bare Attribute expressions
// inside SpansetFilters with `true`. These are leftovers from incomplete matchers
// (e.g. { event:name = } cleaned to { event:name }) that wouldn't pass validation.
func removeBareAttributes(root *RootExpr) {
	removeBareAttrsInPipeline(root.Pipeline)
}

func removeBareAttrsInPipeline(p Pipeline) {
	for _, e := range p.Elements {
		switch v := e.(type) {
		case *SpansetFilter:
			if _, ok := v.Expression.(Attribute); ok {
				v.Expression = NewStaticBool(true)
			}
		case SpansetOperation:
			removeBareAttrsInElement(v.LHS)
			removeBareAttrsInElement(v.RHS)
		}
	}
}

func removeBareAttrsInElement(e SpansetExpression) {
	switch v := e.(type) {
	case *SpansetFilter:
		if _, ok := v.Expression.(Attribute); ok {
			v.Expression = NewStaticBool(true)
		}
	case SpansetOperation:
		removeBareAttrsInElement(v.LHS)
		removeBareAttrsInElement(v.RHS)
	case Pipeline:
		removeBareAttrsInPipeline(v)
	}
}

// token represents a lexed token with its type and string value.
type token struct {
	typ int
	str string
}

// removeIncompleteMatchers tokenizes the input, removes incomplete matchers
// (attribute + comparison operator with no following value), cleans up dangling
// connectors, and rebuilds the query string.
// Only the part before the first pipe is cleaned — the cleanup logic doesn't
// understand pipeline syntax (function calls like rate(), count(), grouping
// like by()) and would mangle it. The pipeline is re-appended after cleanup.
func removeIncompleteMatchers(s string) string {
	tokens := tokenize(s)
	if len(tokens) == 0 {
		return ""
	}

	// Split at the first pipe: clean matchers only, preserve pipeline as-is.
	var pipelineTokens []token
	pipeIdx := -1
	for i, t := range tokens {
		if t.typ == PIPE {
			pipeIdx = i
			break
		}
	}

	if pipeIdx != -1 {
		pipelineTokens = tokens[pipeIdx:]
		tokens = tokens[:pipeIdx]
	}

	remove := make([]bool, len(tokens))
	markIncompleteMatchers(tokens, remove)
	cleanDanglingConnectorsAndParens(tokens, remove)

	// Re-append the pipeline tokens (not subject to cleanup).
	tokens = append(tokens, pipelineTokens...)
	remove = append(remove, make([]bool, len(pipelineTokens))...)

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
			t.str = strconv.FormatFloat(lval.staticFloat, 'g', -1, 64)
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

		// Incomplete matcher: attr + op but no value → mark operator for removal
		if i+1 < len(tokens) && isComparisonOperator(tokens[i+1].typ) {
			remove[i+1] = true
			i += 2
			continue
		}

		// Not a matcher pattern (just an attribute reference) → leave it
		i = attrStart + 1
	}
}

// skipAttribute advances idx past scope prefix tokens so it points to the
// attribute name (IDENTIFIER or intrinsic). For bare intrinsics (e.g. statusMessage),
// idx is not advanced since the intrinsic token is the attribute itself.
// If the scope prefix is the last token (e.g. a trailing "."), *idx may equal
// len(tokens) after this call; callers must check bounds before accessing tokens[*idx].
func skipAttribute(tokens []token, idx *int) {
	i := *idx
	switch tokens[i].typ {
	case PARENT_DOT:
		i++
		if i < len(tokens) && (tokens[i].typ == SPAN_DOT || tokens[i].typ == RESOURCE_DOT) {
			i++
		}
	case DOT, SPAN_DOT, RESOURCE_DOT, EVENT_DOT, LINK_DOT, INSTRUMENTATION_DOT:
		i++
	case EVENT_COLON, LINK_COLON, TRACE_COLON, SPAN_COLON, INSTRUMENTATION_COLON:
		i++
	default:
		// Bare intrinsic (statusMessage, duration, etc.) — already the attribute name.
	}
	*idx = i
}

// cleanDanglingConnectorsAndParens removes AND/OR tokens left dangling after
// incomplete matcher removal (e.g. adjacent to braces or other connectors),
// and removes parentheses pairs that contain only removed tokens.
func cleanDanglingConnectorsAndParens(tokens []token, remove []bool) {
	changed := true
	for changed {
		changed = false
		for i := range tokens {
			if remove[i] {
				continue
			}

			switch tokens[i].typ {
			case AND, OR:
				// Remove connectors with no valid expression on either side
				prev := findAdjacentToken(tokens, remove, i, -1)
				next := findAdjacentToken(tokens, remove, i, 1)

				if prev == -1 || tokens[prev].typ == OPEN_BRACE || tokens[prev].typ == OPEN_PARENS ||
					next == -1 || tokens[next].typ == CLOSE_BRACE || tokens[next].typ == CLOSE_PARENS ||
					isConnector(tokens[prev].typ) || isConnector(tokens[next].typ) {
					remove[i] = true
					changed = true
				}
			case OPEN_PARENS:
				// check if all tokens inside the parens are removed, if so remove the parens too
				closeParensIdx := findNextCloseParens(tokens, remove, i+1)
				if closeParensIdx != -1 {
					allRemoved := true
					for j := i + 1; j < closeParensIdx; j++ {
						if !remove[j] {
							allRemoved = false
							break
						}
					}
					if allRemoved {
						remove[i] = true
						remove[closeParensIdx] = true
						changed = true
					}
				}
			default:
				continue
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

func findNextCloseParens(tokens []token, remove []bool, from int) int {
	depth := 0
	for j := from; j < len(tokens); j++ {
		if remove[j] {
			continue
		}
		switch tokens[j].typ {
		case OPEN_PARENS:
			depth++
		case CLOSE_PARENS:
			if depth == 0 {
				return j
			}
			depth--
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
	return startsAttribute(typ) ||
		typ == EVENT_COLON || typ == LINK_COLON || typ == TRACE_COLON ||
		typ == SPAN_COLON || typ == INSTRUMENTATION_COLON
}

var reverseTokenMap = (func() map[int]string {
	m := make(map[int]string, len(tokenMap))
	for str, tok := range tokenMap {
		m[tok] = str
	}
	return m
})()

// tokenRepr returns the string representation of a token for query rebuilding.
// Explicit mappings are needed for multi-character tokens where scanner.TokenText()
// only returns the last scanned character (e.g. "&&" → "&").
// IDENTIFIER tokens that contain non-attribute runes (e.g. spaces) are re-quoted
// so the output round-trips through the parser correctly (e.g. span."foo bar").
func tokenRepr(t token) string {
	if str, ok := reverseTokenMap[t.typ]; ok {
		return str
	}
	if t.typ == STRING || (t.typ == IDENTIFIER && ContainsNonAttributeRune(t.str)) {
		return strconv.Quote(t.str)
	}
	return t.str
}

func isAttributeToken(typ int) bool {
	return typ == IDENTIFIER || isIntrinsicToken(typ) || isScopeToken(typ)
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
