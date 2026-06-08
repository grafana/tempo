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

// StripComments removes all comments
type StripComments struct {
	pass.Base
}

// Fodder implements this pass.
func (c *StripComments) Fodder(p pass.ASTPass, fodder *ast.Fodder, ctx pass.Context) {
	newFodder := make(ast.Fodder, 0)
	for _, el := range *fodder {
		if el.Kind == ast.FodderLineEnd {
			newElement := el
			newElement.Comment = nil
			newFodder = append(newFodder, newElement)
		}
	}
	*fodder = newFodder
}

// StripEverything removes all comments and newlines
type StripEverything struct {
	pass.Base
}

// Fodder implements this pass.
func (c *StripEverything) Fodder(p pass.ASTPass, fodder *ast.Fodder, ctx pass.Context) {
	*fodder = nil
}

// StripAllButComments removes all comments and newlines
type StripAllButComments struct {
	pass.Base
	comments ast.Fodder
}

// Fodder remembers all the fodder in c.comments
func (c *StripAllButComments) Fodder(p pass.ASTPass, fodder *ast.Fodder, ctx pass.Context) {
	for _, el := range *fodder {
		if el.Kind == ast.FodderParagraph {
			c.comments = append(c.comments, ast.FodderElement{
				Kind:    ast.FodderParagraph,
				Comment: el.Comment,
			})
		} else if el.Kind == ast.FodderInterstitial {
			c.comments = append(c.comments, el)
			c.comments = append(c.comments, ast.FodderElement{
				Kind: ast.FodderLineEnd,
			})
		}
	}
	*fodder = nil
}

// File replaces the entire file with the remembered comments.
func (c *StripAllButComments) File(p pass.ASTPass, node *ast.Node, finalFodder *ast.Fodder) {
	c.Base.File(p, node, finalFodder)
	*node = &ast.LiteralNull{
		NodeBase: ast.NodeBase{
			LocRange: *(*node).Loc(),
			Fodder:   c.comments,
		},
	}
	*finalFodder = nil
}
