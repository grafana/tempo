// Copyright (c) 2023 Alexey Mayshev and contributors. All rights reserved.
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
	"math"
	"math/rand/v2"
	"runtime"
	"time"
	"unsafe"
)

const (
	// CacheLineSize is useful for preventing false sharing.
	CacheLineSize = 64

	MaxDuration = time.Duration(math.MaxInt64)
)

// Parallelism returns the maximum possible number of concurrently running goroutines.
func Parallelism() uint32 {
	//nolint:gosec // there will never be an overflow
	maxProcs := uint32(runtime.GOMAXPROCS(0))
	//nolint:gosec // there will never be an overflow
	numCPU := uint32(runtime.NumCPU())
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func Fastrand() uint32 {
	//nolint:gosec // we don't need a cryptographically secure random number generator
	return rand.Uint32()
}

// StackHash returns a hash of the calling goroutine's stack address.
//
// It stays the same while the goroutine runs and differs between goroutines that run at
// the same time, so it can spread concurrent operations over stripes without any
// per-goroutine state. A stack that grows is copied elsewhere and gets a new hash, which
// only moves the goroutine to another stripe.
func StackHash() uint32 {
	var x byte
	return hashStack(uintptr(unsafe.Pointer(&x)))
}

func hashStack(addr uintptr) uint32 {
	// The bits below the minimum stack size (2 KiB) depend on the call depth rather than
	// on the goroutine, so they are dropped. Stacks are aligned to their power-of-two
	// size, so the addresses of neighboring stacks may differ only in their upper bits,
	// and the hash has to mix those into the low bits that pick a stripe.
	return uint32(hashUint64(0, uint64(addr)>>11))
}
