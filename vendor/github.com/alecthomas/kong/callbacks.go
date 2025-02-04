package kong

import (
	"fmt"
	"reflect"
	"strings"
)

// A map of type to function that returns a value of that type.
//
// The function should have the signature func(...) (T, error). Arguments are recursively resolved.
type bindings map[reflect.Type]any

func (b bindings) String() string {
	out := []string{}
	for k := range b {
		out = append(out, k.String())
	}
	return "bindings{" + strings.Join(out, ", ") + "}"
}

func (b bindings) add(values ...any) bindings {
	for _, v := range values {
		v := v
		b[reflect.TypeOf(v)] = func() (any, error) { return v, nil }
	}
	return b
}

func (b bindings) addTo(impl, iface any) {
	b[reflect.TypeOf(iface).Elem()] = func() (any, error) { return impl, nil }
}

func (b bindings) addProvider(provider any) error {
	pv := reflect.ValueOf(provider)
	t := pv.Type()
	if t.Kind() != reflect.Func {
		return fmt.Errorf("%T must be a function", provider)
	}

	if t.NumOut() == 0 {
		return fmt.Errorf("%T must be a function with the signature func(...)(T, error) or func(...) T", provider)
	}
	if t.NumOut() == 2 {
		if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			return fmt.Errorf("missing error; %T must be a function with the signature func(...)(T, error) or func(...) T", provider)
		}
	}
	rt := pv.Type().Out(0)
	b[rt] = provider
	return nil
}

// Clone and add values.
func (b bindings) clone() bindings {
	out := make(bindings, len(b))
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (b bindings) merge(other bindings) bindings {
	for k, v := range other {
		b[k] = v
	}
	return b
}

func getMethod(value reflect.Value, name string) reflect.Value {
	method := value.MethodByName(name)
	if !method.IsValid() {
		if value.CanAddr() {
			method = value.Addr().MethodByName(name)
		}
	}
	return method
}

// getMethods gets all methods with the given name from the given value
// and any embedded fields.
//
// Returns a slice of bound methods that can be called directly.
func getMethods(value reflect.Value, name string) []reflect.Value {
	// Traverses embedded fields of the struct
	// starting from the given value to collect all possible receivers
	// for the given method name.
	var traverse func(value reflect.Value, receivers []reflect.Value) []reflect.Value
	traverse = func(value reflect.Value, receivers []reflect.Value) []reflect.Value {
		// Always consider the current value for hooks.
		receivers = append(receivers, value)

		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		// If the current value is a struct, also consider embedded fields.
		// Two kinds of embedded fields are considered if they're exported:
		//
		//   - standard Go embedded fields
		//   - fields tagged with `embed:""`
		if value.Kind() == reflect.Struct {
			t := value.Type()
			for i := 0; i < value.NumField(); i++ {
				fieldValue := value.Field(i)
				field := t.Field(i)

				if !field.IsExported() {
					continue
				}

				// Consider a field embedded if it's actually embedded
				// or if it's tagged with `embed:""`.
				_, isEmbedded := field.Tag.Lookup("embed")
				isEmbedded = isEmbedded || field.Anonymous
				if isEmbedded {
					receivers = traverse(fieldValue, receivers)
				}
			}
		}

		return receivers
	}

	receivers := traverse(value, nil /* receivers */)

	// Search all receivers for methods
	var methods []reflect.Value
	for _, receiver := range receivers {
		if method := getMethod(receiver, name); method.IsValid() {
			methods = append(methods, method)
		}
	}
	return methods
}

func callFunction(f reflect.Value, bindings bindings) error {
	if f.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %s", f.Type())
	}
	t := f.Type()
	if t.NumOut() != 1 || !t.Out(0).Implements(callbackReturnSignature) {
		return fmt.Errorf("return value of %s must implement \"error\"", t)
	}
	out, err := callAnyFunction(f, bindings)
	if err != nil {
		return err
	}
	ferr := out[0]
	if ferrv := reflect.ValueOf(ferr); !ferrv.IsValid() || ((ferrv.Kind() == reflect.Interface || ferrv.Kind() == reflect.Pointer) && ferrv.IsNil()) {
		return nil
	}
	return ferr.(error) //nolint:forcetypeassert
}

func callAnyFunction(f reflect.Value, bindings bindings) (out []any, err error) {
	if f.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected function, got %s", f.Type())
	}
	in := []reflect.Value{}
	t := f.Type()
	for i := 0; i < t.NumIn(); i++ {
		pt := t.In(i)
		argf, ok := bindings[pt]
		if !ok {
			return nil, fmt.Errorf("couldn't find binding of type %s for parameter %d of %s(), use kong.Bind(%s)", pt, i, t, pt)
		}
		// Recursively resolve binding functions.
		argv, err := callAnyFunction(reflect.ValueOf(argf), bindings)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pt, err)
		}
		if ferrv := reflect.ValueOf(argv[len(argv)-1]); ferrv.IsValid() && ferrv.Type().Implements(callbackReturnSignature) && !ferrv.IsNil() {
			return nil, ferrv.Interface().(error) //nolint:forcetypeassert
		}
		in = append(in, reflect.ValueOf(argv[0]))
	}
	outv := f.Call(in)
	out = make([]any, len(outv))
	for i, v := range outv {
		out[i] = v.Interface()
	}
	return out, nil
}
