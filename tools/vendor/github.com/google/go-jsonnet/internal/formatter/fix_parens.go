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

// FixParens is a formatter pass that replaces ((e)) with (e).
type FixParens struct {
	pass.Base
}

// Parens handles that type of node
func (c *FixParens) Parens(p pass.ASTPass, node *ast.Parens, ctx pass.Context) {
	innerParens, ok := node.Inner.(*ast.Parens)
	if ok {
		node.Inner = innerParens.Inner
		ast.FodderMoveFront(openFodder(node), &innerParens.Fodder)
		ast.FodderMoveFront(&node.CloseFodder, &innerParens.CloseFodder)
	}
	c.Base.Parens(p, node, ctx)
}
