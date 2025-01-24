// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// Safeguard to statically ensure the Parser.ParseStatements method can be reflectively
// invoked by the ottlParserWrapper.parseStatements
var _ interface {
	ParseStatements(statements []string) ([]*Statement[any], error)
} = (*Parser[any])(nil)

// Safeguard to statically ensure any ParsedStatementConverter method can be reflectively
// invoked by the statementsConverterWrapper.call
var _ ParsedStatementConverter[any, any] = func(
	_ *ParserCollection[any],
	_ *Parser[any],
	_ string,
	_ StatementsGetter,
	_ []*Statement[any],
) (any, error) {
	return nil, nil
}

// StatementsGetter represents a set of statements to be parsed.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type StatementsGetter interface {
	// GetStatements retrieves the OTTL statements to be parsed
	GetStatements() []string
}

type defaultStatementsGetter []string

func (d defaultStatementsGetter) GetStatements() []string {
	return d
}

// NewStatementsGetter creates a new StatementsGetter.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func NewStatementsGetter(statements []string) StatementsGetter {
	return defaultStatementsGetter(statements)
}

// ottlParserWrapper wraps an ottl.Parser using reflection, so it can invoke exported
// methods without knowing its generic type (transform context).
type ottlParserWrapper struct {
	parser                         reflect.Value
	prependContextToStatementPaths func(context string, statement string) (string, error)
}

func newParserWrapper[K any](parser *Parser[K]) *ottlParserWrapper {
	return &ottlParserWrapper{
		parser:                         reflect.ValueOf(parser),
		prependContextToStatementPaths: parser.prependContextToStatementPaths,
	}
}

func (g *ottlParserWrapper) parseStatements(statements []string) (reflect.Value, error) {
	method := g.parser.MethodByName("ParseStatements")
	parseStatementsRes := method.Call([]reflect.Value{reflect.ValueOf(statements)})
	err := parseStatementsRes[1]
	if !err.IsNil() {
		return reflect.Value{}, err.Interface().(error)
	}
	return parseStatementsRes[0], nil
}

func (g *ottlParserWrapper) prependContextToStatementsPaths(context string, statements []string) ([]string, error) {
	result := make([]string, 0, len(statements))
	for _, s := range statements {
		prependedStatement, err := g.prependContextToStatementPaths(context, s)
		if err != nil {
			return nil, err
		}
		result = append(result, prependedStatement)
	}
	return result, nil
}

// statementsConverterWrapper is a reflection-based wrapper to the ParsedStatementConverter function,
// which does not require knowing all generic parameters to be called.
type statementsConverterWrapper reflect.Value

func newStatementsConverterWrapper[K any, R any](converter ParsedStatementConverter[K, R]) statementsConverterWrapper {
	return statementsConverterWrapper(reflect.ValueOf(converter))
}

func (s statementsConverterWrapper) call(
	parserCollection reflect.Value,
	ottlParser *ottlParserWrapper,
	context string,
	statements StatementsGetter,
	parsedStatements reflect.Value,
) (reflect.Value, error) {
	result := reflect.Value(s).Call([]reflect.Value{
		parserCollection,
		ottlParser.parser,
		reflect.ValueOf(context),
		reflect.ValueOf(statements),
		parsedStatements,
	})

	resultValue := result[0]
	resultError := result[1]
	if !resultError.IsNil() {
		return reflect.Value{}, resultError.Interface().(error)
	}

	return resultValue, nil
}

// parserCollectionParser holds an ottlParserWrapper and its respectively
// statementsConverter function.
type parserCollectionParser struct {
	ottlParser          *ottlParserWrapper
	statementsConverter statementsConverterWrapper
}

// ParserCollection is a configurable set of ottl.Parser that can handle multiple OTTL contexts
// parsings, inferring the context, choosing the right parser for the given statements, and
// transforming the parsed ottl.Statement[K] slice into a common result of type R.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type ParserCollection[R any] struct {
	contextParsers            map[string]*parserCollectionParser
	contextInferrer           contextInferrer
	contextInferrerCandidates map[string]*priorityContextInferrerCandidate
	candidatesLowerContexts   map[string][]string
	modifiedStatementLogging  bool
	Settings                  component.TelemetrySettings
	ErrorMode                 ErrorMode
}

