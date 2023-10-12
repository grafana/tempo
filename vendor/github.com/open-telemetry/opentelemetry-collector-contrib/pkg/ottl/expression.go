// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

type ExprFunc[K any] func(ctx context.Context, tCtx K) (interface{}, error)

type Expr[K any] struct {
	exprFunc ExprFunc[K]
}

func (e Expr[K]) Eval(ctx context.Context, tCtx K) (interface{}, error) {
	return e.exprFunc(ctx, tCtx)
}

type Getter[K any] interface {
	Get(ctx context.Context, tCtx K) (interface{}, error)
}

type Setter[K any] interface {
	Set(ctx context.Context, tCtx K, val interface{}) error
}

type GetSetter[K any] interface {
	Getter[K]
	Setter[K]
}

type StandardGetSetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
	Setter func(ctx context.Context, tCtx K, val interface{}) error
}

func (path StandardGetSetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	return path.Getter(ctx, tCtx)
}

func (path StandardGetSetter[K]) Set(ctx context.Context, tCtx K, val interface{}) error {
	return path.Setter(ctx, tCtx, val)
}

type literal[K any] struct {
	value interface{}
}

func (l literal[K]) Get(context.Context, K) (interface{}, error) {
	return l.value, nil
}

type exprGetter[K any] struct {
	expr Expr[K]
	keys []Key
}

func (g exprGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	result, err := g.expr.Eval(ctx, tCtx)
	if err != nil {
		return nil, err
	}

	if g.keys == nil {
		return result, nil
	}

	for _, k := range g.keys {
		switch {
		case k.String != nil:
			switch r := result.(type) {
			case pcommon.Map:
				val, ok := r.Get(*k.String)
				if !ok {
					return nil, fmt.Errorf("key not found in map")
				}
				result = ottlcommon.GetValue(val)
			case map[string]interface{}:
				val, ok := r[*k.String]
				if !ok {
					return nil, fmt.Errorf("key not found in map")
				}
				result = val
			default:
				return nil, fmt.Errorf("type, %T, does not support string indexing", result)
			}
		case k.Int != nil:
			switch r := result.(type) {
			case pcommon.Slice:
				if int(*k.Int) >= r.Len() || int(*k.Int) < 0 {
					return nil, fmt.Errorf("index %v out of bounds", *k.Int)
				}
				result = ottlcommon.GetValue(r.At(int(*k.Int)))
			case []interface{}:
				if int(*k.Int) >= len(r) || int(*k.Int) < 0 {
					return nil, fmt.Errorf("index %v out of bounds", *k.Int)
				}
				result = r[*k.Int]
			default:
				return nil, fmt.Errorf("type, %T, does not support int indexing", result)
			}
		default:
			return nil, fmt.Errorf("neither map nor slice index were set; this is an error in OTTL")
		}
	}
	return result, nil
}

type listGetter[K any] struct {
	slice []Getter[K]
}

func (l *listGetter[K]) Get(ctx context.Context, tCtx K) (interface{}, error) {
	evaluated := make([]any, len(l.slice))

	for i, v := range l.slice {
		val, err := v.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		evaluated[i] = val
	}

	return evaluated, nil
}

// TypeError represents that a value was not an expected type.
type TypeError string

func (t TypeError) Error() string {
	return string(t)
}

// StringGetter is a Getter that must return a string.
type StringGetter[K any] interface {
	// Get retrieves a string value.
	Get(ctx context.Context, tCtx K) (string, error)
}

// StandardStringGetter is a basic implementation of StringGetter
type StandardStringGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves a string value.
// If the value is not a string a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardStringGetter[K]) Get(ctx context.Context, tCtx K) (string, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return "", fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return "", TypeError("expected string but got nil")
	}
	switch v := val.(type) {
	case string:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeStr {
			return v.Str(), nil
		}
		return "", TypeError(fmt.Sprintf("expected string but got %v", v.Type()))
	default:
		return "", TypeError(fmt.Sprintf("expected string but got %T", val))
	}
}

