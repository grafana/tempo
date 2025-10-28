// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"golang.org/x/exp/constraints"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

// ExprFunc is a function in OTTL
type ExprFunc[K any] func(ctx context.Context, tCtx K) (any, error)

// Expr is a struct that represents a function
type Expr[K any] struct {
	exprFunc ExprFunc[K]
}

// Eval invokes the OTTL function
func (e Expr[K]) Eval(ctx context.Context, tCtx K) (any, error) {
	return e.exprFunc(ctx, tCtx)
}

// Getter resolves a value at runtime without performing any type checking on the value that is returned.
type Getter[K any] interface {
	// Get retrieves a value of type 'Any' and returns an error if there are any issues during retrieval.
	Get(ctx context.Context, tCtx K) (any, error)
}

// Setter allows setting an untyped value on a predefined field within some data at runtime.
type Setter[K any] interface {
	// Set sets a value of type 'Any' and returns an error if there are any issues during the setting process.
	Set(ctx context.Context, tCtx K, val any) error
}

// GetSetter is an interface that combines the Getter and Setter interfaces.
// It should be used to represent the ability to both get and set a value.
type GetSetter[K any] interface {
	Getter[K]
	Setter[K]
}

// literalGetter is an interface that allows Getter implementations to indicate if they
// support literal values.
type literalGetter interface {
	// isLiteral returns true if the Getter is a literal value.
	isLiteral() bool
	// getLiteral retrieves the literal value of the Getter.
	getLiteral() (any, error)
}

// StandardGetSetter is a standard way to construct a GetSetter
type StandardGetSetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (any, error)
	Setter func(ctx context.Context, tCtx K, val any) error
}

func (path StandardGetSetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
	return path.Getter(ctx, tCtx)
}

func (path StandardGetSetter[K]) Set(ctx context.Context, tCtx K, val any) error {
	return path.Setter(ctx, tCtx, val)
}

type literal[K any] struct {
	value any
}

func (l literal[K]) Get(context.Context, K) (any, error) {
	return l.value, nil
}

func (l literal[K]) getLiteral() (any, error) {
	return l.value, nil
}

func (literal[K]) isLiteral() bool {
	return true
}

type exprGetter[K any] struct {
	expr Expr[K]
	keys []key
}

func (g exprGetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
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
					return nil, errors.New("key not found in map")
				}
				result = ottlcommon.GetValue(val)
			case map[string]any:
				val, ok := r[*k.String]
				if !ok {
					return nil, errors.New("key not found in map")
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
			case []any:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			case []string:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			case []bool:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			case []float64:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			case []int64:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			case []byte:
				result, err = getElementByIndex(r, k.Int)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("type, %T, does not support int indexing", result)
			}
		default:
			return nil, errors.New("neither map nor slice index were set; this is an error in OTTL")
		}
	}
	return result, nil
}

func getElementByIndex[T any](r []T, idx *int64) (any, error) {
	if int(*idx) >= len(r) || int(*idx) < 0 {
		return nil, fmt.Errorf("index %v out of bounds", *idx)
	}
	return r[*idx], nil
}

type listGetter[K any] struct {
	slice []Getter[K]
}

