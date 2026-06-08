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

// EnforceStringStyle is a formatter pass that manages string literals
type EnforceStringStyle struct {
	pass.Base
	Options Options
}

// LiteralString implements this pass.
func (c *EnforceStringStyle) LiteralString(p pass.ASTPass, lit *ast.LiteralString, ctx pass.Context) {
	if lit.Kind == ast.StringBlock {
		return
	}
	if lit.Kind == ast.VerbatimStringDouble {
		return
	}
	if lit.Kind == ast.VerbatimStringSingle {
		return
	}

	canonical, err := parser.StringUnescape(lit.Loc(), lit.Value)
	if err != nil {
		panic("Badly formatted string, should have been caught in lexer.")
	}
	numSingle := 0
	numDouble := 0
	for _, r := range canonical {
		if r == '\'' {
			numSingle++
		}
		if r == '"' {
			numDouble++
		}
	}
	if numSingle > 0 && numDouble > 0 {
		return // Don't change it.
	}
	useSingle := c.Options.StringStyle == StringStyleSingle

	if numSingle > 0 {
		useSingle = false
	}
	if numDouble > 0 {
		useSingle = true
	}

	// Change it.
	lit.Value = parser.StringEscape(canonical, useSingle)
	if useSingle {
		lit.Kind = ast.StringSingle
	} else {
		lit.Kind = ast.StringDouble
	}
}
