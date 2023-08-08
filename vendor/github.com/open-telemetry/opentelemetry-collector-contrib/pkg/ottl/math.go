// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
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
	switch {
	case val.Literal != nil:
		return p.newGetter(value{Literal: val.Literal})
	case val.SubExpression != nil:
		return p.evaluateMathExpression(val.SubExpression)
	}

	return nil, fmt.Errorf("unsupported mathematical value %v", val)
}

func attemptMathOperation[K any](lhs Getter[K], op mathOp, rhs Getter[K]) Getter[K] {
	return exprGetter[K]{
		expr: Expr[K]{
			exprFunc: func(ctx context.Context, tCtx K) (interface{}, error) {
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
				default:
					return nil, fmt.Errorf("%v must be int64 or float64", x)
				}
			},
		},
	}
}

func performOp[N int64 | float64](x N, y N, op mathOp) (N, error) {
	switch op {
	case ADD:
		return x + y, nil
	case SUB:
		return x - y, nil
	case MULT:
		return x * y, nil
	case DIV:
		if y == 0 {
			return 0, fmt.Errorf("attempted to divide by 0")
		}
		return x / y, nil
	}
	return 0, fmt.Errorf("invalid operation %v", op)
}