func (l *listGetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
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

func (l *listGetter[K]) isLiteral() bool {
	if len(l.slice) == 0 {
		return false
	}
	for _, v := range l.slice {
		if getter, ok := v.(literalGetter); !ok || !getter.isLiteral() {
			return false
		}
	}
	return true
}

func (l *listGetter[K]) getLiteral() (any, error) {
	return l.Get(context.Background(), *new(K))
}

type mapGetter[K any] struct {
	mapValues map[string]Getter[K]
}

func (m *mapGetter[K]) Get(ctx context.Context, tCtx K) (any, error) {
	result := pcommon.NewMap()
	for k, v := range m.mapValues {
		val, err := v.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}
		switch typedVal := val.(type) {
		case pcommon.Map:
			target := result.PutEmpty(k).SetEmptyMap()
			typedVal.CopyTo(target)
		case []any:
			target := result.PutEmpty(k).SetEmptySlice()
			for _, el := range typedVal {
				switch typedEl := el.(type) {
				case pcommon.Map:
					m := target.AppendEmpty().SetEmptyMap()
					typedEl.CopyTo(m)
				default:
					err := target.AppendEmpty().FromRaw(el)
					if err != nil {
						return nil, err
					}
				}
			}
		default:
			err := result.PutEmpty(k).FromRaw(val)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (m *mapGetter[K]) isLiteral() bool {
	if len(m.mapValues) == 0 {
		return false
	}
	for _, v := range m.mapValues {
		if getter, ok := v.(literalGetter); !ok || !getter.isLiteral() {
			return false
		}
	}
	return true
}

func (m *mapGetter[K]) getLiteral() (any, error) {
	return m.Get(context.Background(), *new(K))
}

// PSliceGetter is a Getter that must return a pcommon.Slice.
type PSliceGetter[K any] interface {
	Get(ctx context.Context, tCtx K) (pcommon.Slice, error)
}

// newStandardPSliceGetter creates a new StandardPSliceGetter from a Getter[K],
// also checking if the Getter is a literalGetter.
func newStandardPSliceGetter[K any](getter Getter[K]) StandardPSliceGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardPSliceGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardPSliceGetter is a basic implementation of PSliceGetter
type StandardPSliceGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
}

// Get retrieves a pcommon.Slice value.
// If the value is not a pcommon.Slice a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardPSliceGetter[K]) Get(ctx context.Context, tCtx K) (pcommon.Slice, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return pcommon.Slice{}, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return pcommon.Slice{}, TypeError("expected pcommon.Slice but got nil")
	}
	switch v := val.(type) {
	case pcommon.Slice:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeSlice {
			return v.Slice(), nil
		}
		return pcommon.Slice{}, TypeError(fmt.Sprintf("expected pcommon.Slice but got %v", v.Type()))
	case []any:
		s := pcommon.NewSlice()
		err = s.FromRaw(v)
		if err != nil {
			return pcommon.Slice{}, err
		}
		return s, nil
	// Handle common slice types returned by OTTL functions
	case []string:
		return newPSliceFrom(v, func(target *pcommon.Value, value string) { target.SetStr(value) })
	case []int:
		return newPSliceFromIntegers(v)
	case []int16:
		return newPSliceFromIntegers(v)
	case []int32:
		return newPSliceFromIntegers(v)
	case []int64:
		return newPSliceFromIntegers(v)
	case []uint:
		return newPSliceFromIntegers(v)
	case []uint16:
		return newPSliceFromIntegers(v)
	case []uint32:
		return newPSliceFromIntegers(v)
	case []uint64:
		return newPSliceFromIntegers(v)
	case []float32:
		return newPSliceFrom(v, func(target *pcommon.Value, value float32) { target.SetDouble(float64(value)) })
	case []float64:
		return newPSliceFrom(v, func(target *pcommon.Value, value float64) { target.SetDouble(value) })
	case []bool:
		return newPSliceFrom(v, func(target *pcommon.Value, value bool) { target.SetBool(value) })
	default:
		return pcommon.Slice{}, TypeError(fmt.Sprintf("expected pcommon.Slice but got %T", val))
	}
}

func (g StandardPSliceGetter[K]) isLiteral() bool {
	return g.literal
}

