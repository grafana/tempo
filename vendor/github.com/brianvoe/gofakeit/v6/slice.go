package gofakeit

import (
	"reflect"
)

// Slice fills built-in types and exported fields of a struct with random data.
func Slice(v any) { sliceFunc(globalFaker, v) }

// Slice fills built-in types and exported fields of a struct with random data.
func (f *Faker) Slice(v any) { sliceFunc(f, v) }

func sliceFunc(f *Faker, v any) {
	r(f, reflect.TypeOf(v), reflect.ValueOf(v), "", -1)
}
