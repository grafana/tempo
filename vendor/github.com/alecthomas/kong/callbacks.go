package kong

import (
	"fmt"
	"reflect"
	"strings"
)

type bindings map[reflect.Type]func() (reflect.Value, error)

func (b bindings) String() string {
	out := []string{}
	for k := range b {
		out = append(out, k.String())
	}
	return "bindings{" + strings.Join(out, ", ") + "}"
}

func (b bindings) add(values ...interface{}) bindings {
	for _, v := range values {
		v := v
		b[reflect.TypeOf(v)] = func() (reflect.Value, error) { return reflect.ValueOf(v), nil }
	}
	return b
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

func callMethod(name string, v, f reflect.Value, bindings bindings) error {
	in := []reflect.Value{}
	t := f.Type()
	if t.NumOut() != 1 || t.Out(0) != callbackReturnSignature {
		return fmt.Errorf("return value of %T.%s() must be exactly \"error\"", v.Type(), name)
	}
	for i := 0; i < t.NumIn(); i++ {
		pt := t.In(i)
		if argf, ok := bindings[pt]; ok {
			argv, err := argf()
			if err != nil {
				return err
			}
			in = append(in, argv)
		} else {
			return fmt.Errorf("couldn't find binding of type %s for parameter %d of %s.%s(), use kong.Bind(%s)", pt, i, v.Type(), name, pt)
		}
	}
	out := f.Call(in)
	if out[0].IsNil() {
		return nil
	}
	return out[0].Interface().(error)
}