// ParserCollectionOption is a configurable ParserCollection option.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type ParserCollectionOption[R any] func(*ParserCollection[R]) error

// NewParserCollection creates a new ParserCollection.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func NewParserCollection[R any](
	settings component.TelemetrySettings,
	options ...ParserCollectionOption[R],
) (*ParserCollection[R], error) {
	contextInferrerCandidates := map[string]*priorityContextInferrerCandidate{}
	pc := &ParserCollection[R]{
		Settings:                  settings,
		contextParsers:            map[string]*parserCollectionParser{},
		contextInferrer:           newPriorityContextInferrer(contextInferrerCandidates),
		contextInferrerCandidates: contextInferrerCandidates,
		candidatesLowerContexts:   map[string][]string{},
	}

	for _, op := range options {
		err := op(pc)
		if err != nil {
			return nil, err
		}
	}

	return pc, nil
}

// ParsedStatementConverter is a function that converts the parsed ottl.Statement[K] into
// a common representation to all parser collection contexts passed through WithParserCollectionContext.
// Given each parser has its own transform context type, they must agree on a common type [R]
// so it can be returned by the ParserCollection.ParseStatements and ParserCollection.ParseStatementsWithContext
// functions.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type ParsedStatementConverter[K any, R any] func(
	collection *ParserCollection[R],
	parser *Parser[K],
	context string,
	statements StatementsGetter,
	parsedStatements []*Statement[K],
) (R, error)

func newNopParsedStatementConverter[K any]() ParsedStatementConverter[K, any] {
	return func(
		_ *ParserCollection[any],
		_ *Parser[K],
		_ string,
		_ StatementsGetter,
		parsedStatements []*Statement[K],
	) (any, error) {
		return parsedStatements, nil
	}
}

// WithParserCollectionContext configures an ottl.Parser for the given context.
// The provided ottl.Parser must be configured to support the provided context using
// the ottl.WithPathContextNames option.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func WithParserCollectionContext[K any, R any](
	context string,
	parser *Parser[K],
	converter ParsedStatementConverter[K, R],
) ParserCollectionOption[R] {
	return func(mp *ParserCollection[R]) error {
		if _, ok := parser.pathContextNames[context]; !ok {
			return fmt.Errorf(`context "%s" must be a valid "%T" path context name`, context, parser)
		}
		mp.contextParsers[context] = &parserCollectionParser{
			ottlParser:          newParserWrapper[K](parser),
			statementsConverter: newStatementsConverterWrapper(converter),
		}

		for lowerContext := range parser.pathContextNames {
			if lowerContext != context {
				mp.candidatesLowerContexts[lowerContext] = append(mp.candidatesLowerContexts[lowerContext], context)
			}
		}

		mp.contextInferrerCandidates[context] = &priorityContextInferrerCandidate{
			hasEnumSymbol: func(enum *EnumSymbol) bool {
				_, err := parser.enumParser(enum)
				return err == nil
			},
			hasFunctionName: func(name string) bool {
				_, ok := parser.functions[name]
				return ok
			},
			getLowerContexts: mp.getLowerContexts,
		}
		return nil
	}
}

func (pc *ParserCollection[R]) getLowerContexts(context string) []string {
	return pc.candidatesLowerContexts[context]
}

// WithParserCollectionErrorMode has no effect on the ParserCollection, but might be used
// by the ParsedStatementConverter functions to handle/create StatementSequence.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func WithParserCollectionErrorMode[R any](errorMode ErrorMode) ParserCollectionOption[R] {
	return func(tp *ParserCollection[R]) error {
		tp.ErrorMode = errorMode
		return nil
	}
}

// EnableParserCollectionModifiedStatementLogging controls the statements modification logs.
// When enabled, it logs any statements modifications performed by the parsing operations,
// instructing users to rewrite the statements accordingly.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func EnableParserCollectionModifiedStatementLogging[R any](enabled bool) ParserCollectionOption[R] {
	return func(tp *ParserCollection[R]) error {
		tp.modifiedStatementLogging = enabled
		return nil
	}
}

