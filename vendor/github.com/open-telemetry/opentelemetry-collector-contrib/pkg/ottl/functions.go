// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type PathExpressionParser[K any] func(*Path) (GetSetter[K], error)

type EnumParser func(*EnumSymbol) (*Enum, error)

type Enum int64

func (p *Parser[K]) newFunctionCall(ed editor) (Expr[K], error) {
	f, ok := p.functions[ed.Function]
	if !ok {
		return Expr[K]{}, fmt.Errorf("undefined function %q", ed.Function)
	}
	args := f.CreateDefaultArguments()

	// A nil value indicates the function takes no arguments.
	if args != nil {
		// Pointer values are necessary to fulfill the Go reflection
		// settability requirements. Non-pointer values are not
		// modifiable through reflection.
		if reflect.TypeOf(args).Kind() != reflect.Pointer {
			return Expr[K]{}, fmt.Errorf("factory for %q must return a pointer to an Arguments value in its CreateDefaultArguments method", ed.Function)
		}

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

func getArgumentIndex(index int, args reflect.Value) (int, error) {
	argsType := args.Type()
	fieldTag, ok := argsType.Field(index).Tag.Lookup("ottlarg")
	if !ok {
		return 0, fmt.Errorf("no `ottlarg` struct tag on Arguments field %q", argsType.Field(index).Name)
	}
	argNum, err := strconv.Atoi(fieldTag)
	if err != nil {
		return 0, fmt.Errorf("ottlarg struct tag on field %q is not a valid integer: %w", argsType.Field(index).Name, err)
	}
	if argNum < 0 || argNum >= args.NumField() {
		return 0, fmt.Errorf("ottlarg struct tag on field %q has value %d, but must be between 0 and %d", argsType.Field(index).Name, argNum, args.NumField())
	}
	return argNum, nil
}

func (p *Parser[K]) buildArgs(ed editor, argsVal reflect.Value) error {
	if len(ed.Arguments) != argsVal.NumField() {
		return fmt.Errorf("incorrect number of arguments. Expected: %d Received: %d", argsVal.NumField(), len(ed.Arguments))
	}

	for i := 0; i < argsVal.NumField(); i++ {
		field := argsVal.Field(i)
		fieldType := field.Type()
		argNum, err := getArgumentIndex(i, argsVal)
		if err != nil {
			return err
		}
		argVal := ed.Arguments[argNum]
		var val any
		switch {
		case strings.HasPrefix(fieldType.Name(), "FunctionGetter"):
			var name string
			switch {
			case argVal.Enum != nil:
				name = string(*argVal.Enum)
			case argVal.FunctionName != nil:
				name = *argVal.FunctionName
			default:
				return fmt.Errorf("invalid function name given")
			}
			f, ok := p.functions[name]
			if !ok {
				return fmt.Errorf("undefined function %s", name)
			}
			val = StandardFunctionGetter[K]{fCtx: FunctionContext{Set: p.telemetrySettings}, fact: f}
		case fieldType.Kind() == reflect.Slice:
			val, err = p.buildSliceArg(argVal, fieldType)
		default:
			val, err = p.buildArg(argVal, fieldType)
		}
		if err != nil {
			return fmt.Errorf("invalid argument at position %v: %w", i, err)
		}
		field.Set(reflect.ValueOf(val))
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
		return nil, errors.New("unsupported argument type")
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
