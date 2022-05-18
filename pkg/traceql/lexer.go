package traceql

import (
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"
)

var tokens = map[string]int{
	".":     DOT,
	"{":     OPEN_BRACE,
	"}":     CLOSE_BRACE,
	"(":     OPEN_PARENS,
	")":     CLOSE_PARENS,
	"=":     EQ,
	"!=":    NEQ,
	"=~":    RE,
	"!~":    NRE,
	">":     GT,
	">=":    GTE,
	"<":     LT,
	"<=":    LTE,
	"+":     ADD,
	"-":     SUB,
	"/":     DIV,
	"%":     MOD,
	"*":     MUL,
	"^":     POW,
	"true":  TRUE,
	"false": FALSE,
	"nil":   NIL,
	"ok":    STATUS_OK,
	"error": STATUS_ERROR,
	"unset": STATUS_UNSET,
	"&&":    AND,
	"||":    OR,
	"!":     NOT,
	"|":     PIPE,
	">>":    DESC,
	"~":     TILDE,
}

var functionTokens = map[string]int{
	"count":    COUNT,
	"avg":      AVG,
	"max":      MAX,
	"min":      MIN,
	"sum":      SUM,
	"by":       BY,
	"coalesce": COALESCE,
}

type lexer struct {
	scanner.Scanner
	expr   *RootExpr
	parser *yyParserImpl
	errs   []ParseError
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

	if tok, ok := functionTokens[l.TokenText()]; ok {
		// this matches a "function token", but could also be identifier if it's used to attempt to
		// identify a span attribute in a construction like { count > 2 }. if the next rune is a (
		// assume it's a function.
		if l.Peek() == '(' {
			return tok
		}
	}

	if tok, ok := tokens[l.TokenText()+string(l.Peek())]; ok {
		l.Next()
		return tok
	}

	if tok, ok := tokens[l.TokenText()]; ok {
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
