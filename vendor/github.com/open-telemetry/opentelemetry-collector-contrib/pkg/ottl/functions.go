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
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type PathExpressionParser[K any] func(*Path) (GetSetter[K], error)

type EnumParser func(*EnumSymbol) (*Enum, error)

type Enum int64

func (p *Parser[K]) newFunctionCall(inv invocation) (Expr[K], error) {
	f, ok := p.functions[inv.Function]
	if !ok {
		return Expr[K]{}, fmt.Errorf("undefined function %v", inv.Function)
	}
	args, err := p.buildArgs(inv, reflect.TypeOf(f))
	if err != nil {
		return Expr[K]{}, fmt.Errorf("error while parsing arguments for call to '%v': %w", inv.Function, err)
	}

	returnVals := reflect.ValueOf(f).Call(args)

	if returnVals[1].IsNil() {
		err = nil
	} else {
		err = returnVals[1].Interface().(error)
	}

	return Expr[K]{exprFunc: returnVals[0].Interface().(ExprFunc[K])}, err
}

func (p *Parser[K]) buildArgs(inv invocation, fType reflect.Type) ([]reflect.Value, error) {
	var args []reflect.Value
	// Some function arguments may be intended to take values from the calling processor
	// instead of being passed by the caller of the OTTL function, so we have to keep
	// track of the index of the argument passed within the DSL.
	// e.g. TelemetrySettings, which is provided by the processor to the OTTL Parser struct.
	DSLArgumentIndex := 0
	for i := 0; i < fType.NumIn(); i++ {
		argType := fType.In(i)

		arg, isInternalArg := p.buildInternalArg(argType)
		if isInternalArg {
			args = append(args, reflect.ValueOf(arg))
			continue
		}

		if DSLArgumentIndex >= len(inv.Arguments) {
			return nil, fmt.Errorf("not enough arguments")
		}

		argVal := inv.Arguments[DSLArgumentIndex]

		var val any
		var err error
		if argType.Kind() == reflect.Slice {
			val, err = p.buildSliceArg(argVal, argType)
		} else {
			val, err = p.buildArg(argVal, argType)
		}

		if err != nil {
			return nil, fmt.Errorf("invalid argument at position %v: %w", DSLArgumentIndex, err)
		}
		args = append(args, reflect.ValueOf(val))

		DSLArgumentIndex++
	}

	if len(inv.Arguments) > DSLArgumentIndex {
		return nil, fmt.Errorf("too many arguments")
	}

	return args, nil
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
	default:
		return nil, fmt.Errorf("unsupported slice type '%s' for function", argType.Elem().Name())
	}
}

// Handle interfaces that can be passed as arguments to OTTL function invocations.
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
		return StandardTypeGetter[K, string]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "IntGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardTypeGetter[K, int64]{Getter: arg.Get}, nil
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

// Handle interfaces that can be declared as parameters to a OTTL function, but will
// never be called in an invocation. Returns whether the arg is an internal arg.
func (p *Parser[K]) buildInternalArg(argType reflect.Type) (any, bool) {
	if argType.Name() == "TelemetrySettings" {
		return p.telemetrySettings, true
	}
	return nil, false
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