func (g StandardPSliceGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardPSliceGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func newPSliceFromIntegers[T constraints.Integer](source []T) (pcommon.Slice, error) {
	return newPSliceFrom(source, func(target *pcommon.Value, value T) {
		target.SetInt(int64(value))
	})
}

func newPSliceFrom[T any](source []T, set func(target *pcommon.Value, value T)) (pcommon.Slice, error) {
	s := pcommon.NewSlice()
	s.EnsureCapacity(len(source))
	for _, v := range source {
		empty := s.AppendEmpty()
		set(&empty, v)
	}
	return s, nil
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

// newStandardStringGetter creates a new StandardStringGetter from a Getter[K],
// also checking if the Getter is a literalGetter.
func newStandardStringGetter[K any](getter Getter[K]) StandardStringGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardStringGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardStringGetter is a basic implementation of StringGetter
type StandardStringGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardStringGetter[K]) isLiteral() bool {
	return g.literal
}

func (g StandardStringGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardStringGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

// IntGetter is a Getter that must return an int64.
type IntGetter[K any] interface {
	// Get retrieves an int64 value.
	Get(ctx context.Context, tCtx K) (int64, error)
}

// newStandardIntGetter creates a new StandardIntGetter from a Getter[K],
// also checking if the Getter is a literalGetter.
func newStandardIntGetter[K any](getter Getter[K]) StandardIntGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardIntGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardIntGetter is a basic implementation of IntGetter
type StandardIntGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardIntGetter[K]) isLiteral() bool {
	return g.literal
}

func (g StandardIntGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardIntGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

// FloatGetter is a Getter that must return a float64.
type FloatGetter[K any] interface {
	// Get retrieves a float64 value.
	Get(ctx context.Context, tCtx K) (float64, error)
}

func newStandardFloatGetter[K any](getter Getter[K]) StandardFloatGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardFloatGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardFloatGetter is a basic implementation of FloatGetter
type StandardFloatGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardFloatGetter[K]) isLiteral() bool {
	return g.literal
}

func (g StandardFloatGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardFloatGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

// BoolGetter is a Getter that must return a bool.
type BoolGetter[K any] interface {
	// Get retrieves a bool value.
	Get(ctx context.Context, tCtx K) (bool, error)
}

func newStandardBoolGetter[K any](getter Getter[K]) StandardBoolGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardBoolGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardBoolGetter is a basic implementation of BoolGetter
type StandardBoolGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
}

// Get retrieves a bool value.
// If the value is not a bool a new TypeError is returned.
// If there is an error getting the value it will be returned.
func (g StandardBoolGetter[K]) Get(ctx context.Context, tCtx K) (bool, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return false, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return false, TypeError("expected bool but got nil")
	}
	switch v := val.(type) {
	case bool:
		return v, nil
	case pcommon.Value:
		if v.Type() == pcommon.ValueTypeBool {
			return v.Bool(), nil
		}
		return false, TypeError(fmt.Sprintf("expected bool but got %v", v.Type()))
	default:
		return false, TypeError(fmt.Sprintf("expected bool but got %T", val))
	}
}

func (g StandardBoolGetter[K]) isLiteral() bool {
	return g.literal
}

