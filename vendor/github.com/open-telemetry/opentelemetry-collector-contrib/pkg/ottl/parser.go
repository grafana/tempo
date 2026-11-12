// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"
)

// Statement holds a top level Statement for processing telemetry data. A Statement is a combination of a function
// invocation and the boolean expression to match telemetry for invoking the function.
type Statement[K any] struct {
	function          Expr[K]
	condition         boolExpr[K]
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
		if s.telemetrySettings.Logger.Core().Enabled(zap.DebugLevel) {
			s.telemetrySettings.Logger.Debug("TransformContext after statement execution", zap.String("statement", s.origText), zap.Bool("condition matched", condition), newTransformContextField(tCtx))
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

// String returns the original statement text used to create the Statement.
func (s *Statement[K]) String() string {
	return s.origText
}

// Condition holds a top level Condition. A Condition is a boolean expression to match telemetry.
type Condition[K any] struct {
	condition boolExpr[K]
	origText  string
}

// String returns the original condition text used to create the Condition.
func (c *Condition[K]) String() string {
	return c.origText
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

// NewParser creates a new Parser
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

// Option is an option for a Parser
type Option[K any] func(*Parser[K])

// WithEnumParser allows setting the enums a parser can use during parsing.
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

	experimentalFuncs := map[string]struct{}{}
	for _, statement := range statements {
		ps, experimental, err := p.buildStatement(statement)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("unable to parse OTTL statement %q: %w", statement, err))
			continue
		}
		maps.Copy(experimentalFuncs, experimental)
		parsedStatements = append(parsedStatements, ps)
	}

	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	p.warnExperimentalFuncs(experimentalFuncs)
	return parsedStatements, nil
}

// ParseStatement parses a single string statement into a Statement struct ready for execution.
// Returns a Statement and a nil error on successful parsing.
// If parsing fails, returns nil and an error.
func (p *Parser[K]) ParseStatement(statement string) (*Statement[K], error) {
	stmt, experimentalFuncs, err := p.buildStatement(statement)
	if err != nil {
		return nil, err
	}
	p.warnExperimentalFuncs(experimentalFuncs)
	return stmt, nil
}

// buildStatement parses a single statement and returns the Statement along with the set of
// experimental function names it uses.
func (p *Parser[K]) buildStatement(statement string) (*Statement[K], map[string]struct{}, error) {
	parsed, err := parseStatement(statement)
	if err != nil {
		return nil, nil, err
	}

	pc := p.newParseContext()
	function, err := pc.newFunctionCall(parsed.Editor)
	if err != nil {
		return nil, nil, formatParseError("statement", statement, err)
	}
	expression, err := pc.newBoolExpr(parsed.WhereClause)
	if err != nil {
		return nil, nil, formatParseError("statement", statement, err)
	}
	return &Statement[K]{
		function:          function,
		condition:         expression,
		origText:          statement,
		telemetrySettings: p.telemetrySettings,
	}, pc.experimentalFuncs, nil
}

// ParseConditions parses string conditions into a Condition slice ready for execution.
// Returns a slice of Condition and a nil error on successful parsing.
// If parsing fails, returns nil and an error containing each error per failed condition.
func (p *Parser[K]) ParseConditions(conditions []string) ([]*Condition[K], error) {
	parsedConditions := make([]*Condition[K], 0, len(conditions))
	var parseErrs []error

	experimentalFuncs := map[string]struct{}{}
	for _, condition := range conditions {
		ps, experimental, err := p.buildCondition(condition)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("unable to parse OTTL condition %q: %w", condition, err))
			continue
		}
		maps.Copy(experimentalFuncs, experimental)
		parsedConditions = append(parsedConditions, ps)
	}

	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	p.warnExperimentalFuncs(experimentalFuncs)
	return parsedConditions, nil
}

