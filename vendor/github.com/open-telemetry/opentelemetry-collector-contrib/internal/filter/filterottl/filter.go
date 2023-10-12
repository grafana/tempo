// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
)

// NewBoolExprForSpan creates a BoolExpr[ottlspan.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottlspan.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForSpan(conditions []string, functions map[string]ottl.Factory[ottlspan.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlspan.TransformContext], error) {
	drop := newDropFactory[ottlspan.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottlspan.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottlspan.NewStatements(statements, set, ottlspan.WithErrorMode(errorMode))
	return &s, nil
}

// NewBoolExprForSpanEvent creates a BoolExpr[ottlspanevent.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottlspanevent.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForSpanEvent(conditions []string, functions map[string]ottl.Factory[ottlspanevent.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlspanevent.TransformContext], error) {
	drop := newDropFactory[ottlspanevent.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottlspanevent.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottlspanevent.NewStatements(statements, set, ottlspanevent.WithErrorMode(errorMode))
	return &s, nil
}

// NewBoolExprForMetric creates a BoolExpr[ottlmetric.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottlmetric.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForMetric(conditions []string, functions map[string]ottl.Factory[ottlmetric.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlmetric.TransformContext], error) {
	drop := newDropFactory[ottlmetric.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottlmetric.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottlmetric.NewStatements(statements, set, ottlmetric.WithErrorMode(errorMode))
	return &s, nil
}

// NewBoolExprForDataPoint creates a BoolExpr[ottldatapoint.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottldatapoint.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForDataPoint(conditions []string, functions map[string]ottl.Factory[ottldatapoint.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottldatapoint.TransformContext], error) {
	drop := newDropFactory[ottldatapoint.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottldatapoint.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottldatapoint.NewStatements(statements, set, ottldatapoint.WithErrorMode(errorMode))
	return &s, nil
}

// NewBoolExprForLog creates a BoolExpr[ottllog.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottllog.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForLog(conditions []string, functions map[string]ottl.Factory[ottllog.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottllog.TransformContext], error) {
	drop := newDropFactory[ottllog.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottllog.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottllog.NewStatements(statements, set, ottllog.WithErrorMode(errorMode))
	return &s, nil
}

// NewBoolExprForResource creates a BoolExpr[ottlresource.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottlresource.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForResource(conditions []string, functions map[string]ottl.Factory[ottlresource.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlresource.TransformContext], error) {
	drop := newDropFactory[ottlresource.TransformContext]()
	if _, ok := functions[drop.Name()]; !ok {
		functions[drop.Name()] = drop
	}
	statmentsStr := conditionsToStatements(conditions)
	parser, err := ottlresource.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseStatements(statmentsStr)
	if err != nil {
		return nil, err
	}
	s := ottlresource.NewStatements(statements, set, ottlresource.WithErrorMode(errorMode))
	return &s, nil
}

func conditionsToStatements(conditions []string) []string {
	statements := make([]string, len(conditions))
	for i, condition := range conditions {
		statements[i] = "drop() where " + condition
	}
	return statements
}
