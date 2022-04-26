package traceql

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"
)

func init() {
	yyErrorVerbose = true
	// yyDebug = 3
	// replaces constants with actual identifiers in error messages
	//   i.e. "expecting OPEN_BRACE" => "expecting {"
	for str, tok := range tokens {
		yyToknames[tok-yyPrivate+1] = str
	}
}

func Parse(s string) (expr *RootExpr, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); ok {
				if errors.Is(err, ParseError{}) {
					return
				}
				err = newParseError(err.Error(), 0, 0)
			}
		}
	}()
	l := lexer{
		parser: yyNewParser().(*yyParserImpl),
	}
	l.Init(strings.NewReader(s))
	l.Scanner.Error = func(_ *scanner.Scanner, msg string) {
		l.Error(msg)
	}
	e := l.parser.Parse(&l)
	if e != 0 || len(l.errs) > 0 { // jpe this check is weird. why check for e != 0
		return nil, l.errs[0]
	}
	return l.expr, nil
}

// ParseError is what is returned when we failed to parse.
type ParseError struct {
	msg       string
	line, col int
}

func (p ParseError) Error() string {
	if p.col == 0 && p.line == 0 {
		return fmt.Sprintf("parse error : %s", p.msg)
	}
	return fmt.Sprintf("parse error at line %d, col %d: %s", p.line, p.col, p.msg)
}

func newParseError(msg string, line, col int) ParseError {
	return ParseError{
		msg:  msg,
		line: line,
		col:  col,
	}
}
