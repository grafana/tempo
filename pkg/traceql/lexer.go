package traceql

import (
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"github.com/prometheus/common/model"
)

var tokens = map[string]int{
	",":               COMMA,
	".":               DOT,
	"{":               OPEN_BRACE,
	"}":               CLOSE_BRACE,
	"(":               OPEN_PARENS,
	")":               CLOSE_PARENS,
	"=":               EQ,
	"!=":              NEQ,
	"=~":              RE,
	"!~":              NRE,
	">":               GT,
	">=":              GTE,
	"<":               LT,
	"<=":              LTE,
	"+":               ADD,
	"-":               SUB,
	"/":               DIV,
	"%":               MOD,
	"*":               MUL,
	"^":               POW,
	"true":            TRUE,
	"false":           FALSE,
	"nil":             NIL,
	"ok":              STATUS_OK,
	"error":           STATUS_ERROR,
	"unset":           STATUS_UNSET,
	"unspecified":     KIND_UNSPECIFIED,
	"internal":        KIND_INTERNAL,
	"server":          KIND_SERVER,
	"client":          KIND_CLIENT,
	"producer":        KIND_PRODUCER,
	"consumer":        KIND_CONSUMER,
	"&&":              AND,
	"||":              OR,
	"!":               NOT,
	"|":               PIPE,
	">>":              DESC,
	"~":               TILDE,
	"duration":        IDURATION,
	"childCount":      CHILDCOUNT,
	"name":            NAME,
	"status":          STATUS,
	"statusMessage":   STATUS_MESSAGE,
	"kind":            KIND,
	"rootName":        ROOTNAME,
	"rootServiceName": ROOTSERVICENAME,
	"traceDuration":   TRACEDURATION,
	"parent":          PARENT,
	"parent.":         PARENT_DOT,
	"resource.":       RESOURCE_DOT,
	"span.":           SPAN_DOT,
	"count":           COUNT,
	"avg":             AVG,
	"max":             MAX,
	"min":             MIN,
	"sum":             SUM,
	"by":              BY,
	"coalesce":        COALESCE,
	"select":          SELECT,
}

type lexer struct {
	scanner.Scanner
	expr   *RootExpr
	parser *yyParserImpl
	errs   []ParseError

	parsingAttribute bool
}

func (l *lexer) Lex(lval *yySymType) int {
	// if we are currently parsing an attribute and the next rune suggests that
	//  this attribute will end, then return a special token indicating that the attribute is
	//  done parsing
	if l.parsingAttribute && !isAttributeRune(l.Peek()) {
		l.parsingAttribute = false
		return END_ATTRIBUTE
	}

	r := l.Scan()

	// if we are currently parsing an attribute then just grab everything until we find a character that ends the attribute.
	// we will handle parsing this out in ast.go
	if l.parsingAttribute {
		str := l.TokenText()
		// parse out any scopes here
		tok := tokens[str+string(l.Peek())]
		if tok == RESOURCE_DOT || tok == SPAN_DOT {
			l.Next()
			return tok
		}

		// go forward until we find the end of the attribute
		r := l.Peek()
		for isAttributeRune(r) {
			str += string(l.Next())
			r = l.Peek()
		}

		lval.staticStr = str
		return IDENTIFIER
	}

	// now that we know we're not parsing an attribute, let's look for everything else
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
		numberText := l.TokenText()

		// first try to parse as duration
		duration, ok := tryScanDuration(numberText, &l.Scanner)
		if ok {
			lval.staticDuration = duration
			return DURATION
		}

		var err error
		lval.staticFloat, err = strconv.ParseFloat(numberText, 64)
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		return FLOAT
	}

	tokStrNext := l.TokenText() + string(l.Peek())
	if tok, ok := tokens[tokStrNext]; ok {
		l.Next()
		l.parsingAttribute = startsAttribute(tok)
		return tok
	}

	if tok, ok := tokens[l.TokenText()]; ok {
		l.parsingAttribute = startsAttribute(tok)
		return tok
	}

	lval.staticStr = l.TokenText()
	return IDENTIFIER
}

func (l *lexer) Error(msg string) {
	l.errs = append(l.errs, newParseError(msg, l.Line, l.Column))
}

func tryScanDuration(number string, l *scanner.Scanner) (time.Duration, bool) {
	var sb strings.Builder
	sb.WriteString(number)
	// copy the scanner to avoid advancing it in case it's not a duration.
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
	d, err := parseDuration(sb.String())
	if err != nil {
		return 0, false
	}
	// we need to consume the scanner, now that we know this is a duration.
	for i := 0; i < consumed; i++ {
		_ = l.Next()
	}
	return d, true
}

func parseDuration(d string) (time.Duration, error) {
	var duration time.Duration
	// Try to parse promql style durations first, to ensure that we support the same duration
	// units as promql
	prometheusDuration, err := model.ParseDuration(d)
	if err != nil {
		// Fall back to standard library's time.ParseDuration if a promql style
		// duration couldn't be parsed.
		duration, err = time.ParseDuration(d)
		if err != nil {
			return 0, err
		}
	} else {
		duration = time.Duration(prometheusDuration)
	}

	return duration, nil
}

func isDurationRune(r rune) bool {
	// "ns", "us" (or "µs"), "ms", "s", "m", "h".
	switch r {
	case 'n', 's', 'u', 'm', 'h', 'µ', 'd', 'w', 'y':
		return true
	default:
		return false
	}
}

func isAttributeRune(r rune) bool {
	if unicode.IsSpace(r) {
		return false
	}

	switch r {
	case scanner.EOF, '{', '}', '(', ')', '=', '~', '!', '<', '>', '&', '|', '^', ',':
		return false
	default:
		return true
	}
}

func startsAttribute(tok int) bool {
	return tok == DOT ||
		tok == RESOURCE_DOT ||
		tok == SPAN_DOT ||
		tok == PARENT_DOT
}
