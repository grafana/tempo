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

func containsNewline(fodder ast.Fodder) bool {
	for _, f := range fodder {
		if f.Kind != ast.FodderInterstitial {
			return true
		}
	}
	return false
}

// FixTrailingCommas is a formatter pass that ensures trailing commas are
// present when a list is split over several lines.
type FixTrailingCommas struct {
	pass.Base
}

func (c *FixTrailingCommas) fixComma(lastCommaFodder *ast.Fodder, trailingComma *bool, closeFodder *ast.Fodder) {
	needComma := containsNewline(*closeFodder) || containsNewline(*lastCommaFodder)
	if *trailingComma {
		if !needComma {
			// Remove it but keep fodder.
			*trailingComma = false
			ast.FodderMoveFront(closeFodder, lastCommaFodder)
		} else if containsNewline(*lastCommaFodder) {
			// The comma is needed but currently is separated by a newline.
			ast.FodderMoveFront(closeFodder, lastCommaFodder)
		}
	} else {
		if needComma {
			// There was no comma, but there was a newline before the ] so add a comma.
			*trailingComma = true
		}
	}
}

func (c *FixTrailingCommas) removeComma(lastCommaFodder *ast.Fodder, trailingComma *bool, closeFodder *ast.Fodder) {
	if *trailingComma {
		// Remove it but keep fodder.
		*trailingComma = false
		ast.FodderMoveFront(closeFodder, lastCommaFodder)
	}
}

// Array handles that type of node
func (c *FixTrailingCommas) Array(p pass.ASTPass, node *ast.Array, ctx pass.Context) {
	if len(node.Elements) == 0 {
		// No comma present and none can be added.
		return
	}
	c.fixComma(&node.Elements[len(node.Elements)-1].CommaFodder, &node.TrailingComma, &node.CloseFodder)
	c.Base.Array(p, node, ctx)
}

// ArrayComp handles that type of node
func (c *FixTrailingCommas) ArrayComp(p pass.ASTPass, node *ast.ArrayComp, ctx pass.Context) {
	c.removeComma(&node.TrailingCommaFodder, &node.TrailingComma, &node.Spec.ForFodder)
	c.Base.ArrayComp(p, node, ctx)
}

// Object handles that type of node
func (c *FixTrailingCommas) Object(p pass.ASTPass, node *ast.Object, ctx pass.Context) {
	if len(node.Fields) == 0 {
		// No comma present and none can be added.
		return
	}
	c.fixComma(&node.Fields[len(node.Fields)-1].CommaFodder, &node.TrailingComma, &node.CloseFodder)
	c.Base.Object(p, node, ctx)
}

// ObjectComp handles that type of node
func (c *FixTrailingCommas) ObjectComp(p pass.ASTPass, node *ast.ObjectComp, ctx pass.Context) {
	c.removeComma(&node.Fields[len(node.Fields)-1].CommaFodder, &node.TrailingComma, &node.Spec.ForFodder)
	c.Base.ObjectComp(p, node, ctx)
}
