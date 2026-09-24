package traceql

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// This file implements lenient parsing of incomplete TraceQL queries,
// used by the tag-name and tag-value autocomplete endpoints
// where clients (e.g. Grafana) send partially-typed queries such as `{ .foo = "bar" && name = }`.
//
// Anything else (e.g. a dangling connector like `{ .a = 1 && }`) intentionally returns the original parse error,
// and callers degrade to unfiltered results.
// This is not a general TraceQL repair tool; do not widen the shape.
//
// The approach is deliberately a single string-level transformation:
//
//  1. Tokenize the query with the regular TraceQL lexer.
//  2. Replace each incomplete matcher (attribute + comparison operator with nothing
//     after it, e.g. `name =`; see isTerminator) with the literal `true`.
//  3. Rebuild the query string (balancing unclosed braces) and re-parse it with the strict parser.
//
// Replacing incomplete matchers with `true` — rather than deleting tokens —
// keeps the expression structurally complete and boolean-typed by construction:
// no dangling `&&`/`||` connectors, no empty parentheses, no type errors from leftover bare attributes.
// `true` is also semantically right for condition extraction:
// it contributes no conditions in an AND chain,
// and as an OR branch it correctly collapses the query to match-all
// (see ExtractConditionGroups in lenient_extract.go).

// ParseLenient attempts to parse a query string.
// If parsing succeeds, the result is returned as-is.
// If parsing fails (e.g. due to incomplete matchers like `.foo =`),
// incomplete matchers are replaced with `true` and the cleaned query is re-parsed.
// The original query structure (ORs, ANDs, pipes, structural operators, etc.) is preserved.
func ParseLenient(s string) (*RootExpr, error) {
	expr, err := ParseNoOptimizations(s)
	if err == nil {
		return expr, nil
	}

	// Replace incomplete matchers and try again.
	cleaned := replaceIncompleteMatchers(s)
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

// replaceIncompleteMatchers tokenizes the input, replaces incomplete matchers
// (attribute + comparison operator with no following value) with `true`, and rebuilds the query string.
// Only the part before the first pipe is rewritten — the rewrite logic doesn't understand pipeline syntax
// (function calls like rate(), count(), grouping like by()) and would mangle it.
// The pipeline is re-appended after the rewrite.
func replaceIncompleteMatchers(s string) string {
	tokens := tokenize(s)
	if len(tokens) == 0 {
		return ""
	}

	// Split at the first pipe: rewrite matchers only, preserve pipeline as-is.
	var pipelineTokens []token
	for i, t := range tokens {
		if t.typ == PIPE {
			pipelineTokens = tokens[i:]
			tokens = tokens[:i]
			break
		}
	}

	out := make([]token, 0, len(tokens)+len(pipelineTokens))
	for i := 0; i < len(tokens); {
		if !isAttributeToken(tokens[i].typ) {
			out = append(out, tokens[i])
			i++
			continue
		}

		attrStart := i
		skipAttribute(tokens, &i)

		// Matcher: attr + comparison operator. It is incomplete only when
		// nothing follows the operator: end of input or a structural
		// terminator. Anything else may be the RHS — keep it and let the
		// strict re-parse decide whether it's valid.
		if i+1 < len(tokens) && isComparisonOperator(tokens[i+1].typ) {
			if i+2 >= len(tokens) || isTerminator(tokens[i+2].typ) {
				// Incomplete: no RHS → replace with `true`. tokenRepr renders
				// the TRUE token via reverseTokenMap, so no str is needed.
				out = append(out, token{typ: TRUE})
			} else {
				out = append(out, tokens[attrStart:i+2]...)
			}
			i += 2
			continue
		}

		// Not a matcher pattern (just an attribute reference) → keep the token
		// and rescan from the next one.
		out = append(out, tokens[attrStart])
		i = attrStart + 1
	}

	// Re-append the pipeline tokens (not subject to the rewrite).
	out = append(out, pipelineTokens...)

	return rebuildQuery(out)
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

// rebuildQuery reconstructs a query string from tokens, balancing any unclosed braces.
func rebuildQuery(tokens []token) string {
	var b strings.Builder
	braceDepth := 0
	for i, t := range tokens {
		if i > 0 && !isScopeToken(tokens[i-1].typ) {
			b.WriteString(" ")
		}
		b.WriteString(tokenRepr(t))
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

var reverseTokenMap = func() map[int]string {
	m := make(map[int]string, len(tokenMap))
	for str, tok := range tokenMap {
		m[tok] = str
	}
	return m
}()

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

// isTerminator reports whether a token closes a matcher's slot in a filter:
// a closing brace/paren or a connector. A comparison operator followed by one
// of these (or by end of input) has no RHS, i.e. the matcher is incomplete.
// This is deliberately a closed set: the pass only knows what "nothing" looks
// like, so it needs no updates as the expression grammar grows.
func isTerminator(typ int) bool {
	return typ == CLOSE_BRACE || typ == CLOSE_PARENS || typ == AND || typ == OR
}

// lenientLexer is a lexer that doesn't stop on errors.
type lenientLexer struct {
	lexer
}
