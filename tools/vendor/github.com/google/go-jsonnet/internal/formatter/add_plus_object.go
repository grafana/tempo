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

// AddPlusObject is a formatter pass that replaces e {} with e + {}.
type AddPlusObject struct {
	pass.Base
}

// Visit replaces ApplyBrace with Binary node.
func (c *AddPlusObject) Visit(p pass.ASTPass, node *ast.Node, ctx pass.Context) {
	applyBrace, ok := (*node).(*ast.ApplyBrace)
	if ok {
		*node = &ast.Binary{
			NodeBase: applyBrace.NodeBase,
			Left:     applyBrace.Left,
			Op:       ast.BopPlus,
			Right:    applyBrace.Right,
		}
	}
	c.Base.Visit(p, node, ctx)
}
