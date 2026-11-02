// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
)

// LambdaExpression is a parsed OTTL lambda expression. OTTL functions may accept it as an argument.
// For each outer invocation, call [LambdaExpression.Activate] with the evaluation context and the
// number of arguments to bind, so the [LambdaExpression.Formals] length must match it. Use
// [LambdaActivation.SetArg] to bind positional arguments, [LambdaActivation.Eval] to run the body
// (possibly multiple times with different arguments), and [LambdaActivation.Close] when finished.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
type LambdaExpression[K any] struct {
	formals        []LocalIdentifierDecl
	body           Getter[K] // mutually exclusive with bodyExpr
	bodyExpr       boolExpr[K]
	activationPool *sync.Pool
	arityValidated *atomic.Bool
}

// newLambdaExpression creates a new LambdaExpression. It must either have a body or a bodyExpr, but not both.
func newLambdaExpression[K any](formals []LocalIdentifierDecl, body Getter[K], bodyExpr boolExpr[K]) *LambdaExpression[K] {
	v := &LambdaExpression[K]{
		formals:  formals,
		body:     body,
		bodyExpr: bodyExpr,
	}
	nonBlankFormals := countNonBlankIdentifiers(formals)
	v.activationPool = &sync.Pool{
		New: func() any {
			return &LambdaActivation[K]{
				expr:       v,
				argValues:  make([]any, len(formals)),
				activation: &localActivation{bindings: make(map[string]any, nonBlankFormals)},
			}
		},
	}
	v.arityValidated = &atomic.Bool{}
	return v
}

// Formals returns a copy of the lambda's formal parameters in declaration order (left to right).
// Blank ("_") placeholders are included.
func (l *LambdaExpression[K]) Formals() []LocalIdentifierDecl {
	return slices.Clone(l.formals)
}

// ValidateArity checks that the number of arguments that will be passed to the
// lambda matches the number of declared formals. If the counts differ, an error
// is returned. This allows statically verifying arity: it should be run inside
// the OTTL function factory, i.e. outside the closure the factory returns. It's
// meant to verify the lambda passed by the user has the correct number of
// formals before calling [LambdaExpression.Activate].
//
// While this is only intended to be called once, if it is called with an
// invalid arity after being called with a valid arity, the lambda will be
// marked as needing validation again, and [LambdaExpression.Activate] will
// return an error until [LambdaExpression.ValidateArity] is called with a valid
// arity.
func (l *LambdaExpression[K]) ValidateArity(arity int) error {
	if len(l.formals) != arity {
		l.arityValidated.Store(false)
		return fmt.Errorf("lambda should be defined with exactly %d formal(s), but has %d", arity, len(l.formals))
	}
	l.arityValidated.Store(true)
	return nil
}

// Activate creates a [LambdaActivation] for a single outer function invocation, allocating
// its own activation and argument storage, linking the resulting activation to the given ctx.
// Call [LambdaActivation.SetArg] for every index in (0...arity) before each [LambdaActivation.Eval],
// and then [LambdaActivation.Close] on the returned activation when it is no longer needed.
//
// [LambdaExpression.ValidateArity] must be called successfully before Activate; otherwise Activate
// returns an error. ValidateArity is meant to run once in the OTTL function factory, while Activate
// runs inside the closure the factory returns.
func (l *LambdaExpression[K]) Activate(ctx context.Context) (*LambdaActivation[K], error) {
	if !l.arityValidated.Load() {
		return nil, errors.New("lambda arity was not validated: ValidateArity must be called before Activate")
	}
	v := l.activationPool.Get().(*LambdaActivation[K])
	v.ctx = pushLocalActivation(ctx, v.activation)
	return v, nil
}

// LambdaActivation is a local activation of a [LambdaExpression] produced by [LambdaExpression.Activate].
type LambdaActivation[K any] struct {
	expr       *LambdaExpression[K]
	ctx        context.Context
	argValues  []any
	activation *localActivation
}

// SetArg sets the i-th positional argument for the next [LambdaActivation.Eval] call.
func (l *LambdaActivation[K]) SetArg(i int, v any) error {
	if i < 0 || i >= len(l.argValues) {
		return fmt.Errorf("argument index %d out of range (len=%d)", i, len(l.argValues))
	}
	l.argValues[i] = v
	return nil
}

// IsArgBound reports whether the i-th argument is bound to a named formal parameter in the
// lambda, meaning it must be explicitly set via [LambdaActivation.SetArg] before calling
// [LambdaActivation.Eval]. Arguments whose formal is declared as a blank identifier ("_")
// are discarded and do not need to be set. Because activations are reused across invocations,
// skipping SetArg for a bound argument may produce stale values from a prior call.
// Panics if i is out of range.
func (l *LambdaActivation[K]) IsArgBound(i int) bool {
	return !l.expr.formals[i].IsBlank()
}

// Eval runs the lambda with positional arguments set via [LambdaActivation.SetArg].
// The result type follows the lambda body (value or boolean sub-expression) evaluation and
// may be nil if the body evaluates to nil.
func (l *LambdaActivation[K]) Eval(tCtx K) (any, error) {
	if v, ok := l.expr.getLiteralValue(); ok {
		return v, nil
	}
	l.bindArguments()
	return l.evalBody(tCtx)
}

func (l *LambdaActivation[K]) bindArguments() {
	for i, formal := range l.expr.formals {
		if formal.IsBlank() {
			continue
		}
		l.activation.bindings[formal.Name()] = l.argValues[i]
	}
}

func (l *LambdaActivation[K]) evalBody(tCtx K) (any, error) {
	switch {
	case l.expr.bodyExpr != nil:
		return l.expr.bodyExpr.Eval(l.ctx, tCtx)
	case l.expr.body != nil:
		return l.expr.body.Get(l.ctx, tCtx)
	default:
		return nil, errors.New("invalid lambda: no body")
	}
}

func (l *LambdaExpression[K]) getLiteralValue() (any, bool) {
	if l.body != nil {
		if literalValue, ok := GetLiteralValue(l.body); ok {
			return literalValue, true
		}
	}
	if l.bodyExpr != nil {
		if litExp, ok := l.bodyExpr.(*literalBoolExpr[K]); ok {
			return litExp.getValue(), true
		}
	}
	return nil, false
}

// Close releases the activation's resources. Call it when the activation is no longer needed.
func (l *LambdaActivation[K]) Close() {
	l.ctx = nil
	l.activation.parent = nil
	clear(l.activation.bindings)
	clear(l.argValues)
	l.expr.activationPool.Put(l)
}

// NewTestingLambdaExpression creates a LambdaExpression with a value body for use in tests.
// eval is called with resolveBinding to resolve local identifier values from the active scope.
//
// Experimental: *NOTE* this API is subject to change or removal in the future.
func NewTestingLambdaExpression[K any](
	params []string,
	eval func(ctx context.Context, tCtx K, resolveBinding func(string) any) (any, error),
) *LambdaExpression[K] {
	getter := exprGetter[K]{
		expr: Expr[K]{exprFunc: func(ctx context.Context, tCtx K) (any, error) {
			resolveBinding := func(name string) any {
				v, err := resolveLocalIdentifierBinding(ctx, name)
				if err != nil {
					return nil
				}
				return v
			}
			return eval(ctx, tCtx, resolveBinding)
		}},
	}
	return newLambdaExpression(makeLocalIdentifiers(params...), &getter, nil)
}
