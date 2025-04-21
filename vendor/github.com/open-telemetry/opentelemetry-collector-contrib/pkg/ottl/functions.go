// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

type PathExpressionParser[K any] func(Path[K]) (GetSetter[K], error)

type EnumParser func(*EnumSymbol) (*Enum, error)

type Enum int64

type EnumSymbol string

func buildOriginalText(path *path) string {
	var builder strings.Builder
	if path.Context != "" {
		builder.WriteString(path.Context)
		if len(path.Fields) > 0 {
			builder.WriteString(".")
		}
	}
	for i, f := range path.Fields {
		builder.WriteString(f.Name)
		if len(f.Keys) > 0 {
			builder.WriteString(buildOriginalKeysText(f.Keys))
		}
		if i != len(path.Fields)-1 {
			builder.WriteString(".")
		}
	}
	return builder.String()
}

func buildOriginalKeysText(keys []key) string {
	var builder strings.Builder
	if len(keys) > 0 {
		for _, k := range keys {
			builder.WriteString("[")
			if k.Int != nil {
				builder.WriteString(strconv.FormatInt(*k.Int, 10))
			}
			if k.String != nil {
				builder.WriteString(*k.String)
			}
			if k.Expression != nil {
				if k.Expression.Path != nil {
					builder.WriteString(buildOriginalText(k.Expression.Path))
				}
				if k.Expression.Float != nil {
					builder.WriteString(strconv.FormatFloat(*k.Expression.Float, 'f', 10, 64))
				}
				if k.Expression.Int != nil {
					builder.WriteString(strconv.FormatInt(*k.Expression.Int, 10))
				}
			}
			builder.WriteString("]")
		}
	}
	return builder.String()
}

func (p *Parser[K]) newPath(path *path) (*basePath[K], error) {
	if len(path.Fields) == 0 {
		return nil, errors.New("cannot make a path from zero fields")
	}

	pathContext, fields, err := p.parsePathContext(path)
	if err != nil {
		return nil, err
	}

	originalText := buildOriginalText(path)
	var current *basePath[K]
	for i := len(fields) - 1; i >= 0; i-- {
		keys, err := p.newKeys(fields[i].Keys)
		if err != nil {
			return nil, err
		}
		current = &basePath[K]{
			context:      pathContext,
			name:         fields[i].Name,
			keys:         keys,
			nextPath:     current,
			originalText: originalText,
		}
	}
	current.fetched = true
	current.originalText = originalText
	return current, nil
}

func (p *Parser[K]) parsePathContext(path *path) (string, []field, error) {
	hasPathContextNames := len(p.pathContextNames) > 0
	if path.Context != "" {
		// no pathContextNames means the Parser isn't handling the grammar path's context yet,
		// so it falls back to the previous behavior with the path.Context value as the first
		// path's segment.
		if !hasPathContextNames {
			return "", append([]field{{Name: path.Context}}, path.Fields...), nil
		}

		if _, ok := p.pathContextNames[path.Context]; !ok {
			return "", path.Fields, fmt.Errorf(`context "%s" from path "%s" is not valid, it must be replaced by one of: %s`, path.Context, buildOriginalText(path), p.buildPathContextNamesText(""))
		}

		return path.Context, path.Fields, nil
	}

	if hasPathContextNames {
		originalText := buildOriginalText(path)
		return "", nil, fmt.Errorf(`missing context name for path "%s", possibly valid options are: %s`, originalText, p.buildPathContextNamesText(originalText))
	}

	return "", path.Fields, nil
}

func (p *Parser[K]) buildPathContextNamesText(path string) string {
	var builder strings.Builder
	var suffix string
	if path != "" {
		suffix = "." + path
	}

	i := 0
	for ctx := range p.pathContextNames {
		builder.WriteString(fmt.Sprintf(`"%s%s"`, ctx, suffix))
		if i != len(p.pathContextNames)-1 {
			builder.WriteString(", ")
		}
		i++
	}
	return builder.String()
}

// Path represents a chain of path parts in an OTTL statement, such as `body.string`.
// A Path has a name, and potentially a set of keys.
// If the path in the OTTL statement contains multiple parts (separated by a dot (`.`)), then the Path will have a pointer to the next Path.
type Path[K any] interface {
	// Context is the OTTL context name of this Path.
	Context() string

	// Name is the name of this segment of the path.
	Name() string

	// Next provides the next path segment for this Path.
	// Will return nil if there is no next path.
	Next() Path[K]

	// Keys provides the Keys for this Path.
	// Will return nil if there are no Keys.
	Keys() []Key[K]

	// String returns a string representation of this Path and the next Paths
	String() string
}

var _ Path[any] = &basePath[any]{}

type basePath[K any] struct {
	context      string
	name         string
	keys         []Key[K]
	nextPath     *basePath[K]
	fetched      bool
	fetchedKeys  bool
	originalText string
}

func (p *basePath[K]) Context() string {
	return p.context
}

func (p *basePath[K]) Name() string {
	return p.name
}

func (p *basePath[K]) Next() Path[K] {
	if p.nextPath == nil {
		return nil
	}
	p.nextPath.fetched = true
	return p.nextPath
}

func (p *basePath[K]) Keys() []Key[K] {
	if p.keys == nil {
		return nil
	}
	p.fetchedKeys = true
	return p.keys
}

func (p *basePath[K]) String() string {
	return p.originalText
}