// ParseStatements parses the given statements into [R] using the configured context's ottl.Parser
// and subsequently calling the ParsedStatementConverter function.
// The statement's context is automatically inferred from the [Path.Context] values, choosing the
// highest priority context found.
// If no contexts are present in the statements, or if the inferred value is not supported by
// the [ParserCollection], it returns an error.
// If parsing the statements fails, it returns the underlying [ottl.Parser.ParseStatements] error.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func (pc *ParserCollection[R]) ParseStatements(statements StatementsGetter) (R, error) {
	statementsValues := statements.GetStatements()
	inferredContext, err := pc.contextInferrer.infer(statementsValues)
	if err != nil {
		return *new(R), err
	}

	if inferredContext == "" {
		return *new(R), fmt.Errorf("unable to infer context from statements %+q, path's first segment must be a valid context name: %+q", statementsValues, pc.supportedContextNames())
	}

	_, ok := pc.contextParsers[inferredContext]
	if !ok {
		return *new(R), fmt.Errorf(`context "%s" inferred from the statements %+q is not a supported context: %+q`, inferredContext, statementsValues, pc.supportedContextNames())
	}

	return pc.ParseStatementsWithContext(inferredContext, statements, false)
}

// ParseStatementsWithContext parses the given statements into [R] using the configured
// context's ottl.Parser and subsequently calling the ParsedStatementConverter function.
// Unlike ParseStatements, it uses the provided context and does not infer it
// automatically. The context value must be supported by the [ParserCollection],
// otherwise an error is returned.
// If the statement's Path does not provide their Path.Context value, the prependPathsContext
// argument should be set to true, so it rewrites the statements prepending the missing paths
// contexts.
// If parsing the statements fails, it returns the underlying [ottl.Parser.ParseStatements] error.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func (pc *ParserCollection[R]) ParseStatementsWithContext(context string, statements StatementsGetter, prependPathsContext bool) (R, error) {
	contextParser, ok := pc.contextParsers[context]
	if !ok {
		return *new(R), fmt.Errorf(`unknown context "%s" for stataments: %v`, context, statements.GetStatements())
	}

	var err error
	var parsingStatements []string
	if prependPathsContext {
		originalStatements := statements.GetStatements()
		parsingStatements, err = contextParser.ottlParser.prependContextToStatementsPaths(context, originalStatements)
		if err != nil {
			return *new(R), err
		}
		if pc.modifiedStatementLogging {
			pc.logModifiedStatements(originalStatements, parsingStatements)
		}
	} else {
		parsingStatements = statements.GetStatements()
	}

	parsedStatements, err := contextParser.ottlParser.parseStatements(parsingStatements)
	if err != nil {
		return *new(R), err
	}

	convertedStatements, err := contextParser.statementsConverter.call(
		reflect.ValueOf(pc),
		contextParser.ottlParser,
		context,
		statements,
		parsedStatements,
	)
	if err != nil {
		return *new(R), err
	}

	if convertedStatements.IsNil() {
		return *new(R), nil
	}

	return convertedStatements.Interface().(R), nil
}

func (pc *ParserCollection[R]) logModifiedStatements(originalStatements, modifiedStatements []string) {
	var fields []zap.Field
	for i, original := range originalStatements {
		if modifiedStatements[i] != original {
			statementKey := fmt.Sprintf("[%v]", i)
			fields = append(fields, zap.Dict(
				statementKey,
				zap.String("original", original),
				zap.String("modified", modifiedStatements[i])),
			)
		}
	}
	if len(fields) > 0 {
		pc.Settings.Logger.Info("one or more statements were modified to include their paths context, please rewrite them accordingly", zap.Dict("statements", fields...))
	}
}

func (pc *ParserCollection[R]) supportedContextNames() []string {
	contextsNames := make([]string, 0, len(pc.contextParsers))
	for k := range pc.contextParsers {
		contextsNames = append(contextsNames, k)
	}
	return contextsNames
}