func (g StandardBoolGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardBoolGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

// FunctionGetter uses a function factory to return an instantiated function as an Expr.
type FunctionGetter[K any] interface {
	// Get returns a function as an Expr[K] built with the provided Arguments
	Get(args Arguments) (Expr[K], error)
}

// StandardFunctionGetter is a basic implementation of FunctionGetter.
type StandardFunctionGetter[K any] struct {
	FCtx FunctionContext
	Fact Factory[K]
}

// Get takes an Arguments struct containing arguments the caller wants passed to the
// function and instantiates the function with those arguments.
// If there is a mismatch between the function's signature and the arguments the caller
// wants to pass to the function, an error is returned.
func (g StandardFunctionGetter[K]) Get(args Arguments) (Expr[K], error) {
	if g.Fact == nil {
		return Expr[K]{}, errors.New("undefined function")
	}
	fArgs := g.Fact.CreateDefaultArguments()
	if reflect.TypeOf(fArgs).Kind() != reflect.Pointer {
		return Expr[K]{}, fmt.Errorf("factory for %q must return a pointer to an Arguments value in its CreateDefaultArguments method", g.Fact.Name())
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
		fArgsVal.Field(i).Set(field)
	}
	fn, err := g.Fact.CreateFunction(g.FCtx, fArgs)
	if err != nil {
		return Expr[K]{}, fmt.Errorf("couldn't create function: %w", err)
	}
	return Expr[K]{exprFunc: fn}, nil
}

// PMapGetSetter is a GetSetter that must interact with a pcommon.Map
type PMapGetSetter[K any] interface {
	Get(ctx context.Context, tCtx K) (pcommon.Map, error)
	Set(ctx context.Context, tCtx K, val pcommon.Map) error
}

// StandardPMapGetSetter is a basic implementation of PMapGetSetter
type StandardPMapGetSetter[K any] struct {
	Getter func(ctx context.Context, tCtx K) (pcommon.Map, error)
	Setter func(ctx context.Context, tCtx K, val any) error
}

func (path StandardPMapGetSetter[K]) Get(ctx context.Context, tCtx K) (pcommon.Map, error) {
	return path.Getter(ctx, tCtx)
}

func (path StandardPMapGetSetter[K]) Set(ctx context.Context, tCtx K, val pcommon.Map) error {
	return path.Setter(ctx, tCtx, val)
}

// PMapGetter is a Getter that must return a pcommon.Map.
type PMapGetter[K any] interface {
	// Get retrieves a pcommon.Map value.
	Get(ctx context.Context, tCtx K) (pcommon.Map, error)
}

// newStandardPMapGetter creates a new StandardPMapGetter from a Getter[K],
// also checking if the Getter is a literalGetter.
func newStandardPMapGetter[K any](getter Getter[K]) StandardPMapGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardPMapGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardPMapGetter is a basic implementation of PMapGetter
type StandardPMapGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardPMapGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardPMapGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardPMapGetter[K]) isLiteral() bool {
	return g.literal
}

// StringLikeGetter is a Getter that returns a string by converting the underlying value to a string if necessary.
type StringLikeGetter[K any] interface {
	// Get retrieves a string value.
	// Unlike `StringGetter`, the expectation is that the underlying value is converted to a string if possible.
	// If the value cannot be converted to a string, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*string, error)
}

// newStandardStringLikeGetter creates a new StandardStringLikeGetter from a Getter[K],
// also checking if the Getter is a literalGetter.
func newStandardStringLikeGetter[K any](getter Getter[K]) StandardStringLikeGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardStringLikeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardStringLikeGetter is a basic implementation of StringLikeGetter
type StandardStringLikeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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
		resultBytes, err := json.Marshal(v.AsRaw())
		if err != nil {
			return nil, err
		}
		result = string(resultBytes)
	case pcommon.Slice:
		resultBytes, err := json.Marshal(v.AsRaw())
		if err != nil {
			return nil, err
		}
		result = string(resultBytes)
	case pcommon.Value:
		result = v.AsString()
	default:
		resultBytes, err := json.Marshal(v)
		if err != nil {
			return nil, TypeError(fmt.Sprintf("unsupported type: %T", v))
		}
		result = string(resultBytes)
	}
	return &result, nil
}

func (g StandardStringLikeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardStringLikeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardStringLikeGetter[K]) isLiteral() bool {
	return g.literal
}

// FloatLikeGetter is a Getter that returns a float64 by converting the underlying value to a float64 if necessary.
type FloatLikeGetter[K any] interface {
	// Get retrieves a float64 value.
	// Unlike `FloatGetter`, the expectation is that the underlying value is converted to a float64 if possible.
	// If the value cannot be converted to a float64, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*float64, error)
}

func newStandardFloatLikeGetter[K any](getter Getter[K]) StandardFloatLikeGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardFloatLikeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardFloatLikeGetter is a basic implementation of FloatLikeGetter
type StandardFloatLikeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardFloatLikeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardFloatLikeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardFloatLikeGetter[K]) isLiteral() bool {
	return g.literal
}

