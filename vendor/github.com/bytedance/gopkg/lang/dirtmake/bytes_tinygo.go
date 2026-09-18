// Copyright 2024 ByteDance Inc.
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

//go:build tinygo

package dirtmake

// Bytes allocates a byte slice for TinyGo environment.
// Uses make() to avoid go:linkname with runtime.mallocgc, which causes
// "undefined symbol: runtime.mallocgc" errors in TinyGo.
func Bytes(len, cap int) (b []byte) {
	return make([]byte, len, cap)
}
