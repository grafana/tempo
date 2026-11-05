// Copyright (c) 2025 Alexey Mayshev and contributors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xruntime

import (
	"hash/maphash"
	"math/bits"
	"math/rand/v2"
	"reflect"
	"unsafe"
)

// Hasher hashes keys of type T with a random seed.
//
// Integer keys, including named integer types, skip maphash.Comparable and
// go through two rounds of multiply-xorshift mixing, which is several times
// cheaper and does not depend on the runtime's hash function for the type.
type Hasher[T comparable] struct {
	seed    maphash.Seed
	intSeed uint64
	isInt   bool
}

func NewHasher[T comparable]() Hasher[T] {
	return Hasher[T]{
		seed:    maphash.MakeSeed(),
		intSeed: rand.Uint64(),
		isInt:   isIntKind(reflect.TypeFor[T]().Kind()),
	}
}

func (h Hasher[T]) Hash(t T) uint64 {
	if h.isInt {
		return hashUint64(h.intSeed, toUint64(t))
	}
	return maphash.Comparable(h.seed, t)
}

func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

// hashUint64 mixes v with two rounds of multiply-xorshift. A single round lets
// keys that differ by certain XOR deltas collide in the low bits whatever the
// seed is, which makes the table easy to flood.
func hashUint64(seed, v uint64) uint64 {
	hi, lo := bits.Mul64(v^seed, 0x2d358dccaa6c78a5)
	hi, lo = bits.Mul64(hi^lo, 0x8bb84b93962eacc9)
	return hi ^ lo
}

// toUint64 reinterprets an integer key as uint64. The switch is resolved at
// compile time for each instantiation of T.
func toUint64[T any](t T) uint64 {
	switch unsafe.Sizeof(t) {
	case 8:
		return *(*uint64)(unsafe.Pointer(&t))
	case 4:
		return uint64(*(*uint32)(unsafe.Pointer(&t)))
	case 2:
		return uint64(*(*uint16)(unsafe.Pointer(&t)))
	case 1:
		return uint64(*(*uint8)(unsafe.Pointer(&t)))
	default:
		panic("unreachable")
	}
}
