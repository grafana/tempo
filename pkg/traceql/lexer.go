package traceql

import (
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"
)

var tokens = map[string]int{
	".":          DOT,
	"{":          OPEN_BRACE,
	"}":          CLOSE_BRACE,
	"(":          OPEN_PARENS,
	")":          CLOSE_PARENS,
	"=":          EQ,
	"!=":         NEQ,
	"=~":         RE,
	"!~":         NRE,
	">":          GT,
	">=":         GTE,
	"<":          LT,
	"<=":         LTE,
	"+":          ADD,
	"-":          SUB,
	"/":          DIV,
	"%":          MOD,
	"*":          MUL,
	"^":          POW,
	"true":       TRUE,
	"false":      FALSE,
	"nil":        NIL,
	"ok":         STATUS_OK,
	"error":      STATUS_ERROR,
	"unset":      STATUS_UNSET,
	"&&":         AND,
	"||":         OR,
	"!":          NOT,
	"|":          PIPE,
	">>":         DESC,
	"~":          TILDE,
	"start":      ISTART,
	"end":        IEND,
	"duration":   IDURATION,
	"childCount": ICHILDCOUNT,
	"name":       INAME,
	"status":     ISTATUS,
	"parent":     APARENT,
	"resource":   ARESOURCE,
	"span":       ASPAN,
	"count":      COUNT,
	"avg":        AVG,
	"max":        MAX,
	"min":        MIN,
	"sum":        SUM,
	"by":         BY,
	"coalesce":   COALESCE,
}

type lexer struct {
	scanner.Scanner
	expr   *RootExpr
	parser *yyParserImpl
	errs   []ParseError

	prevToken int
}

func (l *lexer) Lex(lval *yySymType) int {
	r := l.Scan()
	switch r {
	case scanner.EOF:
		return 0

	case scanner.String, scanner.RawString:
		var err error
		lval.staticStr, err = strconv.Unquote(l.TokenText())
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		return STRING

	case scanner.Int:
		numberText := l.TokenText()

		// first try to parse as duration
		duration, ok := tryScanDuration(numberText, &l.Scanner)
		if ok {
			lval.staticDuration = duration
			return DURATION
		}

		// if we can't then just try an int
		var err error
		lval.staticInt, err = strconv.Atoi(numberText)
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		return INTEGER

	case scanner.Float:
		var err error
		lval.staticFloat, err = strconv.ParseFloat(l.TokenText(), 64)
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		return FLOAT
	}

	// if the previous token was a dot we will always consider the current token an IDENTIFIER.
	//  this is b/c DOT is always used in attribute selection like { .status }
	if l.prevToken == DOT {
		l.prevToken = -1
		lval.staticStr = l.TokenText()
		return IDENTIFIER
	}

	if tok, ok := tokens[l.TokenText()+string(l.Peek())]; ok {
		l.Next()
		return tok
	}

	if tok, ok := tokens[l.TokenText()]; ok {
		l.prevToken = tok // save the previous token for the above logic regarding identifiers
		return tok
	}

	lval.staticStr = l.TokenText()
	return IDENTIFIER
}

func (l *lexer) Error(msg string) {
	l.errs = append(l.errs, newParseError(msg, l.Line, l.Column))
}

// ***************************
// Donated with Love from Loki jpe - apache
// ***************************
func tryScanDuration(number string, l *scanner.Scanner) (time.Duration, bool) {
	var sb strings.Builder
	sb.WriteString(number)
	//copy the scanner to avoid advancing it in case it's not a duration.
	s := *l
	consumed := 0
	for r := s.Peek(); r != scanner.EOF && !unicode.IsSpace(r); r = s.Peek() {
		if !unicode.IsNumber(r) && !isDurationRune(r) && r != '.' {
			break
		}
		_, _ = sb.WriteRune(r)
		_ = s.Next()
		consumed++
	}

	if consumed == 0 {
		return 0, false
	}
	// we've found more characters before a whitespace or the end
	d, err := time.ParseDuration(sb.String())
	if err != nil {
		return 0, false
	}
	// we need to consume the scanner, now that we know this is a duration.
	for i := 0; i < consumed; i++ {
		_ = l.Next()
	}
	return d, true
}

func isDurationRune(r rune) bool {
	// "ns", "us" (or "µs"), "ms", "s", "m", "h".
	switch r {
	case 'n', 's', 'u', 'm', 'h', 'µ':
		return true
	default:
		return false
	}
}