// ParseCondition parses a single string condition into a Condition objects ready for execution.
// Returns an Condition and a nil error on successful parsing.
// If parsing fails, returns nil and an error.
func (p *Parser[K]) ParseCondition(condition string) (*Condition[K], error) {
	cond, experimentalFuncs, err := p.buildCondition(condition)
	if err != nil {
		return nil, err
	}
	p.warnExperimentalFuncs(experimentalFuncs)
	return cond, nil
}

// buildCondition parses a single condition and returns the Condition along with the set of
// experimental function names it uses.
func (p *Parser[K]) buildCondition(condition string) (*Condition[K], map[string]struct{}, error) {
	parsed, err := parseCondition(condition)
	if err != nil {
		return nil, nil, err
	}

	pc := p.newParseContext()
	expression, err := pc.newBoolExpr(parsed)
	if err != nil {
		return nil, nil, formatParseError("condition", condition, err)
	}
	return &Condition[K]{
		condition: expression,
		origText:  condition,
	}, pc.experimentalFuncs, nil
}

func (p *Parser[K]) prependContextToPaths(context, ottl string, ottlPathsGetter func(ottl string) ([]path, error)) (string, error) {
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
func (p *Parser[K]) prependContextToStatementPaths(context, statement string) (string, error) {
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
func (p *Parser[K]) prependContextToConditionPaths(context, condition string) (string, error) {
	return p.prependContextToPaths(context, condition, func(ottl string) ([]path, error) {
		parsed, err := parseCondition(ottl)
		if err != nil {
			return nil, err
		}
		return getBooleanExpressionPaths(parsed), nil
	})
}

// prependContextToValueExpressionPaths changes the given OTTL value expression adding the context name prefix
// to all context-less paths. No modifications are performed for paths which [Path.Context]
// value matches any WithPathContextNames value.
// The context argument must be valid WithPathContextNames value, otherwise an error is returned.
func (p *Parser[K]) prependContextToValueExpressionPaths(context, expr string) (string, error) {
	return p.prependContextToPaths(context, expr, func(ottl string) ([]path, error) {
		parsed, err := parseValueExpression(ottl)
		if err != nil {
			return nil, err
		}
		return getValuePaths(parsed), nil
	})
}

var (
	parser                = sync.OnceValue(newParser[parsedStatement])
	conditionParser       = sync.OnceValue(newParser[booleanExpression])
	valueExpressionParser = sync.OnceValue(newParser[value])
)

func parseStatement(raw string) (*parsedStatement, error) {
	parsed, err := parser().ParseString("", raw)
	if err != nil {
		return nil, formatParseError("statement", raw, err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, formatParseError("statement", raw, err)
	}

	return parsed, nil
}

func parseCondition(raw string) (*booleanExpression, error) {
	parsed, err := conditionParser().ParseString("", raw)
	if err != nil {
		return nil, formatParseError("condition", raw, err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, formatParseError("condition", raw, err)
	}

	return parsed, nil
}

func parseValueExpression(raw string) (*value, error) {
	parsed, err := valueExpressionParser().ParseString("", raw)
	if err != nil {
		return nil, formatParseError("expression", raw, err)
	}
	err = parsed.checkForCustomError()
	if err != nil {
		return nil, formatParseError("expression", raw, err)
	}

	return parsed, nil
}

func formatSyntaxError(kind, ottl string, err *participle.UnexpectedTokenError) string {
	var expected string
	if msg := err.Message(); msg != "" {
		if idx := strings.Index(msg, "(expected "); idx >= 0 {
			expected = " " + msg[idx:]
		}
	}
	pos := err.Position()
	if near := nearParseError(ottl, pos.Offset); near != "" {
		return fmt.Sprintf("%s has invalid syntax near `%s`:%s", kind, near, expected)
	}
	return fmt.Sprintf("%s has invalid syntax:%s", kind, expected)
}

func formatErrorWithPosition(kind, ottl string, err error, pos lexer.Position) string {
	var msg string
	if e, ok := err.(interface{ Message() string }); ok {
		msg = e.Message()
	} else {
		msg = err.Error()
	}
	if near := nearParseError(ottl, pos.Offset); near != "" {
		return fmt.Sprintf("%s failed to parse near `%s`: %s", kind, near, msg)
	}
	return fmt.Sprintf("%s failed to parse: %s", kind, msg)
}

// errorPosition extracts a lexer.Position from any error kind that carries one.
func errorPosition(err error) (lexer.Position, bool) {
	if pe, ok := errors.AsType[ottlerror.Error](err); ok {
		return ottlerror.AsLexerPosition(pe.Position()), true
	}
	if v, ok := errors.AsType[participle.Error](err); ok {
		return v.Position(), true
	}
	return lexer.Position{}, false
}

func formatParseError(kind, ottl string, err error) error {
	if v, ok := errors.AsType[*grammarCustomError](err); ok {
		if len(v.errs) == 1 {
			if pos, ok := errorPosition(v.errs[0]); ok {
				return ottlerror.Wrap(pos, v.errs[0], formatErrorWithPosition(kind, ottl, v.errs[0], pos))
			}
		}
		return ottlerror.Wrap(lexer.Position{}, v, fmt.Sprintf("%s failed to parse: %s", kind, v.Error()))
	}
	if v, ok := errors.AsType[*participle.UnexpectedTokenError](err); ok {
		return ottlerror.Wrap(v.Position(), err, formatSyntaxError(kind, ottl, v))
	}
	if pos, ok := errorPosition(err); ok {
		return ottlerror.Wrap(pos, err, formatErrorWithPosition(kind, ottl, err, pos))
	}
	return ottlerror.Wrapf(lexer.Position{}, err, "%s is invalid", kind)
}

// parseErrorSnippetLen is the number of source characters shown after the error position in the "near" clause.
const parseErrorSnippetLen = 10

// nearParseError returns a short, UTF-8 safe snippet of raw starting at offset for use in error messages.
func nearParseError(raw string, offset int) string {
	if offset < 0 || offset >= len(raw) {
		return ""
	}
	end := min(offset+parseErrorSnippetLen, len(raw))
	return strings.TrimSpace(strings.ToValidUTF8(raw[offset:end], ""))
}

func insertContextIntoPathsOffsets(context, statement string, offsets []int) (string, error) {
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

// StatementSequenceOption is an option for a StatementSequence
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
	if s.telemetrySettings.Logger.Core().Enabled(zap.DebugLevel) {
		s.telemetrySettings.Logger.Debug("initial TransformContext before executing StatementSequence", zap.Any("TransformContext", tCtx))
	}
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

// ConditionSequenceOption is an option for a ConditionSequence
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
		if c.telemetrySettings.Logger.Core().Enabled(zap.DebugLevel) {
			c.telemetrySettings.Logger.Debug("condition evaluation result", zap.String("condition", condition.origText), zap.Bool("match", match), newTransformContextField(tCtx))
		}
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
	getter   Getter[K]
	origText string
}

// Eval evaluates the given expression and returns the value the expression resolves to.
func (e *ValueExpression[K]) Eval(ctx context.Context, tCtx K) (any, error) {
	return e.getter.Get(ctx, tCtx)
}

// String returns the original OTTL expression used to create the ValueExpression.
func (e *ValueExpression[K]) String() string {
	return e.origText
}

// ParseValueExpressions parses string expressions into a ValueExpression slice ready for execution.
// Returns a slice of ValueExpression and a nil error on successful parsing.
// If parsing fails, returns nil and an error containing each error per failed condition.
func (p *Parser[K]) ParseValueExpressions(expressions []string) ([]*ValueExpression[K], error) {
	parsedValueExpressions := make([]*ValueExpression[K], 0, len(expressions))
	var parseErrs []error

	experimentalFuncs := map[string]struct{}{}
	for _, expression := range expressions {
		ps, experimental, err := p.buildValueExpression(expression)
		if err != nil {
			parseErrs = append(parseErrs, fmt.Errorf("unable to parse OTTL value expression %q: %w", expression, err))
			continue
		}
		maps.Copy(experimentalFuncs, experimental)
		parsedValueExpressions = append(parsedValueExpressions, ps)
	}

	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	p.warnExperimentalFuncs(experimentalFuncs)
	return parsedValueExpressions, nil
}

// ParseValueExpression parses an expression string into a ValueExpression. The ValueExpression's Eval
// method can then be used to extract the value from the context of the incoming signal.
func (p *Parser[K]) ParseValueExpression(raw string) (*ValueExpression[K], error) {
	valueExpression, experimentalFuncs, err := p.buildValueExpression(raw)
	if err != nil {
		return nil, err
	}
	p.warnExperimentalFuncs(experimentalFuncs)
	return valueExpression, nil
}

// buildValueExpression parses a single value expression and returns the ValueExpression along with
// the set of experimental function names it uses.
func (p *Parser[K]) buildValueExpression(raw string) (*ValueExpression[K], map[string]struct{}, error) {
	parsed, err := parseValueExpression(raw)
	if err != nil {
		return nil, nil, err
	}
	pc := p.newParseContext()
	getter, err := pc.newGetter(*parsed)
	if err != nil {
		return nil, nil, err
	}

	return &ValueExpression[K]{
		origText: raw,
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
	}, pc.experimentalFuncs, nil
}

// warnExperimentalFuncs emits a single warning listing the experimental functions used by a parsed
// configuration. It does nothing when no experimental functions were used.
func (p *Parser[K]) warnExperimentalFuncs(funcs map[string]struct{}) {
	if len(funcs) == 0 {
		return
	}
	names := make([]string, 0, len(funcs))
	for name := range funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	p.telemetrySettings.Logger.Warn(
		"OTTL configuration uses experimental functions that are not covered by stability guarantees; their names, arguments, and behavior may change or be removed",
		zap.Strings("functions", names),
	)
}

// parseContext represents the context used during parsing operations. It is used to store
// the current parser reference and lexical scopes for local identifiers (e.g. in lambda bodies).
type parseContext[K any] struct {
	*Parser[K]
	localScopes       localScopeStack
	experimentalFuncs map[string]struct{}
}

func (p *Parser[K]) newParseContext() *parseContext[K] {
	return &parseContext[K]{
		Parser:            p,
		experimentalFuncs: make(map[string]struct{}),
	}
}

// legacyExperimentalFuncGates maps experimental functions that predate the experimental functions
// feature gate to the feature gate they require instead. A nil gate means the function is ungated.
var legacyExperimentalFuncGates = map[string]*featuregate.Gate{
	"All":       metadata.OttlFunctionsEnableLambdaFeatureGate,
	"Any":       metadata.OttlFunctionsEnableLambdaFeatureGate,
	"Filter":    metadata.OttlFunctionsEnableLambdaFeatureGate,
	"Find":      metadata.OttlFunctionsEnableLambdaFeatureGate,
	"MapEach":   metadata.OttlFunctionsEnableLambdaFeatureGate,
	"MapKeys":   metadata.OttlFunctionsEnableLambdaFeatureGate,
	"Reduce":    metadata.OttlFunctionsEnableLambdaFeatureGate,
	"When":      metadata.OttlFunctionsEnableLambdaFeatureGate,
	"ProfileID": nil,
}

// recordExperimentalFunc records the given Factory's function name when it is experimental so a
// warning can be emitted after parsing completes. Experimental functions are rejected unless their
// feature gate is enabled.
func (p *parseContext[K]) recordExperimentalFunc(f Factory[K]) error {
	if !f.Experimental() {
		return nil
	}
	gate, ok := legacyExperimentalFuncGates[f.Name()]
	if !ok {
		gate = metadata.PkgOttlFunctionsEnableExperimentalFeatureGate
	}
	if gate != nil && !gate.IsEnabled() {
		return fmt.Errorf("function %q is experimental and requires the `%s` feature gate to be enabled", f.Name(), gate.ID())
	}
	p.experimentalFuncs[f.Name()] = struct{}{}
	return nil
}
