// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/iancoleman/strcase"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/lambda"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"
)

// PathExpressionParser is how a context provides OTTL access to all its Paths.
type PathExpressionParser[K any] func(Path[K]) (GetSetter[K], error)

// EnumParser is how a context provides OTTL access to all its Enums.
type EnumParser func(*EnumSymbol) (*Enum, error)

// Enum is how OTTL represents an enum's numeric value.
type Enum int64

// EnumSymbol is how OTTL represents an enum's string value.
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

func (p *parseContext[K]) newPath(path *path) (*basePath[K], error) {
	if len(path.Fields) == 0 {
		return nil, ottlerror.New(path.Pos, "cannot make a path from zero fields")
	}

	var err error
	var pathContext string
	var fields []field
	var localIdentifier bool

	if path.inScope(p.localScopes) {
		fields = path.dottedSegments()
		localIdentifier = true
	} else {
		pathContext, fields, err = p.parsePathContext(path)
		if err != nil {
			return nil, err
		}
	}

	originalText := buildOriginalText(path)
	var current *basePath[K]
	for i := range slices.Backward(fields) {
		keys, err := p.newKeys(fields[i].Keys)
		if err != nil {
			return nil, err
		}
		current = &basePath[K]{
			context:         pathContext,
			name:            fields[i].Name,
			keys:            keys,
			nextPath:        current,
			originalText:    originalText,
			localIdentifier: localIdentifier,
			pos:             fields[i].Pos,
		}
	}
	current.fetched = true
	current.originalText = originalText
	return current, nil
}

func (p *parseContext[K]) parsePathContext(path *path) (string, []field, error) {
	hasPathContextNames := len(p.pathContextNames) > 0
	if path.Context != "" {
		// no pathContextNames means the Parser isn't handling the grammar path's context yet,
		// so it falls back to the previous behavior with the path.Context value as the first
		// path's segment.
		if !hasPathContextNames {
			return "", path.dottedSegments(), nil
		}

		if _, ok := p.pathContextNames[path.Context]; !ok {
			return "", path.Fields, ottlerror.Errorf(path.Pos, `context "%s" from path "%s" is not valid, it must be replaced by one of: %s`, path.Context, buildOriginalText(path), p.buildPathContextNamesText(""))
		}

		return path.Context, path.Fields, nil
	}

	if hasPathContextNames {
		originalText := buildOriginalText(path)
		return "", nil, ottlerror.Errorf(path.Pos, `missing context name for path "%s", possibly valid options are: %s`, originalText, p.buildPathContextNamesText(originalText))
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
		fmt.Fprintf(&builder, `"%s%s"`, ctx, suffix)
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
	context         string
	name            string
	keys            []Key[K]
	nextPath        *basePath[K]
	fetched         bool
	fetchedKeys     bool
	originalText    string
	localIdentifier bool
	pos             lexer.Position
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
		return ottlerror.Errorf(p.pos, "the path section %q was not used by the context - this likely means you are using extra path sections", p.name)
	}
	if p.keys != nil && !p.fetchedKeys {
		return ottlerror.Errorf(p.pos, "the keys indexing %q were not used by the context - this likely means you are trying to index a path that does not support indexing", p.name)
	}
	if p.nextPath == nil {
		return nil
	}
	return p.nextPath.isComplete()
}

func (p *basePath[K]) validate(pc *parseContext[K]) error {
	if p.localIdentifier {
		if p.nextPath != nil {
			return ottlerror.Errorf(p.pos, `local identifier %q cannot use "." to access nested fields`, p.originalText)
		}
		if pc.telemetrySettings.Logger.Core().Enabled(zapcore.DebugLevel) {
			if _, ok := pc.pathContextNames[p.name]; ok {
				pc.telemetrySettings.Logger.Debug(fmt.Sprintf("in-scope local identifier %q shadows OTTL context %q", p.name, p.name))
			} else {
				if _, err := pc.pathParser(p); err == nil {
					pc.telemetrySettings.Logger.Debug(fmt.Sprintf("in-scope local identifier shadows the OTTL context path %q", p.name))
				}
			}
		}
	}
	return nil
}

