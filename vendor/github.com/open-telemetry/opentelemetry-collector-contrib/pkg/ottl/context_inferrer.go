// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"cmp"
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
	"resource",
	"scope",
	"instrumentation_scope",
}

// contextInferrer is an interface used to infer the OTTL context from statements.
type contextInferrer interface {
	// inferFromStatements returns the OTTL context inferred from the given statements.
	inferFromStatements(statements []string) (string, error)
	// inferFromConditions returns the OTTL context inferred from the given conditions.
	inferFromConditions(conditions []string) (string, error)
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
	return s.infer(conditions, s.getConditionHints)
}

func (s *priorityContextInferrer) inferFromStatements(statements []string) (inferredContext string, err error) {
	return s.infer(statements, s.getStatementHints)
}

// hinterFunc is used by the infer function to generate the hints (paths, functions, enums, etc.) for the given OTTL.
type hinterFunc func(string) ([]path, map[string]struct{}, map[enumSymbol]struct{}, error)

func (s *priorityContextInferrer) infer(ottls []string, hinter hinterFunc) (inferredContext string, err error) {
	s.telemetrySettings.Logger.Debug("Inferring context from OTTL",
		zap.Strings("candidates", maps.Keys(s.contextCandidate)),
		zap.Any("priority", s.contextPriority),
		zap.Strings("values", ottls),
	)

	defer func() {
		if inferredContext != "" {
			s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Inferred context: "%s"`, inferredContext))
		} else {
			s.telemetrySettings.Logger.Debug("Unable to infer context from statements")
		}
	}()

	requiredFunctions := map[string]struct{}{}
	requiredEnums := map[enumSymbol]struct{}{}

	var inferredContextPriority int
	for _, ottl := range ottls {
		ottlPaths, ottlFunctions, ottlEnums, err := hinter(ottl)
		if err != nil {
			return "", err
		}
		for _, p := range ottlPaths {
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
		for function := range ottlFunctions {
			requiredFunctions[function] = struct{}{}
		}
		for enum := range ottlEnums {
			requiredEnums[enum] = struct{}{}
		}
	}
	// No inferred context or nothing left to verify.
	if inferredContext == "" || (len(requiredFunctions) == 0 && len(requiredEnums) == 0) {
		s.telemetrySettings.Logger.Debug("No context candidate found in the ottls")
		return inferredContext, nil
	}
	ok := s.validateContextCandidate(inferredContext, requiredFunctions, requiredEnums)
	if ok {
		return inferredContext, nil
	}
	return s.inferFromLowerContexts(inferredContext, requiredFunctions, requiredEnums), nil
}

// validateContextCandidate checks if the given context candidate has all required functions names
// and enums symbols. The functions arity are not verified.
func (s *priorityContextInferrer) validateContextCandidate(
	context string,
	requiredFunctions map[string]struct{},
	requiredEnums map[enumSymbol]struct{},
) bool {
	s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Validating selected context candidate: "%s"`, context))
	candidate, ok := s.contextCandidate[context]
	if !ok {
		s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Context "%s" is not a valid candidate`, context))
		return false
	}
	if len(requiredFunctions) == 0 && len(requiredEnums) == 0 {
		return true
	}
	for function := range requiredFunctions {
		if !candidate.hasFunctionName(function) {
			s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Context "%s" does not meet the function requirement: "%s"`, context, function))
			return false
		}
	}
	for enum := range requiredEnums {
		if !candidate.hasEnumSymbol((*EnumSymbol)(&enum)) {
			s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Context "%s" does not meet the enum requirement: "%s"`, context, string(enum)))
			return false
		}
	}
	return true
}

// inferFromLowerContexts returns the first lower context that supports all required functions
// and enum symbols used on the statements.
// If no lower context meets the requirements, or if the context candidate is unknown, it
// returns an empty string.
func (s *priorityContextInferrer) inferFromLowerContexts(
	context string,
	requiredFunctions map[string]struct{},
	requiredEnums map[enumSymbol]struct{},
) string {
	s.telemetrySettings.Logger.Debug(fmt.Sprintf(`Trying to infer context using "%s" lower contexts`, context))
	inferredContextCandidate, ok := s.contextCandidate[context]
	if !ok {
		return ""
	}

	lowerContextCandidates := inferredContextCandidate.getLowerContexts(context)
	if len(lowerContextCandidates) == 0 {
		return ""
	}

	s.sortContextCandidates(lowerContextCandidates)
	for _, lowerCandidate := range lowerContextCandidates {
		ok = s.validateContextCandidate(lowerCandidate, requiredFunctions, requiredEnums)
		if ok {
			return lowerCandidate
		}
	}
	return ""
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

// getConditionHints extracts all path, function names (editor and converter), and enumSymbol
// from the given condition. These values are used by the context inferrer as hints to
// select a context in which the function/enum are supported.
func (s *priorityContextInferrer) getConditionHints(condition string) ([]path, map[string]struct{}, map[enumSymbol]struct{}, error) {
	parsed, err := parseCondition(condition)
	if err != nil {
		return nil, nil, nil, err
	}

	visitor := newGrammarContextInferrerVisitor()
	parsed.accept(&visitor)
	return visitor.paths, visitor.functions, visitor.enumsSymbols, nil
}

// getStatementHints extracts all path, function names (editor and converter), and enumSymbol
// from the given statement. These values are used by the context inferrer as hints to
// select a context in which the function/enum are supported.
func (s *priorityContextInferrer) getStatementHints(statement string) ([]path, map[string]struct{}, map[enumSymbol]struct{}, error) {
	parsed, err := parseStatement(statement)
	if err != nil {
		return nil, nil, nil, err
	}
	visitor := newGrammarContextInferrerVisitor()
	parsed.Editor.accept(&visitor)
	if parsed.WhereClause != nil {
		parsed.WhereClause.accept(&visitor)
	}
	return visitor.paths, visitor.functions, visitor.enumsSymbols, nil
}

// priorityContextInferrerHintsVisitor is a grammarVisitor implementation that collects
// all path, function names (converter.Function and editor.Function), and enumSymbol.
type priorityContextInferrerHintsVisitor struct {
	paths        []path
	functions    map[string]struct{}
	enumsSymbols map[enumSymbol]struct{}
}

func newGrammarContextInferrerVisitor() priorityContextInferrerHintsVisitor {
	return priorityContextInferrerHintsVisitor{
		paths:        []path{},
		functions:    make(map[string]struct{}),
		enumsSymbols: make(map[enumSymbol]struct{}),
	}
}

func (v *priorityContextInferrerHintsVisitor) visitMathExprLiteral(_ *mathExprLiteral) {}

func (v *priorityContextInferrerHintsVisitor) visitEditor(e *editor) {
	v.functions[e.Function] = struct{}{}
}

func (v *priorityContextInferrerHintsVisitor) visitConverter(c *converter) {
	v.functions[c.Function] = struct{}{}
}

func (v *priorityContextInferrerHintsVisitor) visitValue(va *value) {
	if va.Enum != nil {
		v.enumsSymbols[*va.Enum] = struct{}{}
	}
}

func (v *priorityContextInferrerHintsVisitor) visitPath(value *path) {
	v.paths = append(v.paths, *value)
}
