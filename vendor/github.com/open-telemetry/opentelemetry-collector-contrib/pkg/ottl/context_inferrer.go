// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var defaultContextInferPriority = []string{
	"log",
	"datapoint",
	"metric",
	"spanevent",
	"span",
	"profile",
	"scope",
	"instrumentation_scope",
	"resource",
}

// contextInferrer is an interface used to infer the OTTL context from statements.
type contextInferrer interface {
	// inferFromStatements returns the OTTL context inferred from the given statements.
	inferFromStatements(statements []string) (string, error)
	// inferFromConditions returns the OTTL context inferred from the given conditions.
	inferFromConditions(conditions []string) (string, error)
	// infer returns the OTTL context inferred from the given statements and conditions.
	infer(statements []string, conditions []string) (string, error)
}

type priorityContextInferrer struct {
	telemetrySettings component.TelemetrySettings
	contextPriority   map[string]int
	contextCandidate  map[string]*priorityContextInferrerCandidate
}

type priorityContextInferrerCandidate struct {
	hasEnumSymbol    func(enum *EnumSymbol) bool
	hasFunctionName  func(name string) bool
	getLowerContexts func(context string) []string
}

type priorityContextInferrerOption func(*priorityContextInferrer)

// newPriorityContextInferrer creates a new priority-based context inferrer. To infer the context,
// it uses a slice of priorities (withContextInferrerPriorities) and a set of hints extracted from
// the parsed statements.
//
// To be eligible, a context must support all functions and enums symbols present on the statements.
// If the path context with the highest priority does not meet this requirement, it falls back to its
// lower contexts, testing them with the same logic and choosing the first one that meets all requirements.
//
// If non-prioritized contexts are found on the statements, they get assigned the lowest possible priority,
// and are only selected if no other prioritized context is found.
func newPriorityContextInferrer(telemetrySettings component.TelemetrySettings, contextsCandidate map[string]*priorityContextInferrerCandidate, options ...priorityContextInferrerOption) contextInferrer {
	c := &priorityContextInferrer{
		telemetrySettings: telemetrySettings,
		contextCandidate:  contextsCandidate,
	}
	for _, opt := range options {
		opt(c)
	}
	if len(c.contextPriority) == 0 {
		withContextInferrerPriorities(defaultContextInferPriority)(c)
	}
	return c
}

// withContextInferrerPriorities sets the contexts candidates priorities. The lower the
// context position is in the array, the more priority it will have over other items.
func withContextInferrerPriorities(priorities []string) priorityContextInferrerOption {
	return func(c *priorityContextInferrer) {
		contextPriority := map[string]int{}
		for pri, context := range priorities {
			contextPriority[context] = pri
		}
		c.contextPriority = contextPriority
	}
}

func (s *priorityContextInferrer) inferFromConditions(conditions []string) (inferredContext string, err error) {
	return s.infer(nil, conditions)
}

func (s *priorityContextInferrer) inferFromStatements(statements []string) (inferredContext string, err error) {
	return s.infer(statements, nil)
}

func (s *priorityContextInferrer) infer(statements []string, conditions []string) (inferredContext string, err error) {
	var statementsHints, conditionsHints []priorityContextInferrerHints
	if len(statements) > 0 {
		statementsHints, err = s.getStatementsHints(statements)
		if err != nil {
			return "", err
		}
	}
	if len(conditions) > 0 {
		conditionsHints, err = s.getConditionsHints(conditions)
		if err != nil {
			return "", err
		}
	}
	if s.telemetrySettings.Logger.Core().Enabled(zap.DebugLevel) {
		s.telemetrySettings.Logger.Debug("Inferring context from statements and conditions",
			zap.Strings("candidates", maps.Keys(s.contextCandidate)),
			zap.Any("priority", s.contextPriority),
			zap.Strings("statements", statements),
			zap.Strings("conditions", conditions),
		)
	}
	return s.inferFromHints(append(statementsHints, conditionsHints...))
}

func (s *priorityContextInferrer) inferFromHints(hints []priorityContextInferrerHints) (inferredContext string, err error) {
	defer func() {
		if inferredContext != "" {
			s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Inferred context: "%s"`, inferredContext))
		} else {
			s.telemetrySettings.Logger.Debug("Unable to infer context from statements", zap.Error(err))
		}
	}()

	requiredFunctions := map[string]struct{}{}
	requiredEnums := map[enumSymbol]struct{}{}

	var inferredContextPriority int
	for _, hint := range hints {
		for _, p := range hint.paths {
			candidate := p.Context
			candidatePriority, ok := s.contextPriority[candidate]
			if !ok {
				candidatePriority = math.MaxInt
			}
			if inferredContext == "" || candidatePriority < inferredContextPriority {
				inferredContext = candidate
				inferredContextPriority = candidatePriority
			}
		}
		for function := range hint.functions {
			requiredFunctions[function] = struct{}{}
		}
		for enum := range hint.enumsSymbols {
			requiredEnums[enum] = struct{}{}
		}
	}
	// No inferred context or nothing left to verify.
	if inferredContext == "" || (len(requiredFunctions) == 0 && len(requiredEnums) == 0) {
		s.telemetrySettings.Logger.Debug("No context candidate found in the ottls")
		return inferredContext, nil
	}
	if err = s.validateContextCandidate(inferredContext, requiredFunctions, requiredEnums); err == nil {
		return inferredContext, nil
	}
	if inferredFromLowerContexts, lowerContextErr := s.inferFromLowerContexts(inferredContext, requiredFunctions, requiredEnums); lowerContextErr == nil {
		return inferredFromLowerContexts, nil
	}
	return "", err
}

// validateContextCandidate checks if the given context candidate has all required functions names
// and enums symbols. The functions arity are not verified.
func (s *priorityContextInferrer) validateContextCandidate(
	context string,
	requiredFunctions map[string]struct{},
	requiredEnums map[enumSymbol]struct{},
) error {
	s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Validating selected context candidate: "%s"`, context))
	candidate, ok := s.contextCandidate[context]
	if !ok {
		return fmt.Errorf(`inferred context "%s" is not a valid candidate`, context)
	}
	if len(requiredFunctions) == 0 && len(requiredEnums) == 0 {
		return nil
	}
	for function := range requiredFunctions {
		if !candidate.hasFunctionName(function) {
			return fmt.Errorf(`inferred context "%s" does not support the function "%s"`, context, function)
		}
	}
	for enum := range requiredEnums {
		if !candidate.hasEnumSymbol((*EnumSymbol)(&enum)) {
			return fmt.Errorf(`inferred context "%s" does not support the enum symbol "%s"`, context, string(enum))
		}
	}
	return nil
}

