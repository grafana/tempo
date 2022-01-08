// Copyright (c) 2019 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package setupcontext

var isAllInOne bool

// SetAllInOne sets the internal flag to all in one on.
func SetAllInOne() {
	isAllInOne = true
}

// UnsetAllInOne unsets the internal all-in-one flag.
func UnsetAllInOne() {
	isAllInOne = false
}

// IsAllInOne returns true when all in one mode is on.
func IsAllInOne() bool {
	return isAllInOne
}
