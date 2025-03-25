package kong

import (
	"fmt"
	"reflect"
	"strings"
)

// binding is a single binding registered with Kong.
type binding struct {
	// fn is a function that returns a value of the target type.
	fn reflect.Value

	// val is a value of the target type.
	// Must be set if done and singleton are true.
	val reflect.Value

	// singleton indicates whether the binding is a singleton.
	// If true, the binding will be resolved once and cached.
	singleton bool

	// done indicates whether a singleton binding has been resolved.
	// If singleton is false, this field is ignored.
	done bool
}

// newValueBinding builds a binding with an already resolved value.
func newValueBinding(v reflect.Value) *binding {
	return &binding{val: v, done: true, singleton: true}
}

// newFunctionBinding builds a binding with a function
// that will return a value of the target type.
//
// The function signature must be func(...) (T, error) or func(...) T
// where parameters are recursively resolved.
func newFunctionBinding(f reflect.Value, singleton bool) *binding {
	return &binding{fn: f, singleton: singleton}
}

// Get returns the pre-resolved value for the binding,
// or false if the binding is not resolved.
func (b *binding) Get() (v reflect.Value, ok bool) {
	return b.val, b.done
}

// Set sets the value of the binding to the given value,
// marking it as resolved.
//
// If the binding is not a singleton, this method does nothing.
func (b *binding) Set(v reflect.Value) {
	if b.singleton {
		b.val = v
		b.done = true
	}
}

// A map of type to function that returns a value of that type.
//
// The function should have the signature func(...) (T, error). Arguments are recursively resolved.
type bindings map[reflect.Type]*binding

func (b bindings) String() string {
	out := []string{}
	for k := range b {
		out = append(out, k.String())
	}
	return "bindings{" + strings.Join(out, ", ") + "}"
}

func (b bindings) add(values ...any) bindings {
	for _, v := range values {
		val := reflect.ValueOf(v)
		b[val.Type()] = newValueBinding(val)
	}
	return b
}

func (b bindings) addTo(impl, iface any) {
	val := reflect.ValueOf(impl)
	b[reflect.TypeOf(iface).Elem()] = newValueBinding(val)
}

func (b bindings) addProvider(provider any, singleton bool) error {
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
	b[rt] = newFunctionBinding(pv, singleton)
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
func getMethods(value reflect.Value, name string) (methods []reflect.Value) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if !value.IsValid() {
		return
	}

	if method := getMethod(value, name); method.IsValid() {
		methods = append(methods, method)
	}

	if value.Kind() != reflect.Struct {
		return
	}
	// If the current value is a struct, also consider embedded fields.
	// Two kinds of embedded fields are considered if they're exported:
	//
	//   - standard Go embedded fields
	//   - fields tagged with `embed:""`
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
			methods = append(methods, getMethods(fieldValue, name)...)
		}
	}
	return
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
		binding, ok := bindings[pt]
		if !ok {
			return nil, fmt.Errorf("couldn't find binding of type %s for parameter %d of %s(), use kong.Bind(%s)", pt, i, t, pt)
		}

		// Don't need to call the function if the value is already resolved.
		if val, ok := binding.Get(); ok {
			in = append(in, val)
			continue
		}

		// Recursively resolve binding functions.
		argv, err := callAnyFunction(binding.fn, bindings)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pt, err)
		}
		if ferrv := reflect.ValueOf(argv[len(argv)-1]); ferrv.IsValid() && ferrv.Type().Implements(callbackReturnSignature) && !ferrv.IsNil() {
			return nil, ferrv.Interface().(error) //nolint:forcetypeassert
		}

		val := reflect.ValueOf(argv[0])
		binding.Set(val)
		in = append(in, val)
	}
	outv := f.Call(in)
	out = make([]any, len(outv))
	for i, v := range outv {
		out[i] = v.Interface()
	}
	return out, nil
}
