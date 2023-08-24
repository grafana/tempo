// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"

import (
	"context"

	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

// NewBoolExprForSpan creates a BoolExpr[ottlspan.TransformContext] that will return true if any of the given OTTL conditions evaluate to true.
// The passed in functions should use the ottlspan.TransformContext.
// If a function named `drop` is not present in the function map it will be added automatically so that parsing works as expected
func NewBoolExprForSpan(conditions []string, functions map[string]interface{}, errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlspan.TransformContext], error) {
	if _, ok := functions["drop"]; !ok {
		functions["drop"] = drop[ottlspan.TransformContext]
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
func NewBoolExprForSpanEvent(conditions []string, functions map[string]interface{}, errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlspanevent.TransformContext], error) {
	if _, ok := functions["drop"]; !ok {
		functions["drop"] = drop[ottlspanevent.TransformContext]
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
func NewBoolExprForMetric(conditions []string, functions map[string]interface{}, errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottlmetric.TransformContext], error) {
	if _, ok := functions["drop"]; !ok {
		functions["drop"] = drop[ottlmetric.TransformContext]
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
func NewBoolExprForDataPoint(conditions []string, functions map[string]interface{}, errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottldatapoint.TransformContext], error) {
	if _, ok := functions["drop"]; !ok {
		functions["drop"] = drop[ottldatapoint.TransformContext]
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
func NewBoolExprForLog(conditions []string, functions map[string]interface{}, errorMode ottl.ErrorMode, set component.TelemetrySettings) (expr.BoolExpr[ottllog.TransformContext], error) {
	if _, ok := functions["drop"]; !ok {
		functions["drop"] = drop[ottllog.TransformContext]
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

func conditionsToStatements(conditions []string) []string {
	statements := make([]string, len(conditions))
	for i, condition := range conditions {
		statements[i] = "drop() where " + condition
	}
	return statements
}

func StandardSpanFuncs() map[string]interface{} {
	return standardFuncs[ottlspan.TransformContext]()
}

func StandardSpanEventFuncs() map[string]interface{} {
	return standardFuncs[ottlspanevent.TransformContext]()
}

func StandardMetricFuncs() map[string]interface{} {
	return standardFuncs[ottlmetric.TransformContext]()
}

func StandardDataPointFuncs() map[string]interface{} {
	return standardFuncs[ottldatapoint.TransformContext]()
}

func StandardLogFuncs() map[string]interface{} {
	return standardFuncs[ottllog.TransformContext]()
}

func standardFuncs[K any]() map[string]interface{} {
	return map[string]interface{}{
		"TraceID":     ottlfuncs.TraceID[K],
		"SpanID":      ottlfuncs.SpanID[K],
		"IsMatch":     ottlfuncs.IsMatch[K],
		"Concat":      ottlfuncs.Concat[K],
		"Split":       ottlfuncs.Split[K],
		"Int":         ottlfuncs.Int[K],
		"ConvertCase": ottlfuncs.ConvertCase[K],
		"Substring":   ottlfuncs.Substring[K],
		"drop":        drop[K],
	}
}

func drop[K any]() (ottl.ExprFunc[K], error) {
	return func(context.Context, K) (interface{}, error) {
		return true, nil
	}, nil
}