func (p *parseContext[K]) newKeys(keys []key) ([]Key[K], error) {
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
				return nil, ottlerror.FromError(keys[i].Pos, err)
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

func (p *parseContext[K]) parsePath(ip *basePath[K]) (GetSetter[K], error) {
	if ip.localIdentifier {
		return p.newLocalIdentifierGetter(ip)
	}

	g, err := p.pathParser(ip)
	if err != nil {
		return nil, ottlerror.FromError(ip.pos, err)
	}
	err = ip.isComplete()
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (p *parseContext[K]) newFunctionCall(ed editor) (Expr[K], error) {
	f, ok := p.functions[ed.Function]
	if !ok {
		return Expr[K]{}, ottlerror.Errorf(ed.Pos, "undefined function %q", ed.Function)
	}
	if err := p.recordExperimentalFunc(f); err != nil {
		return Expr[K]{}, ottlerror.FromError(ed.Pos, err)
	}
	defaultArgs := f.CreateDefaultArguments()
	var args Arguments

	// A nil value indicates the function takes no arguments.
	if defaultArgs != nil {
		// Pointer values are necessary to fulfill the Go reflection
		// settability requirements. Non-pointer values are not
		// modifiable through reflection.
		if reflect.TypeOf(defaultArgs).Kind() != reflect.Pointer {
			return Expr[K]{}, ottlerror.Errorf(ed.Pos, "factory for %q must return a pointer to an Arguments value in its CreateDefaultArguments method", ed.Function)
		}

		args = reflect.New(reflect.ValueOf(defaultArgs).Elem().Type()).Interface()

		allowDynamicSlices := f.Experimental() || metadata.PkgOttlFunctionsEnableDynamicSliceArgumentsFeatureGate.IsEnabled()
		err := p.buildArgs(ed, reflect.ValueOf(args).Elem(), allowDynamicSlices)
		if err != nil {
			return Expr[K]{}, ottlerror.Wrapf(ed.Pos, err, "error while parsing arguments for call to %q", ed.Function)
		}
	}

	fn, err := f.CreateFunction(FunctionContext{Set: p.telemetrySettings}, args)
	if err != nil {
		return Expr[K]{}, ottlerror.Wrapf(ed.Pos, err, "couldn't create function")
	}

	return Expr[K]{exprFunc: fn}, err
}

func (p *parseContext[K]) buildArgs(ed editor, argsVal reflect.Value, allowDynamicSlices bool) error {
	requiredArgs := 0
	seenNamed := false

	for i := 0; i < len(ed.Arguments); i++ {
		if !seenNamed && ed.Arguments[i].Name != "" {
			seenNamed = true
		} else if seenNamed && ed.Arguments[i].Name == "" {
			return ottlerror.New(ed.Arguments[i].Pos, "unnamed argument used after named argument")
		}
	}

	for i := 0; i < argsVal.NumField(); i++ {
		if !strings.HasPrefix(argsVal.Field(i).Type().Name(), "Optional") {
			requiredArgs++
		}
	}

	if len(ed.Arguments) < requiredArgs || len(ed.Arguments) > argsVal.NumField() {
		return ottlerror.Errorf(ed.Pos, "incorrect number of arguments. Expected: %d Received: %d", argsVal.NumField(), len(ed.Arguments))
	}

	for i := range ed.Arguments {
		edArg := ed.Arguments[i]
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
				return ottlerror.Errorf(edArg.Pos, "no such parameter: %s", edArg.Name)
			}
			fieldType = field.Type()
			isOptional = strings.HasPrefix(fieldType.Name(), "Optional")
			arg = edArg
		}

		var val any
		var optionalArg reflectTypedArg
		var err error
		var ok bool
		if isOptional {
			optionalArg, ok = reflect.TypeAssert[reflectTypedArg](field.Addr())

			if !ok {
				return ottlerror.New(edArg.Pos, "optional type is not manageable by the OTTL parser. This is an error in the OTTL")
			}

			fieldType = optionalArg.reflectTypeParam()
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
				return ottlerror.New(edArg.Pos, "invalid function name given")
			}
			var f Factory[K]
			f, ok = p.functions[name]
			if !ok {
				return ottlerror.Errorf(edArg.Pos, "undefined function %s", name)
			}
			err = p.recordExperimentalFunc(f)
			if err != nil {
				return err
			}
			val = StandardFunctionGetter[K]{FCtx: FunctionContext{Set: p.telemetrySettings}, Fact: f}
		case strings.HasPrefix(fieldType.Name(), "SliceGetter"):
			var fieldAddr any
			if isOptional {
				fieldAddr = optionalArg.addrReflectValue()
			} else {
				fieldAddr = field.Addr().Interface()
			}
			val, err = p.buildSliceGetterArg(fieldAddr, arg, allowDynamicSlices)
		case fieldType.Kind() == reflect.Slice:
			val, err = p.buildSliceArg(arg.Value, fieldType)
		case fieldType.Kind() == reflect.Pointer:
			val, err = p.buildArg(arg.Value, fieldType.Elem())
		default:
			val, err = p.buildArg(arg.Value, fieldType)
		}
		if err != nil {
			return ottlerror.Wrapf(edArg.Pos, err, "invalid argument at position %v", i)
		}
		if isOptional {
			err = optionalArg.setReflectValue(reflect.ValueOf(val))
			if err != nil {
				return ottlerror.FromError(edArg.Pos, err)
			}
		} else {
			field.Set(reflect.ValueOf(val))
		}
	}

	return nil
}

