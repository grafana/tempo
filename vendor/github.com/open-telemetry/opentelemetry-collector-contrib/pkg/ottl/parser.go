// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/participle/v2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

// Statement holds a top level Statement for processing telemetry data. A Statement is a combination of a function
// invocation and the boolean expression to match telemetry for invoking the function.
type Statement[K any] struct {
	function          Expr[K]
	condition         BoolExpr[K]
	origText          string
	telemetrySettings component.TelemetrySettings
}

// Execute is a function that will execute the statement's function if the statement's condition is met.
// Returns true if the function was run, returns false otherwise.
// If the statement contains no condition, the function will run and true will be returned.
// In addition, the functions return value is always returned.
func (s *Statement[K]) Execute(ctx context.Context, tCtx K) (any, bool, error) {
	condition, err := s.condition.Eval(ctx, tCtx)
	defer func() {
		if s.telemetrySettings.Logger != nil {
			s.telemetrySettings.Logger.Debug("TransformContext after statement execution", zap.String("statement", s.origText), zap.Bool("condition matched", condition), zap.Any("TransformContext", tCtx))
		}
	}()
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

// Condition holds a top level Condition. A Condition is a boolean expression to match telemetry.
type Condition[K any] struct {
	condition BoolExpr[K]
	origText  string
}

// Eval returns true if the condition was met for the given TransformContext and false otherwise.
func (c *Condition[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	return c.condition.Eval(ctx, tCtx)
}

// Parser provides the means to parse OTTL StatementSequence and Conditions given a specific set of functions,
// a PathExpressionParser, and an EnumParser.
type Parser[K any] struct {
	functions         map[string]Factory[K]
	pathParser        PathExpressionParser[K]
	enumParser        EnumParser
	telemetrySettings component.TelemetrySettings
	pathContextNames  map[string]struct{}
}

func NewParser[K any](
	functions map[string]Factory[K],
	pathParser PathExpressionParser[K],
	settings component.TelemetrySettings,
	options ...Option[K],
) (Parser[K], error) {
	if settings.Logger == nil {
		return Parser[K]{}, errors.New("logger cannot be nil")
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

// WithPathContextNames sets the context names to be considered when parsing a Path value.
// When this option is empty or nil, all Path segments are considered fields, and the
// Path.Context value is always empty.
// When this option is configured, and the path's context is empty or is not present in
// this context names list, it results into an error.
func WithPathContextNames[K any](contexts []string) Option[K] {
	return func(p *Parser[K]) {
		pathContextNames := make(map[string]struct{}, len(contexts))
		for _, ctx := range contexts {
			pathContextNames[ctx] = struct{}{}
		}

		p.pathContextNames = pathContextNames
	}
}

// ParseStatements parses string statements into ottl.Statement objects ready for execution.
// Returns a slice of statements and a nil error on successful parsing.
// If parsing fails, returns nil and a joined error containing each error per failed statement.
func (p *Parser[K]) ParseStatements(statements []string) ([]*Statement[K], error) {
	parsedStatements := make([]*Statement[K], 0, len(statements))
	var parseErrs []error

	for _, statement := range statements {
		ps, err := p.ParseStatement(statement)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("unable to parse OTTL statement %q: %w", statement, err))
			continue
		}
		parsedStatements = append(parsedStatements, ps)
	}

	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	return parsedStatements, nil
}

// ParseStatement parses a single string statement into a Statement struct ready for execution.
// Returns a Statement and a nil error on successful parsing.
// If parsing fails, returns nil and an error.
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
		function:          function,
		condition:         expression,
		origText:          statement,
		telemetrySettings: p.telemetrySettings,
	}, nil
}

// ParseConditions parses string conditions into a Condition slice ready for execution.
// Returns a slice of Condition and a nil error on successful parsing.
// If parsing fails, returns nil and an error containing each error per failed condition.
func (p *Parser[K]) ParseConditions(conditions []string) ([]*Condition[K], error) {
	parsedConditions := make([]*Condition[K], 0, len(conditions))
	var parseErrs []error

	for _, condition := range conditions {
		ps, err := p.ParseCondition(condition)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("unable to parse OTTL condition %q: %w", condition, err))
			continue
		}
		parsedConditions = append(parsedConditions, ps)
	}

	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	return parsedConditions, nil
}