// IntLikeGetter is a Getter that returns an int by converting the underlying value to an int if necessary
type IntLikeGetter[K any] interface {
	// Get retrieves an int value.
	// Unlike `IntGetter`, the expectation is that the underlying value is converted to an int if possible.
	// If the value cannot be converted to an int, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*int64, error)
}

func newStandardIntLikeGetter[K any](getter Getter[K]) StandardIntLikeGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardIntLikeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardIntLikeGetter is a basic implementation of IntLikeGetter
type StandardIntLikeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardIntLikeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardIntLikeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardIntLikeGetter[K]) isLiteral() bool {
	return g.literal
}

// ByteSliceLikeGetter is a Getter that returns []byte by converting the underlying value to an []byte if necessary
type ByteSliceLikeGetter[K any] interface {
	// Get retrieves []byte value.
	// The expectation is that the underlying value is converted to []byte if possible.
	// If the value cannot be converted to []byte, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) ([]byte, error)
}

func newStandardByteSliceLikeGetter[K any](getter Getter[K]) StandardByteSliceLikeGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardByteSliceLikeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardByteSliceLikeGetter is a basic implementation of ByteSliceLikeGetter
type StandardByteSliceLikeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
}

func (g StandardByteSliceLikeGetter[K]) Get(ctx context.Context, tCtx K) ([]byte, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return nil, nil
	}
	var result []byte
	switch v := val.(type) {
	case []byte:
		result = v
	case string:
		result = []byte(v)
	case float64, int64, bool:
		result, err = valueToBytes(v)
		if err != nil {
			return nil, fmt.Errorf("error converting value %f of %T: %w", v, g, err)
		}
	case pcommon.Value:
		switch v.Type() {
		case pcommon.ValueTypeBytes:
			result = v.Bytes().AsRaw()
		case pcommon.ValueTypeInt:
			result, err = valueToBytes(v.Int())
			if err != nil {
				return nil, fmt.Errorf("error converting value %d of int64: %w", v.Int(), err)
			}
		case pcommon.ValueTypeDouble:
			result, err = valueToBytes(v.Double())
			if err != nil {
				return nil, fmt.Errorf("error converting value %f of float64: %w", v.Double(), err)
			}
		case pcommon.ValueTypeStr:
			result = []byte(v.Str())
		case pcommon.ValueTypeBool:
			result, err = valueToBytes(v.Bool())
			if err != nil {
				return nil, fmt.Errorf("error converting value %s of bool: %w", v.Str(), err)
			}
		default:
			return nil, TypeError(fmt.Sprintf("unsupported value type: %v", v.Type()))
		}
	default:
		return nil, TypeError(fmt.Sprintf("unsupported type: %T", v))
	}
	return result, nil
}

func (g StandardByteSliceLikeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardByteSliceLikeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardByteSliceLikeGetter[K]) isLiteral() bool {
	return g.literal
}

// valueToBytes converts a value to a byte slice of length 8.
func valueToBytes(n any) ([]byte, error) {
	// Create a buffer to hold the bytes
	buf := new(bytes.Buffer)
	// Write the value to the buffer using binary.Write
	err := binary.Write(buf, binary.BigEndian, n)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// BoolLikeGetter is a Getter that returns a bool by converting the underlying value to a bool if necessary.
type BoolLikeGetter[K any] interface {
	// Get retrieves a bool value.
	// Unlike `BoolGetter`, the expectation is that the underlying value is converted to a bool if possible.
	// If the value cannot be converted to a bool, nil and an error are returned.
	// If the value is nil, nil is returned without an error.
	Get(ctx context.Context, tCtx K) (*bool, error)
}

func newStandardBoolLikeGetter[K any](getter Getter[K]) StandardBoolLikeGetter[K] {
	litGetter, isLiteralGetter := getter.(literalGetter)
	return StandardBoolLikeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteralGetter && litGetter.isLiteral(),
	}
}

