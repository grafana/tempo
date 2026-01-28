/*
Copyright 2019 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package formatter

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/pass"
)

// NoRedundantSliceColon is a formatter pass that preserves fodder in the case
// of arr[1::] being formatted as arr[1:]
type NoRedundantSliceColon struct {
	pass.Base
}

// Slice implements this pass.
func (c *NoRedundantSliceColon) Slice(p pass.ASTPass, slice *ast.Slice, ctx pass.Context) {
	if slice.Step == nil {
		if len(slice.StepColonFodder) > 0 {
			ast.FodderMoveFront(&slice.RightBracketFodder, &slice.StepColonFodder)
		}
	}
	c.Base.Slice(p, slice, ctx)
}