func (p *parseContext[K]) buildSliceArg(argVal value, argType reflect.Type) (any, error) {
	name := argType.Elem().Name()
	switch {
	case name == reflect.Uint8.String():
		if argVal.Bytes == nil {
			return nil, ottlerror.New(argVal.Pos, "slice parameter must be a byte slice literal")
		}
		return []byte(*argVal.Bytes), nil
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
	case strings.HasPrefix(name, "PSliceGetter"):
		arg, err := buildSlice[PSliceGetter[K]](argVal, argType, p.buildArg, name)
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
		return nil, ottlerror.Errorf(argVal.Pos, "unsupported slice type %q for function", argType.Elem().Name())
	}
}

func (p *parseContext[K]) buildGetSetterFromPath(path *path) (GetSetter[K], error) {
	np, err := p.newPath(path)
	if err != nil {
		return nil, err
	}
	err = np.validate(p)
	if err != nil {
		return nil, err
	}
	arg, err := p.parsePath(np)
	if err != nil {
		return nil, err
	}
	return arg, nil
}

func (*parseContext[K]) buildStandardGetSetter(name string, valueGetter Getter[K]) (any, error) {
	switch {
	case
		strings.HasPrefix(name, "Getter"),
		strings.HasPrefix(name, "Setter"),
		strings.HasPrefix(name, "GetSetter"):
		return valueGetter, nil
	case strings.HasPrefix(name, "PMapGetSetter"):
		if setter, ok := any(valueGetter).(GetSetter[K]); ok {
			return newStandardPMapGetSetter(setter)
		}
		return nil, fmt.Errorf("type %q is not a GetSetter and cannot be used as PMapGetSetter", name)
	case strings.HasPrefix(name, "PSliceGetSetter"):
		if setter, ok := any(valueGetter).(GetSetter[K]); ok {
			return newStandardPSliceGetSetter(setter)
		}
		return nil, fmt.Errorf("type %q is not a GetSetter and cannot be used as PSliceGetSetter", name)
	case strings.HasPrefix(name, "StringGetter"):
		return newStandardStringGetter[K](valueGetter)
	case strings.HasPrefix(name, "StringLikeGetter"):
		return newStandardStringLikeGetter[K](valueGetter)
	case strings.HasPrefix(name, "FloatGetter"):
		return newStandardFloatGetter[K](valueGetter)
	case strings.HasPrefix(name, "FloatLikeGetter"):
		return newStandardFloatLikeGetter[K](valueGetter)
	case strings.HasPrefix(name, "IntGetter"):
		return newStandardIntGetter[K](valueGetter)
	case strings.HasPrefix(name, "IntLikeGetter"):
		return newStandardIntLikeGetter[K](valueGetter)
	case strings.HasPrefix(name, "PMapGetter"):
		return newStandardPMapGetter[K](valueGetter)
	case strings.HasPrefix(name, "PSliceGetter"):
		return newStandardPSliceGetter[K](valueGetter)
	case strings.HasPrefix(name, "DurationGetter"):
		return newStandardDurationGetter[K](valueGetter)
	case strings.HasPrefix(name, "TimeGetter"):
		return newStandardTimeGetter[K](valueGetter)
	case strings.HasPrefix(name, "BoolGetter"):
		return newStandardBoolGetter[K](valueGetter)
	case strings.HasPrefix(name, "BoolLikeGetter"):
		return newStandardBoolLikeGetter[K](valueGetter)
	case strings.HasPrefix(name, "ByteSliceLikeGetter"):
		return newStandardByteSliceLikeGetter[K](valueGetter)
	default:
		return nil, fmt.Errorf("unsupported argument type: %s", name)
	}
}

