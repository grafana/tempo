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

// FixNewlines is a formatter pass that adds newlines inside complex structures
// (arrays, objects etc.).
//
// The main principle is that a structure can either be:
// * expanded and contain newlines in all the designated places
// * unexpanded and contain newlines in none of the designated places
//
// It only looks shallowly at the AST nodes, so there may be some newlines deeper that
// don't affect expanding. For example:
// [{
//     'a': 'b',
//     'c': 'd',
// }]
// The outer array can stay unexpanded, because there are no newlines between
// the square brackets and the braces.
type FixNewlines struct {
	pass.Base
}

// Array handles this type of node
func (c *FixNewlines) Array(p pass.ASTPass, array *ast.Array, ctx pass.Context) {
	shouldExpand := false
	for _, element := range array.Elements {
		if ast.FodderCountNewlines(*openFodder(element.Expr)) > 0 {
			shouldExpand = true
		}
	}
	if ast.FodderCountNewlines(array.CloseFodder) > 0 {
		shouldExpand = true
	}
	if shouldExpand {
		for i := range array.Elements {
			ast.FodderEnsureCleanNewline(openFodder(array.Elements[i].Expr))
		}
		ast.FodderEnsureCleanNewline(&array.CloseFodder)
	}
	c.Base.Array(p, array, ctx)
}

func objectFieldOpenFodder(field *ast.ObjectField) *ast.Fodder {
	if field.Kind == ast.ObjectFieldStr {
		// This can only ever be a ast.sStringLiteral, so openFodder
		// will return without recursing.
		return openFodder(field.Expr1)
	}
	return &field.Fodder1
}

// Object handles this type of node
func (c *FixNewlines) Object(p pass.ASTPass, object *ast.Object, ctx pass.Context) {
	shouldExpand := false
	for _, field := range object.Fields {
		if ast.FodderCountNewlines(*objectFieldOpenFodder(&field)) > 0 {
			shouldExpand = true
		}
	}
	if ast.FodderCountNewlines(object.CloseFodder) > 0 {
		shouldExpand = true
	}
	if shouldExpand {
		for i := range object.Fields {
			ast.FodderEnsureCleanNewline(
				objectFieldOpenFodder(&object.Fields[i]))
		}
		ast.FodderEnsureCleanNewline(&object.CloseFodder)
	}
	c.Base.Object(p, object, ctx)
}

// Local handles this type of node
func (c *FixNewlines) Local(p pass.ASTPass, local *ast.Local, ctx pass.Context) {
	shouldExpand := false
	for _, bind := range local.Binds {
		if ast.FodderCountNewlines(bind.VarFodder) > 0 {
			shouldExpand = true
		}
	}
	if shouldExpand {
		for i := range local.Binds {
			if i > 0 {
				ast.FodderEnsureCleanNewline(&local.Binds[i].VarFodder)
			}
		}
	}
	c.Base.Local(p, local, ctx)
}

func shouldExpandSpec(spec ast.ForSpec) bool {
	shouldExpand := false
	if spec.Outer != nil {
		shouldExpand = shouldExpandSpec(*spec.Outer)
	}
	if ast.FodderCountNewlines(spec.ForFodder) > 0 {
		shouldExpand = true
	}
	for _, ifSpec := range spec.Conditions {
		if ast.FodderCountNewlines(ifSpec.IfFodder) > 0 {
			shouldExpand = true
		}
	}
	return shouldExpand
}

func ensureSpecExpanded(spec *ast.ForSpec) {
	if spec.Outer != nil {
		ensureSpecExpanded(spec.Outer)
	}
	ast.FodderEnsureCleanNewline(&spec.ForFodder)
	for i := range spec.Conditions {
		ast.FodderEnsureCleanNewline(&spec.Conditions[i].IfFodder)
	}
}

// ArrayComp handles this type of node
func (c *FixNewlines) ArrayComp(p pass.ASTPass, arrayComp *ast.ArrayComp, ctx pass.Context) {
	shouldExpand := false
	if ast.FodderCountNewlines(*openFodder(arrayComp.Body)) > 0 {
		shouldExpand = true
	}
	if shouldExpandSpec(arrayComp.Spec) {
		shouldExpand = true
	}
	if ast.FodderCountNewlines(arrayComp.CloseFodder) > 0 {
		shouldExpand = true
	}
	if shouldExpand {
		ast.FodderEnsureCleanNewline(openFodder(arrayComp.Body))
		ensureSpecExpanded(&arrayComp.Spec)
		ast.FodderEnsureCleanNewline(&arrayComp.CloseFodder)
	}
	c.Base.ArrayComp(p, arrayComp, ctx)
}