func (p *basePath[K]) isComplete() error {
	if !p.fetched {
		return fmt.Errorf("the path section %q was not used by the context - this likely means you are using extra path sections", p.name)
	}
	if p.keys != nil && !p.fetchedKeys {
		return fmt.Errorf("the keys indexing %q were not used by the context - this likely means you are trying to index a path that does not support indexing", p.name)
	}
	if p.nextPath == nil {
		return nil
	}
	return p.nextPath.isComplete()
}

func (p *Parser[K]) newKeys(keys []key) ([]Key[K], error) {
	if len(keys) == 0 {
		return nil, nil
	}
	ks := make([]Key[K], len(keys))
	for i := range keys {
		var getter Getter[K]
		if keys[i].Expression != nil {
			if keys[i].Expression.Path != nil {
				g, err := p.buildGetSetterFromPath(keys[i].Expression.Path)
				if err != nil {
					return nil, err
				}
				getter = g
			}
			if keys[i].Expression.Converter != nil {
				g, err := p.newGetterFromConverter(*keys[i].Expression.Converter)
				if err != nil {
					return nil, err
				}
				getter = g
			}
		}
		if keys[i].MathExpression != nil {
			g, err := p.evaluateMathExpression(keys[i].MathExpression)
			if err != nil {
				return nil, err
			}
			getter = g
		}
		ks[i] = &baseKey[K]{
			s: keys[i].String,
			i: keys[i].Int,
			g: getter,
		}
	}
	return ks, nil
}

// Key represents a chain of keys in an OTTL statement, such as `attributes["foo"]["bar"]`.
// A Key has a String or Int, and potentially the next Key.
// If the path in the OTTL statement contains multiple keys, then the Key will have a pointer to the next Key.
type Key[K any] interface {
	// String returns a pointer to the Key's string value.
	// If the Key does not have a string value the returned value is nil.
	// If Key experiences an error retrieving the value it is returned.
	String(context.Context, K) (*string, error)

	// Int returns a pointer to the Key's int value.
	// If the Key does not have a int value the returned value is nil.
	// If Key experiences an error retrieving the value it is returned.
	Int(context.Context, K) (*int64, error)

	// ExpressionGetter returns a Getter to the expression, that can be
	// part of the path.
	// If the Key does not have an expression the returned value is nil.
	// If Key experiences an error retrieving the value it is returned.
	ExpressionGetter(context.Context, K) (Getter[K], error)
}

var _ Key[any] = &baseKey[any]{}

type baseKey[K any] struct {
	s *string
	i *int64
	g Getter[K]
}

func (k *baseKey[K]) String(_ context.Context, _ K) (*string, error) {
	return k.s, nil
}

func (k *baseKey[K]) Int(_ context.Context, _ K) (*int64, error) {
	return k.i, nil
}

func (k *baseKey[K]) ExpressionGetter(_ context.Context, _ K) (Getter[K], error) {
	return k.g, nil
}

func (p *Parser[K]) parsePath(ip *basePath[K]) (GetSetter[K], error) {
	g, err := p.pathParser(ip)
	if err != nil {
		return nil, err
	}
	err = ip.isComplete()
	if err != nil {
		return nil, err
	}
	return g, nil
}

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
			case arg.FunctionName != nil:
				name = *arg.FunctionName
			default:
				return errors.New("invalid function name given")
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
			return nil, errors.New("slice parameter must be a byte slice literal")
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

func (p *Parser[K]) buildGetSetterFromPath(path *path) (GetSetter[K], error) {
	np, err := p.newPath(path)
	if err != nil {
		return nil, err
	}
	arg, err := p.parsePath(np)
	if err != nil {
		return nil, err
	}
	return arg, nil
}

// Handle interfaces that can be passed as arguments to OTTL functions.
func (p *Parser[K]) buildArg(argVal value, argType reflect.Type) (any, error) {
	name := argType.Name()
	switch {
	case strings.HasPrefix(name, "Setter"):
		fallthrough
	case strings.HasPrefix(name, "GetSetter"):
		if argVal.Literal != nil && argVal.Literal.Path != nil {
			return p.buildGetSetterFromPath(argVal.Literal.Path)
		}
		return nil, errors.New("must be a path")
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
	case strings.HasPrefix(name, "BoolGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardBoolGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "BoolLikeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardBoolLikeGetter[K]{Getter: arg.Get}, nil
	case strings.HasPrefix(name, "ByteSliceLikeGetter"):
		arg, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return StandardByteSliceLikeGetter[K]{Getter: arg.Get}, nil
	case name == "Enum":
		arg, err := p.enumParser((*EnumSymbol)(argVal.Enum))
		if err != nil {
			return nil, errors.New("must be an Enum")
		}
		return *arg, nil
	case name == reflect.String.String():
		if argVal.String == nil {
			return nil, errors.New("must be a string")
		}
		return *argVal.String, nil
	case name == reflect.Float64.String():
		if argVal.Literal == nil || argVal.Literal.Float == nil {
			return nil, errors.New("must be a float")
		}
		return *argVal.Literal.Float, nil
	case name == reflect.Int64.String():
		if argVal.Literal == nil || argVal.Literal.Int == nil {
			return nil, errors.New("must be an int")
		}
		return *argVal.Literal.Int, nil
	case name == reflect.Bool.String():
		if argVal.Bool == nil {
			return nil, errors.New("must be a bool")
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
