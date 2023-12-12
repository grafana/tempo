// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
)

type PathExpressionParser[K any] func(*Path) (GetSetter[K], error)

type EnumParser func(*EnumSymbol) (*Enum, error)

type Enum int64

func (p *Parser[K]) newFunctionCall(ed editor) (Expr[K], error) {
	f, ok := p.functions[ed.Function]
	if !ok {
		return Expr[K]{}, fmt.Errorf("undefined function %q", ed.Function)
	}
	defaultArgs := f.CreateDefaultArguments()
	var args Arguments

	// A nil value indicates the function takes no arguments.
	if defaultArgs != nil {
		// Pointer values are necessary to fulfill the Go reflection
		// settability requirements. Non-pointer values are not
		// modifiable through reflection.
		if reflect.TypeOf(defaultArgs).Kind() != reflect.Pointer {
			return Expr[K]{}, fmt.Errorf("factory for %q must return a pointer to an Arguments value in its CreateDefaultArguments method", ed.Function)
		}

		args = reflect.New(reflect.ValueOf(defaultArgs).Elem().Type()).Interface()

		err := p.buildArgs(ed, reflect.ValueOf(args).Elem())
		if err != nil {
			return Expr[K]{}, fmt.Errorf("error while parsing arguments for call to %q: %w", ed.Function, err)
		}
	}

	fn, err := f.CreateFunction(FunctionContext{Set: p.telemetrySettings}, args)
	if err != nil {
		return Expr[K]{}, fmt.Errorf("couldn't create function: %w", err)
	}

	return Expr[K]{exprFunc: fn}, err
}

func (p *Parser[K]) buildArgs(ed editor, argsVal reflect.Value) error {
	requiredArgs := 0
	seenNamed := false

	for i := 0; i < len(ed.Arguments); i++ {
		if !seenNamed && ed.Arguments[i].Name != "" {
			seenNamed = true
		} else if seenNamed && ed.Arguments[i].Name == "" {
			return errors.New("unnamed argument used after named argument")
		}
	}

	for i := 0; i < argsVal.NumField(); i++ {
		if !strings.HasPrefix(argsVal.Field(i).Type().Name(), "Optional") {
			requiredArgs++
		}
	}

	if len(ed.Arguments) < requiredArgs || len(ed.Arguments) > argsVal.NumField() {
		return fmt.Errorf("incorrect number of arguments. Expected: %d Received: %d", argsVal.NumField(), len(ed.Arguments))
	}

	for i, edArg := range ed.Arguments {
		var field reflect.Value
		var fieldType reflect.Type
		var isOptional bool
		var arg argument

		if edArg.Name == "" {
			field = argsVal.Field(i)
			fieldType = field.Type()
			isOptional = strings.HasPrefix(fieldType.Name(), "Optional")
			arg = ed.Arguments[i]
		} else {
			field = argsVal.FieldByName(strcase.ToCamel(edArg.Name))
			if !field.IsValid() {
				return fmt.Errorf("no such parameter: %s", edArg.Name)
			}
			fieldType = field.Type()
			isOptional = strings.HasPrefix(fieldType.Name(), "Optional")
			arg = edArg
		}

		var val any
		var manager optionalManager
		var err error
		var ok bool
		if isOptional {
			manager, ok = field.Interface().(optionalManager)

			if !ok {
				return errors.New("optional type is not manageable by the OTTL parser. This is an error in the OTTL")
			}

			fieldType = manager.get().Type()
		}

		switch {
		case strings.HasPrefix(fieldType.Name(), "FunctionGetter"):
			var name string
			switch {
			case arg.Value.Enum != nil:
				name = string(*arg.Value.Enum)
			case arg.Value.FunctionName != nil:
				name = *arg.Value.FunctionName
			default:
				return fmt.Errorf("invalid function name given")
			}
			f, ok := p.functions[name]
			if !ok {
				return fmt.Errorf("undefined function %s", name)
			}
			val = StandardFunctionGetter[K]{FCtx: FunctionContext{Set: p.telemetrySettings}, Fact: f}
		case fieldType.Kind() == reflect.Slice:
			val, err = p.buildSliceArg(arg.Value, fieldType)
		default:
			val, err = p.buildArg(arg.Value, fieldType)
		}
		if err != nil {
			return fmt.Errorf("invalid argument at position %v: %w", i, err)
		}
		if isOptional {
			field.Set(manager.set(val))
		} else {
			field.Set(reflect.ValueOf(val))
		}
	}

	return nil
}

