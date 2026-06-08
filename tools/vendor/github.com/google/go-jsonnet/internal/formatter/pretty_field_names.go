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
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/internal/pass"
)

// PrettyFieldNames forces minimal syntax with field lookups and definitions
type PrettyFieldNames struct {
	pass.Base
}

// Index prettifies the definitions.
func (c *PrettyFieldNames) Index(p pass.ASTPass, index *ast.Index, ctx pass.Context) {
	if index.Index != nil {
		// Maybe we can use an id instead.
		lit, ok := index.Index.(*ast.LiteralString)
		if ok {
			if parser.IsValidIdentifier(lit.Value) {
				index.Index = nil
				id := ast.Identifier(lit.Value)
				index.Id = &id
				index.RightBracketFodder = lit.Fodder
			}
		}
	}
	c.Base.Index(p, index, ctx)
}

// ObjectField prettifies the definitions.
func (c *PrettyFieldNames) ObjectField(p pass.ASTPass, field *ast.ObjectField, ctx pass.Context) {
	if field.Kind == ast.ObjectFieldExpr {
		// First try ["foo"] -> "foo".
		lit, ok := field.Expr1.(*ast.LiteralString)
		if ok {
			field.Kind = ast.ObjectFieldStr
			ast.FodderMoveFront(&lit.Fodder, &field.Fodder1)
			if field.Method != nil {
				ast.FodderMoveFront(&field.Method.ParenLeftFodder, &field.Fodder2)
			} else {
				ast.FodderMoveFront(&field.OpFodder, &field.Fodder2)
			}
		}
	}
	if field.Kind == ast.ObjectFieldStr {
		// Then try "foo" -> foo.
		lit, ok := field.Expr1.(*ast.LiteralString)
		if ok {
			if parser.IsValidIdentifier(lit.Value) {
				field.Kind = ast.ObjectFieldID
				id := ast.Identifier(lit.Value)
				field.Id = &id
				field.Fodder1 = lit.Fodder
				field.Expr1 = nil
			}
		}
	}
	c.Base.ObjectField(p, field, ctx)
}
