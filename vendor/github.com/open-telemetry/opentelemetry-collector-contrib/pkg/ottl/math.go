// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func (p *Parser[K]) evaluateMathExpression(expr *mathExpression) (Getter[K], error) {
	mainGetter, err := p.evaluateAddSubTerm(expr.Left)
	if err != nil {
		return nil, err
	}
	for _, rhs := range expr.Right {
		getter, err := p.evaluateAddSubTerm(rhs.Term)
		if err != nil {
			return nil, err
		}
		mainGetter = attemptMathOperation(mainGetter, rhs.Operator, getter)
	}

	return mainGetter, nil
}

func (p *Parser[K]) evaluateAddSubTerm(term *addSubTerm) (Getter[K], error) {
	mainGetter, err := p.evaluateMathValue(term.Left)
	if err != nil {
		return nil, err
	}
	for _, rhs := range term.Right {
		getter, err := p.evaluateMathValue(rhs.Value)
		if err != nil {
			return nil, err
		}
		mainGetter = attemptMathOperation(mainGetter, rhs.Operator, getter)
	}

	return mainGetter, nil
}

func (p *Parser[K]) evaluateMathValue(val *mathValue) (Getter[K], error) {
	// we have to handle unary plus/minus here because the lexer cannot
	// differentiate between binary and unary operators
	switch {
	case val.UnaryOp != nil && *val.UnaryOp == sub && val.Literal != nil:
		// If the literal is numeric, fold the sign into the literal
		if val.Literal.Float != nil {
			neg := -(*val.Literal.Float)
			newLit := &mathExprLiteral{Float: &neg}
			return p.newGetter(value{Literal: newLit})
		}
		if val.Literal.Int != nil {
			neg := -(*val.Literal.Int)
			newLit := &mathExprLiteral{Int: &neg}
			return p.newGetter(value{Literal: newLit})
		}
		// Non-numeric literals fall back to dynamic negation
		baseGetter, err := p.newGetter(value{Literal: val.Literal})
		if err != nil {
			return nil, err
		}
		return negateGetter(baseGetter), nil
	case val.UnaryOp != nil && *val.UnaryOp == sub && val.SubExpression != nil:
		baseGetter, err := p.evaluateMathExpression(val.SubExpression)
		if err != nil {
			return nil, err
		}
		return negateGetter(baseGetter), nil
	case val.UnaryOp != nil && *val.UnaryOp == add && val.Literal != nil:
		// Unary plus: no-op to be explicit about it
		return p.newGetter(value{Literal: val.Literal})
	case val.UnaryOp != nil && *val.UnaryOp == add && val.SubExpression != nil:
		// Unary plus: no-op to be explicit about it
		return p.evaluateMathExpression(val.SubExpression)
	case val.Literal != nil:
		return p.newGetter(value{Literal: val.Literal})
	case val.SubExpression != nil:
		return p.evaluateMathExpression(val.SubExpression)
	default:
		return nil, fmt.Errorf("unsupported mathematical value %v", val)
	}
}

func negateGetter[K any](baseGetter Getter[K]) Getter[K] {
	return &exprGetter[K]{
		expr: Expr[K]{
			exprFunc: func(ctx context.Context, tCtx K) (any, error) {
				x, err := baseGetter.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				switch v := x.(type) {
				case int64:
					return -v, nil
				case float64:
					return -v, nil
				default:
					return nil, fmt.Errorf("unsupported unary minus operation on type %T", x)
				}
			},
		},
	}
}

func attemptMathOperation[K any](lhs Getter[K], op mathOp, rhs Getter[K]) Getter[K] {
	return &exprGetter[K]{
		expr: Expr[K]{
			exprFunc: func(ctx context.Context, tCtx K) (any, error) {
				x, err := lhs.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				y, err := rhs.Get(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				switch newX := x.(type) {
				case int64:
					switch newY := y.(type) {
					case int64:
						result, err := performOp[int64](newX, newY, op)
						if err != nil {
							return nil, err
						}
						return result, nil
					case float64:
						result, err := performOp[float64](float64(newX), newY, op)
						if err != nil {
							return nil, err
						}
						return result, nil
					default:
						return nil, fmt.Errorf("%v must be int64 or float64", y)
					}
				case float64:
					switch newY := y.(type) {
					case int64:
						result, err := performOp[float64](newX, float64(newY), op)
						if err != nil {
							return nil, err
						}
						return result, nil
					case float64:
						result, err := performOp[float64](newX, newY, op)
						if err != nil {
							return nil, err
						}
						return result, nil
					default:
						return nil, fmt.Errorf("%v must be int64 or float64", y)
					}
				case time.Time:
					return performOpTime(newX, y, op)
				case time.Duration:
					return performOpDuration(newX, y, op)
				default:
					return nil, fmt.Errorf("%v must be int64, float64, time.Time or time.Duration", x)
				}
			},
		},
	}
}

func performOpTime(x time.Time, y any, op mathOp) (any, error) {
	switch op {
	case add:
		switch newY := y.(type) {
		case time.Duration:
			result := x.Add(newY)
			return result, nil
		default:
			return nil, fmt.Errorf("time.Time must be added to time.Duration; found %v instead", y)
		}
	case sub:
		switch newY := y.(type) {
		case time.Time:
			result := x.Sub(newY)
			return result, nil
		case time.Duration:
			result := x.Add(-1 * newY)
			return result, nil
		default:
			return nil, fmt.Errorf("time.Time or time.Duration must be subtracted from time.Time; found %v instead", y)
		}
	}
	return nil, errors.New("only addition and subtraction supported for time.Time and time.Duration")
}

func performOpDuration(x time.Duration, y any, op mathOp) (any, error) {
	switch op {
	case add:
		switch newY := y.(type) {
		case time.Duration:
			result := x + newY
			return result, nil
		case time.Time:
			result := newY.Add(x)
			return result, nil
		default:
			return nil, fmt.Errorf("time.Duration must be added to time.Duration or time.Time; found %v instead", y)
		}
	case sub:
		switch newY := y.(type) {
		case time.Duration:
			result := x - newY
			return result, nil
		default:
			return nil, fmt.Errorf("time.Duration must be subtracted from time.Duration; found %v instead", y)
		}
	}
	return nil, errors.New("only addition and subtraction supported for time.Time and time.Duration")
}

func performOp[N int64 | float64](x, y N, op mathOp) (N, error) {
	switch op {
	case add:
		return x + y, nil
	case sub:
		return x - y, nil
	case mult:
		return x * y, nil
	case div:
		if y == 0 {
			return 0, errors.New("attempted to divide by 0")
		}
		return x / y, nil
	}
	return 0, fmt.Errorf("invalid operation %v", op)
}