// StandardBoolLikeGetter is a basic implementation of BoolLikeGetter
type StandardBoolLikeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
}

func (g StandardBoolLikeGetter[K]) Get(ctx context.Context, tCtx K) (*bool, error) {
	val, err := g.Getter(ctx, tCtx)
	if err != nil {
		return nil, fmt.Errorf("error getting value in %T: %w", g, err)
	}
	if val == nil {
		return nil, nil
	}
	var result bool
	switch v := val.(type) {
	case bool:
		result = v
	case int:
		result = v != 0
	case int64:
		result = v != 0
	case string:
		result, err = strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
	case float64:
		result = v != 0.0
	case pcommon.Value:
		switch v.Type() {
		case pcommon.ValueTypeBool:
			result = v.Bool()
		case pcommon.ValueTypeInt:
			result = v.Int() != 0
		case pcommon.ValueTypeStr:
			result, err = strconv.ParseBool(v.Str())
			if err != nil {
				return nil, err
			}
		case pcommon.ValueTypeDouble:
			result = v.Double() != 0.0
		default:
			return nil, TypeError(fmt.Sprintf("unsupported value type: %v", v.Type()))
		}
	default:
		return nil, TypeError(fmt.Sprintf("unsupported type: %T", val))
	}
	return &result, nil
}

func (g StandardBoolLikeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardBoolLikeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardBoolLikeGetter[K]) isLiteral() bool {
	return g.literal
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
		return &literal[K]{value: []byte(*b)}, nil
	}

	if val.Enum != nil {
		enum, err := p.enumParser((*EnumSymbol)(val.Enum))
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
			np, err := p.newPath(eL.Path)
			if err != nil {
				return nil, err
			}
			return p.parsePath(np)
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

	if val.Map != nil {
		mg := mapGetter[K]{mapValues: map[string]Getter[K]{}}
		for _, kvp := range val.Map.Values {
			getter, err := p.newGetter(*kvp.Value)
			if err != nil {
				return nil, err
			}
			mg.mapValues[*kvp.Key] = getter
		}
		return &mg, nil
	}

	if val.MathExpression == nil {
		// In practice, can't happen since the DSL grammar guarantees one is set
		return nil, errors.New("no value field set. This is a bug in the OpenTelemetry Transformation Language")
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

func newStandardTimeGetter[K any](getter Getter[K]) StandardTimeGetter[K] {
	litGetter, isLiteral := getter.(literalGetter)
	return StandardTimeGetter[K]{
		Getter:  getter.Get,
		literal: isLiteral && litGetter.isLiteral(),
	}
}

// StandardTimeGetter is a basic implementation of TimeGetter
type StandardTimeGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardTimeGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardTimeGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardTimeGetter[K]) isLiteral() bool {
	return g.literal
}

// DurationGetter is a Getter that must return a time.Duration.
type DurationGetter[K any] interface {
	// Get retrieves a time.Duration value.
	Get(ctx context.Context, tCtx K) (time.Duration, error)
}

func newStandardDurationGetter[K any](getter Getter[K]) StandardDurationGetter[K] {
	litGetter, isLiteral := getter.(literalGetter)
	return StandardDurationGetter[K]{
		Getter:  getter.Get,
		literal: isLiteral && litGetter.isLiteral(),
	}
}

// StandardDurationGetter is a basic implementation of DurationGetter
type StandardDurationGetter[K any] struct {
	Getter  func(ctx context.Context, tCtx K) (any, error)
	literal bool
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

func (g StandardDurationGetter[K]) getLiteral() (any, error) {
	if !g.literal {
		return nil, errors.New("StandardDurationGetter value is not a literal")
	}
	return g.Get(context.Background(), *new(K))
}

func (g StandardDurationGetter[K]) isLiteral() bool {
	return g.literal
}