// ObjectComp handles this type of node
func (c *FixNewlines) ObjectComp(p pass.ASTPass, objectComp *ast.ObjectComp, ctx pass.Context) {
	shouldExpand := false
	for _, field := range objectComp.Fields {
		if ast.FodderCountNewlines(*objectFieldOpenFodder(&field)) > 0 {
			shouldExpand = true
		}
	}
	if shouldExpandSpec(objectComp.Spec) {
		shouldExpand = true
	}
	if ast.FodderCountNewlines(objectComp.CloseFodder) > 0 {
		shouldExpand = true
	}
	if shouldExpand {
		for i := range objectComp.Fields {
			ast.FodderEnsureCleanNewline(
				objectFieldOpenFodder(&objectComp.Fields[i]))
		}
		ensureSpecExpanded(&objectComp.Spec)
		ast.FodderEnsureCleanNewline(&objectComp.CloseFodder)
	}
	c.Base.ObjectComp(p, objectComp, ctx)
}

// Parens handles this type of node
func (c *FixNewlines) Parens(p pass.ASTPass, parens *ast.Parens, ctx pass.Context) {
	shouldExpand := false
	if ast.FodderCountNewlines(*openFodder(parens.Inner)) > 0 {
		shouldExpand = true
	}
	if ast.FodderCountNewlines(parens.CloseFodder) > 0 {
		shouldExpand = true
	}
	if shouldExpand {
		ast.FodderEnsureCleanNewline(openFodder(parens.Inner))
		ast.FodderEnsureCleanNewline(&parens.CloseFodder)
	}
	c.Base.Parens(p, parens, ctx)
}

// Parameters handles parameters
// Example2:
//   f(1, 2,
//     3)
// Should be expanded to:
//   f(1,
//     2,
//     3)
// And:
//   foo(
//       1, 2, 3)
// Should be expanded to:
//   foo(
//       1, 2, 3
//   )
func (c *FixNewlines) Parameters(p pass.ASTPass, l *ast.Fodder, params *[]ast.Parameter, r *ast.Fodder, ctx pass.Context) {
	shouldExpandBetween := false
	shouldExpandNearParens := false
	first := true
	for _, param := range *params {
		if ast.FodderCountNewlines(param.NameFodder) > 0 {
			if first {
				shouldExpandNearParens = true
			} else {
				shouldExpandBetween = true
			}
		}
		first = false
	}
	if ast.FodderCountNewlines(*r) > 0 {
		shouldExpandNearParens = true
	}
	first = true
	for i := range *params {
		param := &(*params)[i]
		if first && shouldExpandNearParens || !first && shouldExpandBetween {
			ast.FodderEnsureCleanNewline(&param.NameFodder)
		}
		first = false
	}
	if shouldExpandNearParens {
		ast.FodderEnsureCleanNewline(r)
	}
	c.Base.Parameters(p, l, params, r, ctx)
}

// Arguments handles parameters
// Example2:
//   f(1, 2,
//     3)
// Should be expanded to:
//   f(1,
//     2,
//     3)
// And:
//   foo(
//       1, 2, 3)
// Should be expanded to:
//   foo(
//       1, 2, 3
//   )
func (c *FixNewlines) Arguments(p pass.ASTPass, l *ast.Fodder, args *ast.Arguments, r *ast.Fodder, ctx pass.Context) {
	shouldExpandBetween := false
	shouldExpandNearParens := false
	first := true
	for _, arg := range args.Positional {
		if ast.FodderCountNewlines(*openFodder(arg.Expr)) > 0 {
			if first {
				shouldExpandNearParens = true
			} else {
				shouldExpandBetween = true
			}
		}
		first = false
	}
	for _, arg := range args.Named {
		if ast.FodderCountNewlines(arg.NameFodder) > 0 {
			if first {
				shouldExpandNearParens = true
			} else {
				shouldExpandBetween = true
			}
		}
		first = false
	}
	if ast.FodderCountNewlines(*r) > 0 {
		shouldExpandNearParens = true
	}
	first = true
	for i := range args.Positional {
		arg := &args.Positional[i]
		if first && shouldExpandNearParens || !first && shouldExpandBetween {
			ast.FodderEnsureCleanNewline(openFodder(arg.Expr))
		}
		first = false
	}
	for i := range args.Named {
		arg := &args.Named[i]
		if first && shouldExpandNearParens || !first && shouldExpandBetween {
			ast.FodderEnsureCleanNewline(&arg.NameFodder)
		}
		first = false
	}
	if shouldExpandNearParens {
		ast.FodderEnsureCleanNewline(r)
	}
	c.Base.Arguments(p, l, args, r, ctx)
}