// IntGetter is a Getter that must return an int64.
type IntGetter[K any] interface {
	// Get retrieves an int64 value.
	Get(ctx context.Context, tCtx K) (int64, error)
}

// StandardIntGetter is a basic implementation of IntGetter
type StandardIntGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves an int64 value.
// If the value is not an int64 a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardIntGetter[K]) Get(ctx context.Context, tCtx K) (int64, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return 0, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return 0, TypeError("expected int64 but got nil")
	}
	switch v := val.(type) {
	case int64:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeInt {
			return v.Int(), nil
		}
		return 0, TypeError(fmt.Sprintf("expected int64 but got %v", v.Type()))
	default:
		return 0, TypeError(fmt.Sprintf("expected int64 but got %T", val))
	}
}

// FloatGetter is a Getter that must return a float64.
type FloatGetter[K any] interface {
	// Get retrieves a float64 value.
	Get(ctx context.Context, tCtx K) (float64, error)
}

// StandardFloatGetter is a basic implementation of FloatGetter
type StandardFloatGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves a float64 value.
// If the value is not a float64 a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardFloatGetter[K]) Get(ctx context.Context, tCtx K) (float64, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return 0, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return 0, TypeError("expected float64 but got nil")
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeDouble {
			return v.Double(), nil
		}
		return 0, TypeError(fmt.Sprintf("expected float64 but got %v", v.Type()))
	default:
		return 0, TypeError(fmt.Sprintf("expected float64 but got %T", val))
	}
}

// FunctionGetter uses a function factory to return an instantiated function as an Expr.
type FunctionGetter[K any] interface {
	Get(args Arguments) (Expr[K], error)
}

// StandardFunctionGetter is a basic implementation of FunctionGetter.
type StandardFunctionGetter[K any] struct {
	fCtx FunctionContext
	fact Factory[K]
}

// Get takes an Arguments struct containing arguments the caller wants passed to the
// function and instantiates the function with those arguments.
// If there is a mismatch between the function's signature and the arguments the caller
// wants to pass to the function, an error is returned.
func (g StandardFunctionGetter[K]) Get(args Arguments) (Expr[K], error) {
	if g.fact == nil {
		return Expr[K]{}, fmt.Errorf("undefined function")
	}
	fArgs := g.fact.CreateDefaultArguments()
	if reflect.TypeOf(fArgs).Kind() != reflect.Pointer {
		return Expr[K]{}, fmt.Errorf("factory for %q must return a pointer to an Arguments value in its CreateDefaultArguments method", g.fact.Name())
	}
	if reflect.TypeOf(args).Kind() != reflect.Pointer {
		return Expr[K]{}, fmt.Errorf("%q must be pointer to an Arguments value", reflect.TypeOf(args).Kind())
	}
	fArgsVal := reflect.ValueOf(fArgs).Elem()
	argsVal := reflect.ValueOf(args).Elem()
	if fArgsVal.NumField() != argsVal.NumField() {
		return Expr[K]{}, fmt.Errorf("incorrect number of arguments. Expected: %d Received: %d", fArgsVal.NumField(), argsVal.NumField())
	}
	for i := 0; i < fArgsVal.NumField(); i++ {
		field := argsVal.Field(i)
		argIndex, err := getArgumentIndex(i, argsVal)
		if err != nil {
			return Expr[K]{}, err
		}
		fArgIndex, err := getArgumentIndex(argIndex, fArgsVal)
		if err != nil {
			return Expr[K]{}, err
		}
		fArgsVal.Field(fArgIndex).Set(field)
	}
	fn, err := g.fact.CreateFunction(g.fCtx, fArgs)
	if err != nil {
		return Expr[K]{}, fmt.Errorf("couldn't create function: %w", err)
	}
	return Expr[K]{exprFunc: fn}, nil
}

// PMapGetter is a Getter that must return a pcommon.Map.
type PMapGetter[K any] interface {
	// Get retrieves a pcommon.Map value.
	Get(ctx context.Context, tCtx K) (pcommon.Map, error)
}

