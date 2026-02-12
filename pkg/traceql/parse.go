package traceql

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/util/log"
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
	return parseWithOptimizationOption(s, true)
}

func parseWithOptimizationOption(s string, astOptimization bool) (expr *RootExpr, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); ok {
				var parseErr *ParseError
				if errors.As(err, &parseErr) {
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
	if len(l.errs) > 0 {
		return nil, l.errs[0]
	}
	if e != 0 {
		return nil, fmt.Errorf("unknown parse error: %d", e)
	}

	hintSkipOptimization, _ := l.expr.Hints.GetBool(HintSkipOptimization, true)
	if astOptimization && !hintSkipOptimization {
		l.expr = ApplyDefaultASTRewrites(l.expr)
		level.Debug(log.Logger).Log("msg", "optimize AST for TraceQL query", "query", s, "optimizedQuery", l.expr.String(), "optimizationCount", l.expr.OptimizationCount)
	}

	return l.expr, nil
}

// warning: ParseIdentifier is used to parse filter policies in pkg/spanfilter/config/config.go
// if changed, it can break existing config
func ParseIdentifier(s string) (Attribute, error) {
	// Wrap the identifier in curly braces to create a valid spanset filter expression
	attr := "{" + s + "}"
	expr, err := Parse(attr)
	if err != nil {
		return Attribute{}, fmt.Errorf("failed to parse identifier %s: %w", s, err)
	}

	// Validate the parsed expression structure
	if expr == nil {
		return Attribute{}, fmt.Errorf("failed to parse identifier %s: parsed expression is nil", s)
	}

	if len(expr.Pipeline.Elements) == 0 {
		return Attribute{}, fmt.Errorf("failed to parse identifier %s: no pipeline elements found", s)
	}

	// Extract and validate the spanset filter
	filter, ok := expr.Pipeline.Elements[0].(*SpansetFilter)
	if !ok {
		return Attribute{}, fmt.Errorf("failed to parse identifier %s: expected SpansetFilter but got %T", s, expr.Pipeline.Elements[0])
	}

	// Extract and validate the attribute
	attribute, ok := filter.Expression.(Attribute)
	if !ok {
		return Attribute{}, fmt.Errorf("failed to parse identifier %s: expected Attribute but got %T", s, filter.Expression)
	}

	return attribute, nil
}

func MustParseIdentifier(s string) Attribute {
	a, err := ParseIdentifier(s)
	if err != nil {
		panic(err)
	}
	return a
}

// ParseError is what is returned when we failed to parse.
type ParseError struct {
	msg       string
	line, col int
}

func (p *ParseError) Error() string {
	if p.col == 0 && p.line == 0 {
		return fmt.Sprintf("parse error : %s", p.msg)
	}
	return fmt.Sprintf("parse error at line %d, col %d: %s", p.line, p.col, p.msg)
}

func newParseError(msg string, line, col int) *ParseError {
	return &ParseError{
		msg:  msg,
		line: line,
		col:  col,
	}
}