// inferFromLowerContexts returns the first lower context that supports all required functions
// and enum symbols used on the statements.
// If no lower context meets the requirements, or if the context candidate is unknown, it
// returns an empty string.
func (s *priorityContextInferrer) inferFromLowerContexts(
	context string,
	requiredFunctions map[string]struct{},
	requiredEnums map[enumSymbol]struct{},
) (inferredContext string, err error) {
	s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Trying to infer context using "%s" lower contexts`, context))

	defer func() {
		if err != nil {
			s.telemetrySettings.Logger.Debug("Unable to infer context from lower contexts", zap.Error(err))
		}
	}()

	inferredContextCandidate, ok := s.contextCandidate[context]
	if !ok {
		return "", fmt.Errorf(`context "%s" is not a valid candidate`, context)
	}

	lowerContextCandidates := inferredContextCandidate.getLowerContexts(context)
	if len(lowerContextCandidates) == 0 {
		return "", fmt.Errorf(`context "%s" has no lower contexts candidates`, context)
	}

	s.sortContextCandidates(lowerContextCandidates)
	for _, lowerCandidate := range lowerContextCandidates {
		if candidateErr := s.validateContextCandidate(lowerCandidate, requiredFunctions, requiredEnums); candidateErr == nil {
			return lowerCandidate, nil
		}
		s.telemetrySettings.Logger.Debug(fmt.Sprintf(`lower context "%s" is not a valid candidate`, lowerCandidate), zap.Error(err))
	}
	return "", errors.New("no valid lower context found")
}

// sortContextCandidates sorts the slice candidates using the priorityContextInferrer.contextsPriority order.
func (s *priorityContextInferrer) sortContextCandidates(candidates []string) {
	slices.SortFunc(candidates, func(l, r string) int {
		lp, ok := s.contextPriority[l]
		if !ok {
			lp = math.MaxInt
		}
		rp, ok := s.contextPriority[r]
		if !ok {
			rp = math.MaxInt
		}
		return cmp.Compare(lp, rp)
	})
}

// getConditionsHints extracts all path, function names (editor and converter), and enumSymbol
// from the given condition. These values are used by the context inferrer as hints to
// select a context in which the function/enum are supported.
func (s *priorityContextInferrer) getConditionsHints(conditions []string) ([]priorityContextInferrerHints, error) {
	hints := make([]priorityContextInferrerHints, 0, len(conditions))
	for _, condition := range conditions {
		parsed, err := parseCondition(condition)
		if err != nil {
			return nil, err
		}

		visitor := newGrammarContextInferrerVisitor()
		parsed.accept(&visitor)
		hints = append(hints, visitor)
	}
	return hints, nil
}

// getStatementsHints extracts all path, function names (editor and converter), and enumSymbol
// from the given statement. These values are used by the context inferrer as hints to
// select a context in which the function/enum are supported.
func (s *priorityContextInferrer) getStatementsHints(statements []string) ([]priorityContextInferrerHints, error) {
	hints := make([]priorityContextInferrerHints, 0, len(statements))
	for _, statement := range statements {
		parsed, err := parseStatement(statement)
		if err != nil {
			return nil, err
		}
		visitor := newGrammarContextInferrerVisitor()
		parsed.Editor.accept(&visitor)
		if parsed.WhereClause != nil {
			parsed.WhereClause.accept(&visitor)
		}
		hints = append(hints, visitor)
	}
	return hints, nil
}

// priorityContextInferrerHints is a grammarVisitor implementation that collects
// all path, function names (converter.Function and editor.Function), and enumSymbol.
type priorityContextInferrerHints struct {
	paths        []path
	functions    map[string]struct{}
	enumsSymbols map[enumSymbol]struct{}
}

func newGrammarContextInferrerVisitor() priorityContextInferrerHints {
	return priorityContextInferrerHints{
		paths:        []path{},
		functions:    make(map[string]struct{}),
		enumsSymbols: make(map[enumSymbol]struct{}),
	}
}

func (v *priorityContextInferrerHints) visitMathExprLiteral(_ *mathExprLiteral) {}

func (v *priorityContextInferrerHints) visitEditor(e *editor) {
	v.functions[e.Function] = struct{}{}
}

func (v *priorityContextInferrerHints) visitConverter(c *converter) {
	v.functions[c.Function] = struct{}{}
}

func (v *priorityContextInferrerHints) visitValue(va *value) {
	if va.Enum != nil {
		v.enumsSymbols[*va.Enum] = struct{}{}
	}
}

func (v *priorityContextInferrerHints) visitPath(value *path) {
	v.paths = append(v.paths, *value)
}