// ParseCondition parses a single string condition into a Condition objects ready for execution.
// Returns an Condition and a nil error on successful parsing.
// If parsing fails, returns nil and an error.
func (p *Parser[K]) ParseCondition(condition string) (*Condition[K], error) {
	parsed, err := parseCondition(condition)
	if err != nil {
		return nil, err
	}
	expression, err := p.newBoolExpr(parsed)
	if err != nil {
		return nil, err
	}
	return &Condition[K]{
		condition: expression,
		origText:  condition,
	}, nil
}

func (p *Parser[K]) prependContextToPaths(context string, ottl string, ottlPathsGetter func(ottl string) ([]path, error)) (string, error) {
	if _, ok := p.pathContextNames[context]; !ok {
		return "", fmt.Errorf(`unknown context "%s" for parser %T, valid options are: %s`, context, p, p.buildPathContextNamesText(""))
	}
	paths, err := ottlPathsGetter(ottl)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return ottl, nil
	}

	var missingContextOffsets []int
	for _, it := range paths {
		if _, ok := p.pathContextNames[it.Context]; !ok {
			missingContextOffsets = append(missingContextOffsets, it.Pos.Offset)
		}
	}

	return insertContextIntoPathsOffsets(context, ottl, missingContextOffsets)
}

// prependContextToStatementPaths changes the given OTTL statement adding the context name prefix
// to all context-less paths. No modifications are performed for paths which [Path.Context]
// value matches any WithPathContextNames value.
// The context argument must be valid WithPathContextNames value, otherwise an error is returned.
func (p *Parser[K]) prependContextToStatementPaths(context string, statement string) (string, error) {
	return p.prependContextToPaths(context, statement, func(ottl string) ([]path, error) {
		parsed, err := parseStatement(ottl)
		if err != nil {
			return nil, err
		}
		return getParsedStatementPaths(parsed), nil
	})
}

// prependContextToConditionPaths changes the given OTTL condition adding the context name prefix
// to all context-less paths. No modifications are performed for paths which [Path.Context]
// value matches any WithPathContextNames value.
// The context argument must be valid WithPathContextNames value, otherwise an error is returned.
func (p *Parser[K]) prependContextToConditionPaths(context string, condition string) (string, error) {
	return p.prependContextToPaths(context, condition, func(ottl string) ([]path, error) {
		parsed, err := parseCondition(ottl)
		if err != nil {
			return nil, err
		}
		return getBooleanExpressionPaths(parsed), nil
	})
}

var (
	parser                = newParser[parsedStatement]()
	conditionParser       = newParser[booleanExpression]()
	valueExpressionParser = newParser[value]()
)

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