// Handle interfaces that can be passed as arguments to OTTL functions.
func (p *parseContext[K]) buildArg(argVal value, argType reflect.Type) (any, error) {
	name := argType.Name()
	switch {
	case name == "Enum":
		arg, err := p.enumParser((*EnumSymbol)(argVal.Enum))
		if err != nil {
			return nil, ottlerror.Wrap(argVal.Pos, err, "must be an Enum")
		}
		return *arg, nil
	case name == reflect.String.String():
		if argVal.String == nil {
			return nil, ottlerror.New(argVal.Pos, "must be a string")
		}
		return *argVal.String, nil
	case name == reflect.Float64.String():
		if argVal.Literal == nil || argVal.Literal.Float == nil {
			return nil, ottlerror.New(argVal.Pos, "must be a float")
		}
		return *argVal.Literal.Float, nil
	case name == reflect.Int64.String():
		if argVal.Literal == nil || argVal.Literal.Int == nil {
			return nil, ottlerror.New(argVal.Pos, "must be an int")
		}
		return *argVal.Literal.Int, nil
	case name == reflect.Bool.String():
		if argVal.Bool == nil {
			return nil, ottlerror.New(argVal.Pos, "must be a bool")
		}
		return bool(*argVal.Bool), nil
	case strings.HasPrefix(name, "LambdaExpression"):
		if argVal.Lambda == nil {
			return nil, ottlerror.New(argVal.Pos, "must be a lambda expression")
		}
		lambExpr, err := p.newLambdaExpression(argVal.Lambda)
		if err != nil {
			return nil, err
		}
		return lambExpr, nil
	case strings.HasSuffix(stripGenericArgs(name), "Setter"):
		if argVal.Literal == nil || argVal.Literal.Path == nil {
			return nil, ottlerror.New(argVal.Pos, "must be a path")
		}
		getter, err := p.buildGetSetterFromPath(argVal.Literal.Path)
		if err != nil {
			return nil, err
		}
		return p.buildStandardGetSetter(name, getter)
	default:
		getter, err := p.newGetter(argVal)
		if err != nil {
			return nil, err
		}
		return p.buildStandardGetSetter(name, getter)
	}
}

func stripGenericArgs(name string) string {
	baseName, _, _ := strings.Cut(name, "[")
	return baseName
}