// StandardPMapGetter is a basic implementation of PMapGetter
type StandardPMapGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves a pcommon.Map value.
// If the value is not a pcommon.Map a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardPMapGetter[K]) Get(ctx context.Context, tCtx K) (pcommon.Map, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return pcommon.Map{}, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return pcommon.Map{}, TypeError("expected pcommon.Map but got nil")
	}
	switch v := val.(type) {
	case pcommon.Map:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeMap {
			return v.Map(), nil
		}
		return pcommon.Map{}, TypeError(fmt.Sprintf("expected pcommon.Map but got %v", v.Type()))
	case map[string]any:
		m := pcommon.NewMap()
		err = m.FromRaw(v)
		if err != nil {
			return pcommon.Map{}, err
		}
		return m, nil
	default:
		return pcommon.Map{}, TypeError(fmt.Sprintf("expected pcommon.Map but got %T", val))
	}
}

// StringLikeGetter is a Getter that returns a string by converting the underlying value to a string if necessary.
type StringLikeGetter[K any] interface {
	// Get retrieves a string value.
	// Unlike `StringGetter`, the expectation is that the underlying value is converted to a string if possible.
	// If the value cannot be converted to a string, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*string, error)
}

type StandardStringLikeGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

func (g StandardStringLikeGetter[K]) Get(ctx context.Context, tCtx K) (*string, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return nil, nil
	}
	var result string
	switch v := val.(type) {
	case string:
		result = v
	case []byte:
		result = hex.EncodeToString(v)
	case pcommon.Map:
		result, err = jsoniter.MarshalToString(v.AsRaw())
		if err != nil {
			return nil, err
		}
	case pcommon.Slice:
		result, err = jsoniter.MarshalToString(v.AsRaw())
		if err != nil {
			return nil, err
		}
	case pcommon.Value:
		result = v.AsString()
	default:
		result, err = jsoniter.MarshalToString(v)
		if err != nil {
			return nil, TypeError(fmt.Sprintf("unsupported type: %T", v))
		}
	}
	return &result, nil
}

// FloatLikeGetter is a Getter that returns a float64 by converting the underlying value to a float64 if necessary.
type FloatLikeGetter[K any] interface {
	// Get retrieves a float64 value.
	// Unlike `FloatGetter`, the expectation is that the underlying value is converted to a float64 if possible.
	// If the value cannot be converted to a float64, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*float64, error)
}

type StandardFloatLikeGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

func (g StandardFloatLikeGetter[K]) Get(ctx context.Context, tCtx K) (*float64, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return nil, nil
	}
	var result float64
	switch v := val.(type) {
	case float64:
		result = v
	case int64:
		result = float64(v)
	case string:
		result, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
	case bool:
		if v {
			result = float64(1)
		} else {
			result = float64(0)
		}
	case pcommon.Value:
		switch v.Type() {
		case pcommon.ValueTypeDouble:
			result = v.Double()
		case pcommon.ValueTypeInt:
			result = float64(v.Int())
		case pcommon.ValueTypeStr:
			result, err = strconv.ParseFloat(v.Str(), 64)
			if err != nil {
				return nil, err
			}
		case pcommon.ValueTypeBool:
			if v.Bool() {
				result = float64(1)
			} else {
				result = float64(0)
			}
		default:
			return nil, TypeError(fmt.Sprintf("unsupported value type: %v", v.Type()))
		}
	default:
		return nil, TypeError(fmt.Sprintf("unsupported type: %T", v))
	}
	return &result, nil
}

// IntLikeGetter is a Getter that returns an int by converting the underlying value to an int if necessary
type IntLikeGetter[K any] interface {
	// Get retrieves an int value.
	// Unlike `IntGetter`, the expectation is that the underlying value is converted to an int if possible.
	// If the value cannot be converted to an int, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*int64, error)
}

type StandardIntLikeGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