func (p *Parser[K]) buildSliceArg(argVal value, argType reflect.Type) (any, error) {
	name := argType.Elem().Name()
	switch {
	case name == reflect.Uint8.String():
		if argVal.Bytes == nil {
			return nil, fmt.Errorf("slice parameter must be a byte slice literal")
		}
		return ([]byte)(*argVal.Bytes), nil
	case name == reflect.String.String():
		arg, err := buildSlice[string](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case name == reflect.Float64.String():
		arg, err := buildSlice[float64](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case name == reflect.Int64.String():
		arg, err := buildSlice[int64](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "Getter"):
		arg, err := buildSlice[Getter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "PMapGetter"):
		arg, err := buildSlice[PMapGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "StringGetter"):
		arg, err := buildSlice[StringGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "StringLikeGetter"):
		arg, err := buildSlice[StringLikeGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "FloatGetter"):
		arg, err := buildSlice[FloatGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "FloatLikeGetter"):
		arg, err := buildSlice[FloatLikeGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "IntGetter"):
		arg, err := buildSlice[IntGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "IntLikeGetter"):
		arg, err := buildSlice[IntLikeGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "DurationGetter"):
		arg, err := buildSlice[DurationGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "TimeGetter"):
		arg, err := buildSlice[TimeGetter[K]](argVal, argType, p.buildArg, name)
		if err != nil {
			return nil, err
		}
		return arg, nil
	default:
		return nil, fmt.Errorf("unsupported slice type %q for function", argType.Elem().Name())
	}
}

// Handle interfaces that can be passed as arguments to OTTL functions.
func (p *Parser[K]) buildArg(argVal value, argType reflect.Type) (any, error) {
	name := argType.Name()
	switch {
	case strings.HasPrefix(name, "Setter"):
		fallthrough
	case strings.HasPrefix(name, "GetSetter"):
		if argVal.Literal == nil || argVal.Literal.Path == nil {
			return nil, fmt.Errorf("must be a Path")
		}
		arg, err := p.pathParser(argVal.Literal.Path)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "Getter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return arg, nil
	case strings.HasPrefix(name, "StringGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardStringGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "StringLikeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardStringLikeGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "FloatGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardFloatGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "FloatLikeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardFloatLikeGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "IntGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardIntGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "IntLikeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardIntLikeGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "PMapGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardPMapGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "DurationGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardDurationGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "TimeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardTimeGetter[K]{Getter: arg.Get}, nil
	case name == "Enum":
		arg, err := p.enumParser(argVal.Enum)
		if err != nil {
			return nil, fmt.Errorf("must be an Enum")
		}
		return *arg, nil
	case name == reflect.String.String():
		if argVal.String == nil {
			return nil, fmt.Errorf("must be a string")
		}
		return *argVal.String, nil
	case name == reflect.Float64.String():
		if argVal.Literal == nil || argVal.Literal.Float == nil {
			return nil, fmt.Errorf("must be a float")
		}
		return *argVal.Literal.Float, nil
	case name == reflect.Int64.String():
		if argVal.Literal == nil || argVal.Literal.Int == nil {
			return nil, fmt.Errorf("must be an int")
		}
		return *argVal.Literal.Int, nil
	case name == reflect.Bool.String():
		if argVal.Bool == nil {
			return nil, fmt.Errorf("must be a bool")
		}
		return bool(*argVal.Bool), nil
	default:
		return nil, fmt.Errorf("unsupported argument type: %s", name)
	}
}

type buildArgFunc func(value, reflect.Type) (any, error)

func buildSlice[T any](argVal value, argType reflect.Type, buildArg buildArgFunc, name string) (any, error) {
	if argVal.List == nil {
		return nil, fmt.Errorf("must be a list of type %v", name)
	}

	vals := []T{}
	values := argVal.List.Values
	for j := 0; j < len(values); j++ {
		untypedVal, err := buildArg(values[j], argType.Elem())
		if err != nil {
			return nil, fmt.Errorf("error while parsing list argument at index %v: %w", j, err)
		}

		val, ok := untypedVal.(T)

		if !ok {
			return nil, fmt.Errorf("invalid element type at list index %v, must be of type %v", j, name)
		}

		vals = append(vals, val)
	}

	return vals, nil
}

// optionalManager provides a way for the parser to handle Optional[T] structs
// without needing to know the concrete type of T, which is inaccessible through
// the reflect package.
// Would likely be resolved by https://github.com/golang/go/issues/54393.
type optionalManager interface {
	// set takes a non-reflection value and returns a reflect.Value of
	// an Optional[T] struct with this value set.
	set(val any) reflect.Value

	// get returns a reflect.Value value of the value contained within
	// an Optional[T]. This allows obtaining a reflect.Type for T.
	get() reflect.Value
}

type Optional[T any] struct {
	val      T
	hasValue bool
}

// This is called only by reflection.
// nolint:unused
func (o Optional[T]) set(val any) reflect.Value {
	return reflect.ValueOf(Optional[T]{
		val:      val.(T),
		hasValue: true,
	})
}

func (o Optional[T]) IsEmpty() bool {
	return !o.hasValue
}

func (o Optional[T]) Get() T {
	return o.val
}

func (o Optional[T]) get() reflect.Value {
	// `(reflect.Value).Call` will create a reflect.Value containing a zero-valued T.
	// Trying to create a reflect.Value for T by calling reflect.TypeOf or
	// reflect.ValueOf on an empty T value creates an invalid reflect.Value object,
	// the `Call` method appears to do extra processing to capture the type.
	return reflect.ValueOf(o).MethodByName("Get").Call(nil)[0]
}

// Allows creating an Optional with a value already populated for use in testing
// OTTL functions.
func NewTestingOptional[T any](val T) Optional[T] {
	return Optional[T]{
		val:      val,
		hasValue: true,
	}
}
