// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type ErrorMode string

const (
	IgnoreError    ErrorMode = "ignore"
	PropagateError ErrorMode = "propagate"
)

func (e *ErrorMode) UnmarshalText(text []byte) error {
	str := ErrorMode(strings.ToLower(string(text)))
	switch str {
	case IgnoreError, PropagateError:
		*e = str
		return nil
	default:
		return fmt.Errorf("unknown error mode %v", str)
	}
}

type Parser[K any] struct {
	functions         map[string]Factory[K]
	pathParser        PathExpressionParser[K]
	enumParser        EnumParser
	telemetrySettings component.TelemetrySettings
}

// Statement holds a top level Statement for processing telemetry data. A Statement is a combination of a function
// invocation and the boolean expression to match telemetry for invoking the function.
type Statement[K any] struct {
	function  Expr[K]
	condition BoolExpr[K]
	origText  string
}

// Execute is a function that will execute the statement's function if the statement's condition is met.
// Returns true if the function was run, returns false otherwise.
// If the statement contains no condition, the function will run and true will be returned.
// In addition, the functions return value is always returned.
func (s *Statement[K]) Execute(ctx context.Context, tCtx K) (any, bool, error) {
	condition, err := s.condition.Eval(ctx, tCtx)
	if err != nil {
		return nil, false, err
	}
	var result any
	if condition {
		result, err = s.function.Eval(ctx, tCtx)
		if err != nil {
			return nil, true, err
		}
	}
	return result, condition, nil
}

func NewParser[K any](
	functions map[string]Factory[K],
	pathParser PathExpressionParser[K],
	settings component.TelemetrySettings,
	options ...Option[K],
) (Parser[K], error) {
	if settings.Logger == nil {
		return Parser[K]{}, fmt.Errorf("logger cannot be nil")
	}
	p := Parser[K]{
		functions:  functions,
		pathParser: pathParser,
		enumParser: func(*EnumSymbol) (*Enum, error) {
			return nil, fmt.Errorf("enums aren't supported for the current context: %T", new(K))
		},
		telemetrySettings: settings,
	}
	for _, opt := range options {
		opt(&p)
	}
	return p, nil
}

type Option[K any] func(*Parser[K])

func WithEnumParser[K any](parser EnumParser) Option[K] {
	return func(p *Parser[K]) {
		p.enumParser = parser
	}
}

// ParseStatements parses string statements into ottl.Statement objects ready for execution.
// Returns a slice of statements and a nil error on successful parsing.
// If parsing fails, returns an empty slice  with a multierr error containing
// an error per failed statement.
func (p *Parser[K]) ParseStatements(statements []string) ([]*Statement[K], error) {
	parsedStatements := make([]*Statement[K], 0, len(statements))
	var parseErr error

	for _, statement := range statements {
		ps, err := p.ParseStatement(statement)
		if err != nil {
			parseErr = multierr.Append(parseErr, fmt.Errorf("unable to parse OTTL statement %q: %w", statement, err))
			continue
		}
		parsedStatements = append(parsedStatements, ps)
	}

	if parseErr != nil {
		return nil, parseErr
	}

	return parsedStatements, nil
}

func (p *Parser[K]) ParseStatement(statement string) (*Statement[K], error) {
	parsed, err := parseStatement(statement)
	if err != nil {
		return nil, err
	}
	function, err := p.newFunctionCall(parsed.Editor)
	if err != nil {
		return nil, err
	}
	expression, err := p.newBoolExpr(parsed.WhereClause)
	if err != nil {
		return nil, err
	}
	return &Statement[K]{
		function:  function,
		condition: expression,
		origText:  statement,
	}, nil
}

var parser = newParser[parsedStatement]()

func parseStatement(raw string) (*parsedStatement, error) {
	parsed, err := parser.ParseString("", raw)

	if err != nil {
		return nil, fmt.Errorf("statement has invalid syntax: %w", err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

// newParser returns a parser that can be used to read a string into a parsedStatement. An error will be returned if the string
// is not formatted for the DSL.
func newParser[G any]() *participle.Parser[G] {
	lex := buildLexer()
	parser, err := participle.Build[G](
		participle.Lexer(lex),
		participle.Unquote("String"),
		participle.Elide("whitespace"),
		participle.UseLookahead(participle.MaxLookahead), // Allows negative lookahead to work properly in 'value' for 'mathExprLiteral'.
	)
	if err != nil {
		panic("Unable to initialize parser; this is a programming error in the transformprocessor:" + err.Error())
	}
	return parser
}

// Statements represents a list of statements that will be executed sequentially for a TransformContext.
type Statements[K any] struct {
	statements        []*Statement[K]
	errorMode         ErrorMode
	telemetrySettings component.TelemetrySettings
}

type StatementsOption[K any] func(*Statements[K])

func WithErrorMode[K any](errorMode ErrorMode) StatementsOption[K] {
	return func(s *Statements[K]) {
		s.errorMode = errorMode
	}
}

func NewStatements[K any](statements []*Statement[K], telemetrySettings component.TelemetrySettings, options ...StatementsOption[K]) Statements[K] {
	s := Statements[K]{
		statements:        statements,
		telemetrySettings: telemetrySettings,
	}
	for _, op := range options {
		op(&s)
	}
	return s
}

// Execute is a function that will execute all the statements in the Statements list.
func (s *Statements[K]) Execute(ctx context.Context, tCtx K) error {
	for _, statement := range s.statements {
		_, _, err := statement.Execute(ctx, tCtx)
		if err != nil {
			if s.errorMode == PropagateError {
				err = fmt.Errorf("failed to execute statement: %v, %w", statement.origText, err)
				return err
			}
			s.telemetrySettings.Logger.Warn("failed to execute statement", zap.Error(err), zap.String("statement", statement.origText))
		}
	}
	return nil
}

// Eval returns true if any statement's condition is true and returns false otherwise.
// Does not execute the statement's function.
// When errorMode is `propagate`, errors cause the evaluation to be false and an error is returned.
// When errorMode is `ignore`, errors cause evaluation to continue to the next statement.
func (s *Statements[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	for _, statement := range s.statements {
		match, err := statement.condition.Eval(ctx, tCtx)
		if err != nil {
			if s.errorMode == PropagateError {
				err = fmt.Errorf("failed to eval statement: %v, %w", statement.origText, err)
				return false, err
			}
			s.telemetrySettings.Logger.Warn("failed to eval statement", zap.Error(err), zap.String("statement", statement.origText))
			continue
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
