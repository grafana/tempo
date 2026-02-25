// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
)

// boolExpr represents a condition in OTTL
type boolExpr[K any] interface {
	Eval(ctx context.Context, tCtx K) (bool, error)

	unexported()
}

type literalBoolExpr[K any] = literalExpr[K, bool]

func newAlwaysTrue[K any]() boolExpr[K] {
	return newLiteralExpr[K](true)
}

func newAlwaysFalse[K any]() boolExpr[K] {
	return newLiteralExpr[K](false)
}

func newNot[K any](expr boolExpr[K]) boolExpr[K] {
	if f, ok := expr.(*literalBoolExpr[K]); ok {
		return newLiteralExpr[K](!f.getValue())
	}

	return &notBoolExpr[K]{expr: expr}
}

type notBoolExpr[K any] struct {
	expr boolExpr[K]
}

func (*notBoolExpr[K]) unexported() {}

// Eval evaluates an OTTL condition
func (e *notBoolExpr[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	val, err := e.expr.Eval(ctx, tCtx)
	return !val, err
}

// newAndExprs a boolExpr that returns a short-circuited result of ANDing
// boolExpressionEvaluator funcs
func newAndExprs[K any](exprs []boolExpr[K]) boolExpr[K] {
	newExprs := make([]boolExpr[K], 0, len(exprs))
	for i := range exprs {
		// If any literal evaluates to false, we can simply return false.
		// If a literal evaluates to true, it won't affect the expression's outcome, so we can skip evaluating it again.
		if f, ok := exprs[i].(*literalBoolExpr[K]); ok {
			if !f.getValue() {
				return newAlwaysFalse[K]()
			}
			continue
		}
		newExprs = append(newExprs, exprs[i])
	}

	// All expressions evaluated to true, just return literal true.
	if len(newExprs) == 0 {
		return newAlwaysTrue[K]()
	}

	// One expression left, no need to wrap in "andExprs".
	if len(newExprs) == 1 {
		return newExprs[0]
	}

	return &andExprs[K]{exprs: newExprs}
}

type andExprs[K any] struct {
	exprs []boolExpr[K]
}

func (*andExprs[K]) unexported() {}

// Eval evaluates an OTTL condition
func (e *andExprs[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	for _, f := range e.exprs {
		result, err := f.Eval(ctx, tCtx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// newOrExprs a boolExpr that returns a short-circuited result of ORing
// boolExpressionEvaluator funcs
func newOrExprs[K any](exprs []boolExpr[K]) boolExpr[K] {
	newExprs := make([]boolExpr[K], 0, len(exprs))
	// If any literal evaluates to true, we can simply return true.
	// If a literal evaluates to false, it won't affect the expression's outcome, so we can skip evaluating it again.
	for i := range exprs {
		if f, ok := exprs[i].(*literalBoolExpr[K]); ok {
			if f.getValue() {
				return newAlwaysTrue[K]()
			}
			continue
		}
		newExprs = append(newExprs, exprs[i])
	}

	// All expressions evaluated to false, just return literal false.
	if len(newExprs) == 0 {
		return newAlwaysFalse[K]()
	}

	// One expression left, no need to wrap in "orExprs".
	if len(newExprs) == 1 {
		return newExprs[0]
	}

	return &orExprs[K]{exprs: newExprs}
}

type orExprs[K any] struct {
	exprs []boolExpr[K]
}

func (*orExprs[K]) unexported() {}

// Eval evaluates an OTTL condition
func (e *orExprs[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	for _, f := range e.exprs {
		result, err := f.Eval(ctx, tCtx)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (p *Parser[K]) newComparisonExpr(comparison *comparison) (boolExpr[K], error) {
	if comparison == nil {
		return newAlwaysTrue[K](), nil
	}
	left, err := p.newGetter(comparison.Left)
	if err != nil {
		return nil, err
	}
	right, err := p.newGetter(comparison.Right)
	if err != nil {
		return nil, err
	}

	comparator := NewValueComparator()
	if leftVal, leftOk := GetLiteralValue(left); leftOk {
		if rightVal, rightOk := GetLiteralValue(right); rightOk {
			return newLiteralExpr[K](comparator.compare(leftVal, rightVal, comparison.Op)), nil
		}
	}

	// The parser ensures that we'll never get an invalid comparison.Op, so we don't have to check that case.
	return &comparisonExpr[K]{left: left, right: right, comparator: comparator, op: comparison.Op}, nil
}

type comparisonExpr[K any] struct {
	left, right Getter[K]
	comparator  ValueComparator
	op          compareOp
}

func (*comparisonExpr[K]) unexported() {}

func (e *comparisonExpr[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	a, leftErr := e.left.Get(ctx, tCtx)
	if leftErr != nil {
		return false, leftErr
	}
	b, rightErr := e.right.Get(ctx, tCtx)
	if rightErr != nil {
		return false, rightErr
	}
	return e.comparator.compare(a, b, e.op), nil
}

func (p *Parser[K]) newBoolExpr(expr *booleanExpression) (boolExpr[K], error) {
	if expr == nil {
		return newAlwaysTrue[K](), nil
	}
	f, err := p.newBooleanTermEvaluator(expr.Left)
	if err != nil {
		return nil, err
	}
	funcs := []boolExpr[K]{f}
	for _, rhs := range expr.Right {
		f, err = p.newBooleanTermEvaluator(rhs.Term)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}

	return newOrExprs(funcs), nil
}

func (p *Parser[K]) newBooleanTermEvaluator(term *term) (boolExpr[K], error) {
	if term == nil {
		return newAlwaysTrue[K](), nil
	}
	f, err := p.newBooleanValueEvaluator(term.Left)
	if err != nil {
		return nil, err
	}
	funcs := []boolExpr[K]{f}
	for _, rhs := range term.Right {
		f, err = p.newBooleanValueEvaluator(rhs.Value)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}

	return newAndExprs(funcs), nil
}

func (p *Parser[K]) newBooleanValueEvaluator(value *booleanValue) (boolExpr[K], error) {
	if value == nil {
		return newAlwaysTrue[K](), nil
	}

	var boolExpr boolExpr[K]
	var err error
	switch {
	case value.Comparison != nil:
		boolExpr, err = p.newComparisonExpr(value.Comparison)
		if err != nil {
			return nil, err
		}
	case value.ConstExpr != nil:
		switch {
		case value.ConstExpr.Boolean != nil:
			if *value.ConstExpr.Boolean {
				boolExpr = newAlwaysTrue[K]()
			} else {
				boolExpr = newAlwaysFalse[K]()
			}
		case value.ConstExpr.Converter != nil:
			boolExpr, err = p.newConverterEvaluator(*value.ConstExpr.Converter)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unhandled boolean operation %v", value)
		}
	case value.SubExpr != nil:
		boolExpr, err = p.newBoolExpr(value.SubExpr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unhandled boolean operation %v", value)
	}

	if value.Negation != nil {
		return newNot(boolExpr), nil
	}
	return boolExpr, nil
}

func (p *Parser[K]) newConverterEvaluator(c converter) (boolExpr[K], error) {
	getter, err := p.newGetterFromConverter(c)
	if err != nil {
		return nil, err
	}

	return newConverterExpr(getter)
}

func newConverterExpr[K any](getter Getter[K]) (boolExpr[K], error) {
	if val, ok := GetLiteralValue(getter); ok {
		boolResult, okResult := val.(bool)
		if !okResult {
			return nil, fmt.Errorf("value returned from Converter in constant expression must be bool but got %T", val)
		}
		return newLiteralExpr[K](boolResult), nil
	}

	return &converterExpr[K]{getter: getter}, nil
}

type converterExpr[K any] struct {
	getter Getter[K]
}

func (*converterExpr[K]) unexported() {}

func (e *converterExpr[K]) Eval(ctx context.Context, tCtx K) (bool, error) {
	result, err := e.getter.Get(ctx, tCtx)
	if err != nil {
		return false, err
	}
	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("value returned from Converter in constant expression must be bool but got %T", result)
	}
	return boolResult, nil
}
