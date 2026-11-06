// Copyright (c) 2024 Alexey Mayshev and contributors. All rights reserved.
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

package hashmap

import "unsafe"

// mapNode is comparable so that the map can tell a missing node by comparing it with the zero
// value of N. Every node type is a pointer or an interface holding one.
type mapNode interface {
	comparable
	AsPointer() unsafe.Pointer
}

type mapNodeManager[N mapNode] interface {
	FromPointer(ptr unsafe.Pointer) N
	// KeyOffset returns the offset of the key in a node.
	KeyOffset() uintptr
}
