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

// RemovePlusObject is a formatter pass that replaces ((e)) with (e).
type RemovePlusObject struct {
	pass.Base
}

// Visit replaces e + { ... } with an ApplyBrace in some situations.
func (c *RemovePlusObject) Visit(p pass.ASTPass, node *ast.Node, ctx pass.Context) {
	binary, ok := (*node).(*ast.Binary)
	if ok {
		// Could relax this to allow more ASTs on the LHS but this seems OK for now.
		_, leftIsVar := binary.Left.(*ast.Var)
		_, leftIsIndex := binary.Left.(*ast.Index)
		if leftIsVar || leftIsIndex {
			rhs, ok := binary.Right.(*ast.Object)
			if ok && binary.Op == ast.BopPlus {
				ast.FodderMoveFront(&rhs.Fodder, &binary.OpFodder)
				*node = &ast.ApplyBrace{
					NodeBase: binary.NodeBase,
					Left:     binary.Left,
					Right:    rhs,
				}
			}
		}
	}
	c.Base.Visit(p, node, ctx)
}