func parseCondition(raw string) (*booleanExpression, error) {
	parsed, err := conditionParser.ParseString("", raw)
	if err != nil {
		return nil, fmt.Errorf("condition has invalid syntax: %w", err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func parseValueExpression(raw string) (*value, error) {
	parsed, err := valueExpressionParser.ParseString("", raw)
	if err != nil {
		return nil, fmt.Errorf("expression has invalid syntax: %w", err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func insertContextIntoPathsOffsets(context string, statement string, offsets []int) (string, error) {
	if len(offsets) == 0 {
		return statement, nil
	}

	contextPrefix := context + "."
	var sb strings.Builder
	sb.Grow(len(statement) + (len(contextPrefix) * len(offsets)))

	sort.Ints(offsets)
	left := 0
	for _, offset := range offsets {
		if offset < 0 || offset > len(statement) {
			return statement, fmt.Errorf(`failed to insert context "%s" into statement "%s": offset %d is out of range`, context, statement, offset)
		}
		sb.WriteString(statement[left:offset])
		sb.WriteString(contextPrefix)
		left = offset
	}
	sb.WriteString(statement[left:])

	return sb.String(), nil
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
		panic("Unable to initialize parser; this is a programming error in OTTL:" + err.Error())
	}
	return parser
}

// StatementSequence represents a list of statements that will be executed sequentially for a TransformContext
// and will handle errors based on an ErrorMode.
type StatementSequence[K any] struct {
	statements        []*Statement[K]
	errorMode         ErrorMode
	telemetrySettings component.TelemetrySettings
}

type StatementSequenceOption[K any] func(*StatementSequence[K])

// WithStatementSequenceErrorMode sets the ErrorMode of a StatementSequence
func WithStatementSequenceErrorMode[K any](errorMode ErrorMode) StatementSequenceOption[K] {
	return func(s *StatementSequence[K]) {
		s.errorMode = errorMode
	}
}

// NewStatementSequence creates a new StatementSequence with the provided Statement slice and component.TelemetrySettings.
// The default ErrorMode is `Propagate`.
// You may also augment the StatementSequence with a slice of StatementSequenceOption.
func NewStatementSequence[K any](statements []*Statement[K], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption[K]) StatementSequence[K] {
	s := StatementSequence[K]{
		statements:        statements,
		errorMode:         PropagateError,
		telemetrySettings: telemetrySettings,
	}
	for _, op := range options {
		op(&s)
	}
	return s
}

// Execute is a function that will execute all the statements in the StatementSequence list.
// When the ErrorMode of the StatementSequence is `propagate`, errors cause the execution to halt and the error is returned.
// When the ErrorMode of the StatementSequence is `ignore`, errors are logged and execution continues to the next statement.
// When the ErrorMode of the StatementSequence is `silent`, errors are not logged and execution continues to the next statement.
func (s *StatementSequence[K]) Execute(ctx context.Context, tCtx K) error {
	s.telemetrySettings.Logger.Debug("initial TransformContext before executing StatementSequence", zap.Any("TransformContext", tCtx))
	for _, statement := range s.statements {
		_, _, err := statement.Execute(ctx, tCtx)
		if err != nil {
			if s.errorMode == PropagateError {
				err = fmt.Errorf("failed to execute statement: %v, %w", statement.origText, err)
				return err
			}
			if s.errorMode == IgnoreError {
				s.telemetrySettings.Logger.Warn("failed to execute statement", zap.Error(err), zap.String("statement", statement.origText))
			}
		}
	}
	return nil
}

// ConditionSequence represents a list of Conditions that will be evaluated sequentially for a TransformContext
// and will handle errors returned by conditions based on an ErrorMode.
// By default, the conditions are ORed together, but they can be ANDed together using the WithLogicOperation option.
type ConditionSequence[K any] struct {
	conditions        []*Condition[K]
	errorMode         ErrorMode
	telemetrySettings component.TelemetrySettings
	logicOp           LogicOperation
}

type ConditionSequenceOption[K any] func(*ConditionSequence[K])

// WithConditionSequenceErrorMode sets the ErrorMode of a ConditionSequence
func WithConditionSequenceErrorMode[K any](errorMode ErrorMode) ConditionSequenceOption[K] {
	return func(c *ConditionSequence[K]) {
		c.errorMode = errorMode
	}
}

// WithLogicOperation sets the LogicOperation of a ConditionSequence
// When setting AND the conditions will be ANDed together.
// When setting OR the conditions will be ORed together.
func WithLogicOperation[K any](logicOp LogicOperation) ConditionSequenceOption[K] {
	return func(c *ConditionSequence[K]) {
		c.logicOp = logicOp
	}
}

// NewConditionSequence creates a new ConditionSequence with the provided Condition slice and component.TelemetrySettings.
// The default ErrorMode is `Propagate` and the default LogicOperation is `OR`.
// You may also augment the ConditionSequence with a slice of ConditionSequenceOption.
func NewConditionSequence[K any](conditions []*Condition[K], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption[K]) ConditionSequence[K] {
	c := ConditionSequence[K]{
		conditions:        conditions,
		errorMode:         PropagateError,
		telemetrySettings: telemetrySettings,
		logicOp:           Or,
	}
	for _, op := range options {
		op(&c)
	}
	return c
}

// Eval evaluates the result of each Condition in the ConditionSequence.
// The boolean logic between conditions is based on the ConditionSequence's Logic Operator.
// If using the default OR LogicOperation, if any Condition evaluates to true, then true is returned and if all Conditions evaluate to false, then false is returned.
// If using the AND LogicOperation, if any Condition evaluates to false, then false is returned and if all Conditions evaluate to true, then true is returned.
// When the ErrorMode of the ConditionSequence is `propagate`, errors cause the evaluation to be false and an error is returned.
// When the ErrorMode of the ConditionSequence is `ignore`, errors are logged and cause the evaluation to continue to the next condition.
// When the ErrorMode of the ConditionSequence is `silent`, errors are not logged and cause the evaluation to continue to the next condition.
// When using the AND LogicOperation with the `ignore` ErrorMode the sequence will evaluate to false if all conditions error.
func (c *ConditionSequence[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	var atLeastOneMatch bool
	for _, condition := range c.conditions {
		match, err := condition.Eval(ctx, tCtx)
		c.telemetrySettings.Logger.Debug("condition evaluation result", zap.String("condition", condition.origText), zap.Bool("match", match), zap.Any("TransformContext", tCtx))
		if err != nil {
			if c.errorMode == PropagateError {
				err = fmt.Errorf("failed to eval condition: %v, %w", condition.origText, err)
				return false, err
			}
			if c.errorMode == IgnoreError {
				c.telemetrySettings.Logger.Warn("failed to eval condition", zap.Error(err), zap.String("condition", condition.origText))
			}
			continue
		}
		if match {
			if c.logicOp == Or {
				return true, nil
			}
			atLeastOneMatch = true
		}
		if !match && c.logicOp == And {
			return false, nil
		}
	}
	// When ANDing it is possible to arrive here not because everything was true, but because everything errored and was ignored.
	// In that situation, we don't want to return True when no conditions actually passed. In a situation when everything failed
	// we are essentially left with an empty set, which is normally evaluated in mathematics as False. We will use that
	// idea to return False when ANDing and everything errored. We use atLeastOneMatch here to return true if anything did match.
	// It is not possible to get here if any condition during an AND explicitly failed.
	return c.logicOp == And && atLeastOneMatch, nil
}

// ValueExpression represents an expression that resolves to a value. The returned value can be of any type,
// and the expression can be either a literal value, a path value within the context, or the result of a converter and/or
// a mathematical expression.
// This allows other components using this library to extract data from the context of the incoming signal using OTTL.
type ValueExpression[K any] struct {
	getter Getter[K]
}

// Eval evaluates the given expression and returns the value the expression resolves to.
func (e *ValueExpression[K]) Eval(ctx context.Context, tCtx K) (any, error) {
	return e.getter.Get(ctx, tCtx)
}

// ParseValueExpression parses an expression string into a ValueExpression. The ValueExpression's Eval
// method can then be used to extract the value from the context of the incoming signal.
func (p *Parser[K]) ParseValueExpression(raw string) (*ValueExpression[K], error) {
	parsed, err := parseValueExpression(raw)
	if err != nil {
		return nil, err
	}
	getter, err := p.newGetter(*parsed)
	if err != nil {
		return nil, err
	}

	return &ValueExpression[K]{
		getter: &StandardGetSetter[K]{
			Getter: func(ctx context.Context, tCtx K) (any, error) {
				val, err := getter.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				switch v := val.(type) {
				case map[string]any:
					m := pcommon.NewMap()
					if err := m.FromRaw(v); err != nil {
						return nil, err
					}
					return m, nil
				default:
					return v, nil
				}
			},
		},
	}, nil
}