type buildArgFunc func(value, reflect.Type) (any, error)

func buildSlice[T any](argVal value, argType reflect.Type, buildArg buildArgFunc, name string) ([]T, error) {
	if argVal.List == nil {
		return nil, ottlerror.Errorf(argVal.Pos, "must be a list of type %v", name)
	}

	vals := make([]T, 0, len(argVal.List.Values))
	values := argVal.List.Values
	for i := range values {
		untypedVal, err := buildArg(values[i], argType.Elem())
		if err != nil {
			return nil, ottlerror.Wrapf(values[i].Pos, err, "error while parsing list argument at index %v", i)
		}

		val, ok := untypedVal.(T)
		if !ok {
			return nil, ottlerror.Errorf(values[i].Pos, "invalid element type at list index %v, must be of type %v", i, name)
		}

		vals = append(vals, val)
	}

	return vals, nil
}

var errLambdaExpressionDisable = fmt.Errorf(
	"lambda expression arguments require the `%s` feature gate to be enabled",
	metadata.OttlFunctionsEnableLambdaFeatureGate.ID(),
)

func (p *parseContext[K]) newLambdaExpression(l *lambdaExpr) (*lambda.LambdaExpression[K], error) {
	if !metadata.OttlFunctionsEnableLambdaFeatureGate.IsEnabled() {
		return nil, ottlerror.FromError(l.Pos, errLambdaExpressionDisable)
	}

	formals := make([]string, len(l.Params))
	validFormals := make(localScopeFrame, len(l.Params))
	for i, param := range l.Params {
		name := param.Name()
		if !param.IsBlank() {
			if _, exists := validFormals[name]; exists {
				return nil, ottlerror.Errorf(param.Pos, "duplicate local identifier %q", name)
			}
			validFormals[name] = struct{}{}
		}
		formals[i] = name
	}

	var result *lambda.LambdaExpression[K]
	err := p.withLocalScope(validFormals, func() error {
		switch {
		case l.Body.Expr != nil && l.Body.Value != nil:
			return ottlerror.New(l.Body.Pos, "lambda cannot have both an expression and a value body, this is a programming error in OTTL")
		case l.Body.Expr != nil:
			bodyExpr, err := p.newBoolExpr(l.Body.Expr)
			if err != nil {
				return err
			}
			result = newLambdaExpression[K](formals, nil, bodyExpr)
			return nil
		case l.Body.Value != nil:
			body, err := p.newGetter(*l.Body.Value)
			if err != nil {
				return err
			}
			result = newLambdaExpression[K](formals, body, nil)
			return nil
		default:
			return ottlerror.New(l.Body.Pos, "lambda requires a valid body after =>")
		}
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// newLambdaExpression creates a new LambdaExpression. It must either have a body or a bodyExpr, but not both.
func newLambdaExpression[K any](formals []string, body Getter[K], bodyExpr boolExpr[K]) *lambda.LambdaExpression[K] {
	switch {
	case body != nil:
		if literal, ok := GetLiteralValue(body); ok {
			return lambda.NewLiteral[K](formals, literal)
		}
		return lambda.New(formals, body.Get)
	case bodyExpr != nil:
		if literal, ok := bodyExpr.(*literalBoolExpr[K]); ok {
			return lambda.NewLiteral[K](formals, literal.getValue())
		}
		return lambda.New(formals, func(ctx context.Context, tCtx K) (any, error) {
			return bodyExpr.Eval(ctx, tCtx)
		})
	default:
		return lambda.New[K](formals, nil)
	}
}

// reflectTypedArg is implemented by generic OTTL function argument types that expose
// their type parameter at runtime for reflection-based parsing.
type reflectTypedArg interface {
	// reflectTypeParam returns the generic type parameter at runtime.
	reflectTypeParam() reflect.Type
	// setReflectValue stores val into this argument.
	setReflectValue(val reflect.Value) error
	// addrReflectValue returns a pointer to the underlying stored value when it can be
	// modified in place. Returns nil if there is no addressable underlying value.
	addrReflectValue() any
}

// Optional is used to represent an optional function argument
type Optional[T any] struct {
	val      T
	hasValue bool
}

var _ reflectTypedArg = (*Optional[any])(nil)

func (*Optional[T]) reflectTypeParam() reflect.Type {
	return reflect.TypeFor[T]()
}

func (o *Optional[T]) setReflectValue(val reflect.Value) error {
	typedVal, ok := reflect.TypeAssert[T](val)
	if !ok {
		return fmt.Errorf("cannot set value of type %q to an Optional of type %q", val.Type(), reflect.TypeFor[T]())
	}
	o.val = typedVal
	o.hasValue = true
	return nil
}

func (o *Optional[T]) addrReflectValue() any {
	return &o.val
}

// IsEmpty returns true if the Optional[T] does not contain a value.
func (o *Optional[T]) IsEmpty() bool {
	return !o.hasValue
}

// Get returns the value contained in the Optional[T].
func (o *Optional[T]) Get() T {
	return o.val
}

// GetOr returns the value contained in the Optional[T] if it exists,
// otherwise it returns the default value provided.
func (o *Optional[T]) GetOr(value T) T {
	if !o.hasValue {
		return value
	}
	return o.val
}

// NewTestingOptional allows creating an Optional with a value already populated for use in testing
// OTTL functions.
func NewTestingOptional[T any](val T) Optional[T] {
	return Optional[T]{
		val:      val,
		hasValue: true,
	}
}

// TypedGetter is like Getter, but with typed return values.
type TypedGetter[K, V any] interface {
	Get(ctx context.Context, tCtx K) (V, error)
}

// mockLiteralGetter is a mock implementation of TypedGetter that can be used for testing.
type mockLiteralGetter[K, V any] struct {
	valueGetter func(context.Context, K) (V, error)
}

func (m mockLiteralGetter[K, V]) Get(_ context.Context, _ K) (V, error) {
	return m.valueGetter(context.Background(), *new(K))
}

// NewTestingLiteralGetter creates a mock literal getter for testing OTTL functions.
// Pass `literal` as true if the getter should be treated as a literal.
func NewTestingLiteralGetter[K, V any](literal bool, getter TypedGetter[K, V]) (TypedGetter[K, V], error) {
	if literal {
		val, err := getter.Get(context.Background(), *new(K))
		if err != nil {
			return nil, err
		}
		return newLiteral[K, V](val), nil
	}
	return mockLiteralGetter[K, V]{valueGetter: getter.Get}, nil
}

// OptionalGetter is like TypedGetter, but for getters whose Get also returns a found bool,
// such as the "Like" getters.
type OptionalGetter[K, V any] interface {
	Get(ctx context.Context, tCtx K) (V, bool, error)
}

// mockOptionalLiteralGetter is a mock implementation of an OptionalGetter literal for testing.
type mockOptionalLiteralGetter[K, V any] struct {
	valueGetter func(context.Context, K) (V, bool, error)
}

func (m mockOptionalLiteralGetter[K, V]) Get(_ context.Context, _ K) (V, bool, error) {
	return m.valueGetter(context.Background(), *new(K))
}

// NewTestingOptionalLiteralGetter creates a mock literal getter for testing OTTL functions that
// take a getter whose Get returns a found bool, such as the "Like" getters. Pass `literal` as
// true if the getter should be treated as a literal.
func NewTestingOptionalLiteralGetter[K, V any](literal bool, getter OptionalGetter[K, V]) (OptionalGetter[K, V], error) {
	if literal {
		val, found, err := getter.Get(context.Background(), *new(K))
		if err != nil {
			return nil, err
		}
		return newOptionalLiteral[K, V](val, found), nil
	}
	return mockOptionalLiteralGetter[K, V]{valueGetter: getter.Get}, nil
}
