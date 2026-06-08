package funk

import (
	"reflect"
	"strings"
)

type JoinFnc func(lx, rx reflect.Value) reflect.Value

// Join combines two collections using the given join method.
func Join(larr, rarr interface{}, fnc JoinFnc) interface{} {
	if !IsCollection(larr) {
		panic("First parameter must be a collection")
	}
	if !IsCollection(rarr) {
		panic("Second parameter must be a collection")
	}

	lvalue := reflect.ValueOf(larr)
	rvalue := reflect.ValueOf(rarr)
	if NotEqual(lvalue.Type(), rvalue.Type()) {
		panic("Parameters must have the same type")
	}

	return fnc(lvalue, rvalue).Interface()
}

// InnerJoin finds and returns matching data from two collections.
func InnerJoin(lx, rx reflect.Value) reflect.Value {
	result := reflect.MakeSlice(reflect.SliceOf(lx.Type().Elem()), 0, lx.Len()+rx.Len())
	rhash := hashSlice(rx)
	lhash := make(map[interface{}]struct{}, lx.Len())

	for i := 0; i < lx.Len(); i++ {
		v := lx.Index(i)
		_, ok := rhash[v.Interface()]
		_, alreadyExists := lhash[v.Interface()]
		if ok && !alreadyExists {
			lhash[v.Interface()] = struct{}{}
			result = reflect.Append(result, v)
		}
	}
	return result
}

// OuterJoin finds and returns dissimilar data from two collections.
func OuterJoin(lx, rx reflect.Value) reflect.Value {
	ljoin := LeftJoin(lx, rx)
	rjoin := RightJoin(lx, rx)

	result := reflect.MakeSlice(reflect.SliceOf(lx.Type().Elem()), ljoin.Len()+rjoin.Len(), ljoin.Len()+rjoin.Len())
	for i := 0; i < ljoin.Len(); i++ {
		result.Index(i).Set(ljoin.Index(i))
	}
	for i := 0; i < rjoin.Len(); i++ {
		result.Index(ljoin.Len() + i).Set(rjoin.Index(i))
	}

	return result
}

// LeftJoin finds and returns dissimilar data from the first collection (left).
func LeftJoin(lx, rx reflect.Value) reflect.Value {
	result := reflect.MakeSlice(reflect.SliceOf(lx.Type().Elem()), 0, lx.Len())
	rhash := hashSlice(rx)

	for i := 0; i < lx.Len(); i++ {
		v := lx.Index(i)
		_, ok := rhash[v.Interface()]
		if !ok {
			result = reflect.Append(result, v)
		}
	}
	return result
}

// LeftJoin finds and returns dissimilar data from the second collection (right).
func RightJoin(lx, rx reflect.Value) reflect.Value { return LeftJoin(rx, lx) }

func hashSlice(arr reflect.Value) map[interface{}]struct{} {
	hash := map[interface{}]struct{}{}
	for i := 0; i < arr.Len(); i++ {
		v := arr.Index(i).Interface()
		hash[v] = struct{}{}
	}
	return hash
}

// StringerJoin joins an array of elements which implement the `String() string` function.
// Direct copy of strings.Join() with a few tweaks.
func StringerJoin(elems []interface{ String() string }, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0].String()
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i].String())
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0].String())
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(s.String())
	}
	return b.String()
}