func (g StandardIntLikeGetter[K]) Get(ctx context.Context, tCtx K) (*int64, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return nil, nil
	}
	var result int64
	switch v := val.(type) {
	case int64:
		result = v
	case string:
		result, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, nil
		}
	case float64:
		result = int64(v)
	case bool:
		if v {
			result = int64(1)
		} else {
			result = int64(0)
		}
	case pcommon.Value:
		switch v.Type() {
		case pcommon.ValueTypeInt:
			result = v.Int()
		case pcommon.ValueTypeDouble:
			result = int64(v.Double())
		case pcommon.ValueTypeStr:
			result, err = strconv.ParseInt(v.Str(), 10, 64)
			if err != nil {
				return nil, nil
			}
		case pcommon.ValueTypeBool:
			if v.Bool() {
				result = int64(1)
			} else {
				result = int64(0)
			}
		default:
			return nil, TypeError(fmt.Sprintf("unsupported value type: %v", v.Type()))
		}
	default:
		return nil, TypeError(fmt.Sprintf("unsupported type: %T", v))
	}
	return &result, nil
}

func (p *Parser[K]) newGetter(val value) (Getter[K], error) {
	if val.IsNil != nil && *val.IsNil {
		return &literal[K]{value: nil}, nil
	}

	if s := val.String; s != nil {
		return &literal[K]{value: *s}, nil
	}
	if b := val.Bool; b != nil {
		return &literal[K]{value: bool(*b)}, nil
	}
	if b := val.Bytes; b != nil {
		return &literal[K]{value: ([]byte)(*b)}, nil
	}

	if val.Enum != nil {
		enum, err := p.enumParser(val.Enum)
		if err != nil {
			return nil, err
		}
		return &literal[K]{value: int64(*enum)}, nil
	}

	if eL := val.Literal; eL != nil {
		if f := eL.Float; f != nil {
			return &literal[K]{value: *f}, nil
		}
		if i := eL.Int; i != nil {
			return &literal[K]{value: *i}, nil
		}
		if eL.Path != nil {
			return p.pathParser(eL.Path)
		}
		if eL.Converter != nil {
			return p.newGetterFromConverter(*eL.Converter)
		}
	}

	if val.List != nil {
		lg := listGetter[K]{slice: make([]Getter[K], len(val.List.Values))}
		for i, v := range val.List.Values {
			getter, err := p.newGetter(v)
			if err != nil {
				return nil, err
			}
			lg.slice[i] = getter
		}
		return &lg, nil
	}

	if val.MathExpression == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, fmt.Errorf("no value field set. This is a bug in the OpenTelemetry Transformation Language")
	}
	return p.evaluateMathExpression(val.MathExpression)
}

func (p *Parser[K]) newGetterFromConverter(c converter) (Getter[K], error) {
	call, err := p.newFunctionCall(editor(c))
	if err != nil {
		return nil, err
	}
	return &exprGetter[K]{
		expr: call,
		keys: c.Keys,
	}, nil
}

// TimeGetter is a Getter that must return a time.Time.
type TimeGetter[K any] interface {
	// Get retrieves a time.Time value.
	Get(ctx context.Context, tCtx K) (time.Time, error)
}

// StandardTimeGetter is a basic implementation of TimeGetter
type StandardTimeGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves a time.Time value.
// If the value is not a time.Time, a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardTimeGetter[K]) Get(ctx context.Context, tCtx K) (time.Time, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return time.Time{}, TypeError("expected time but got nil")
	}
	switch v := val.(type) {
	case time.Time:
		return v, nil
	default:
		return time.Time{}, TypeError(fmt.Sprintf("expected time but got %T", val))
	}
}

// DurationGetter is a Getter that must return a time.Duration.
type DurationGetter[K any] interface {
	// Get retrieves a time.Duration value.
	Get(ctx context.Context, tCtx K) (time.Duration, error)
}

// StandardDurationGetter is a basic implementation of DurationGetter
type StandardDurationGetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (interface{}, error)
}

// Get retrieves an time.Duration value.
// If the value is not an time.Duration a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardDurationGetter[K]) Get(ctx context.Context, tCtx K) (time.Duration, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return 0, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return 0, TypeError("expected duration but got nil")
	}
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	default:
		return 0, TypeError(fmt.Sprintf("expected duration but got %T", val))
	}
}
