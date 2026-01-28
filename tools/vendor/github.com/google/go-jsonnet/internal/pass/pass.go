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

// Package pass provides a visitor framework for source code analysis
// and transformation.
package pass

import (
	"github.com/google/go-jsonnet/ast"
)

// Context can be used to provide context when visting child expressions.
type Context interface{}

// ASTPass is an interface for a pass that transforms the AST in some way.
type ASTPass interface {
	FodderElement(ASTPass, *ast.FodderElement, Context)
	Fodder(ASTPass, *ast.Fodder, Context)
	ForSpec(ASTPass, *ast.ForSpec, Context)
	Parameters(ASTPass, *ast.Fodder, *[]ast.Parameter, *ast.Fodder, Context)
	Arguments(ASTPass, *ast.Fodder, *ast.Arguments, *ast.Fodder, Context)
	FieldParams(ASTPass, *ast.ObjectField, Context)
	ObjectField(ASTPass, *ast.ObjectField, Context)
	ObjectFields(ASTPass, *ast.ObjectFields, Context)

	Apply(ASTPass, *ast.Apply, Context)
	ApplyBrace(ASTPass, *ast.ApplyBrace, Context)
	Array(ASTPass, *ast.Array, Context)
	ArrayComp(ASTPass, *ast.ArrayComp, Context)
	Assert(ASTPass, *ast.Assert, Context)
	Binary(ASTPass, *ast.Binary, Context)
	Conditional(ASTPass, *ast.Conditional, Context)
	Dollar(ASTPass, *ast.Dollar, Context)
	Error(ASTPass, *ast.Error, Context)
	Function(ASTPass, *ast.Function, Context)
	Import(ASTPass, *ast.Import, Context)
	ImportStr(ASTPass, *ast.ImportStr, Context)
	ImportBin(ASTPass, *ast.ImportBin, Context)
	Index(ASTPass, *ast.Index, Context)
	Slice(ASTPass, *ast.Slice, Context)
	Local(ASTPass, *ast.Local, Context)
	LiteralBoolean(ASTPass, *ast.LiteralBoolean, Context)
	LiteralNull(ASTPass, *ast.LiteralNull, Context)
	LiteralNumber(ASTPass, *ast.LiteralNumber, Context)
	LiteralString(ASTPass, *ast.LiteralString, Context)
	Object(ASTPass, *ast.Object, Context)
	ObjectComp(ASTPass, *ast.ObjectComp, Context)
	Parens(ASTPass, *ast.Parens, Context)
	Self(ASTPass, *ast.Self, Context)
	SuperIndex(ASTPass, *ast.SuperIndex, Context)
	InSuper(ASTPass, *ast.InSuper, Context)
	Unary(ASTPass, *ast.Unary, Context)
	Var(ASTPass, *ast.Var, Context)

	Visit(ASTPass, *ast.Node, Context)
	BaseContext(ASTPass) Context
	File(ASTPass, *ast.Node, *ast.Fodder)
}

// Base implements basic traversal so other passes can extend it.
type Base struct {
}

// FodderElement cannot descend any further
func (*Base) FodderElement(p ASTPass, element *ast.FodderElement, ctx Context) {
}

// Fodder traverses fodder
func (*Base) Fodder(p ASTPass, fodder *ast.Fodder, ctx Context) {
	for i := range *fodder {
		p.FodderElement(p, &(*fodder)[i], ctx)
	}
}

// ForSpec traverses a ForSpec
func (*Base) ForSpec(p ASTPass, forSpec *ast.ForSpec, ctx Context) {
	if forSpec.Outer != nil {
		p.ForSpec(p, forSpec.Outer, ctx)
	}
	p.Fodder(p, &forSpec.ForFodder, ctx)
	p.Fodder(p, &forSpec.VarFodder, ctx)
	p.Fodder(p, &forSpec.InFodder, ctx)
	p.Visit(p, &forSpec.Expr, ctx)
	for i := range forSpec.Conditions {
		cond := &forSpec.Conditions[i]
		p.Fodder(p, &cond.IfFodder, ctx)
		p.Visit(p, &cond.Expr, ctx)
	}
}

// Parameters traverses the list of parameters
func (*Base) Parameters(p ASTPass, l *ast.Fodder, params *[]ast.Parameter, r *ast.Fodder, ctx Context) {
	p.Fodder(p, l, ctx)
	for i := range *params {
		param := &(*params)[i]
		p.Fodder(p, &param.NameFodder, ctx)
		if param.DefaultArg != nil {
			p.Fodder(p, &param.EqFodder, ctx)
			p.Visit(p, &param.DefaultArg, ctx)
		}
		p.Fodder(p, &param.CommaFodder, ctx)
	}
	p.Fodder(p, r, ctx)
}

// Arguments traverses the list of arguments
func (*Base) Arguments(p ASTPass, l *ast.Fodder, args *ast.Arguments, r *ast.Fodder, ctx Context) {
	p.Fodder(p, l, ctx)
	for i := range args.Positional {
		arg := &args.Positional[i]
		p.Visit(p, &arg.Expr, ctx)
		p.Fodder(p, &arg.CommaFodder, ctx)
	}
	for i := range args.Named {
		arg := &args.Named[i]
		p.Fodder(p, &arg.NameFodder, ctx)
		p.Fodder(p, &arg.EqFodder, ctx)
		p.Visit(p, &arg.Arg, ctx)
		p.Fodder(p, &arg.CommaFodder, ctx)
	}
	p.Fodder(p, r, ctx)
}

// FieldParams is factored out of ObjectField
func (*Base) FieldParams(p ASTPass, field *ast.ObjectField, ctx Context) {
	if field.Method != nil {
		p.Parameters(
			p,
			&field.Method.ParenLeftFodder,
			&field.Method.Parameters,
			&field.Method.ParenRightFodder,
			ctx)
	}
}

// ObjectField traverses a single field
func (*Base) ObjectField(p ASTPass, field *ast.ObjectField, ctx Context) {
	switch field.Kind {
	case ast.ObjectLocal:
		p.Fodder(p, &field.Fodder1, ctx)
		p.Fodder(p, &field.Fodder2, ctx)
		p.FieldParams(p, field, ctx)
		p.Fodder(p, &field.OpFodder, ctx)
		p.Visit(p, &field.Expr2, ctx)

	case ast.ObjectFieldID:
		p.Fodder(p, &field.Fodder1, ctx)
		p.FieldParams(p, field, ctx)
		p.Fodder(p, &field.OpFodder, ctx)
		p.Visit(p, &field.Expr2, ctx)

	case ast.ObjectFieldStr:
		p.Visit(p, &field.Expr1, ctx)
		p.FieldParams(p, field, ctx)
		p.Fodder(p, &field.OpFodder, ctx)
		p.Visit(p, &field.Expr2, ctx)

	case ast.ObjectFieldExpr:
		p.Fodder(p, &field.Fodder1, ctx)
		p.Visit(p, &field.Expr1, ctx)
		p.Fodder(p, &field.Fodder2, ctx)
		p.FieldParams(p, field, ctx)
		p.Fodder(p, &field.OpFodder, ctx)
		p.Visit(p, &field.Expr2, ctx)

	case ast.ObjectAssert:
		p.Fodder(p, &field.Fodder1, ctx)
		p.Visit(p, &field.Expr2, ctx)
		if field.Expr3 != nil {
			p.Fodder(p, &field.OpFodder, ctx)
			p.Visit(p, &field.Expr3, ctx)
		}
	}

	p.Fodder(p, &field.CommaFodder, ctx)
}

// ObjectFields traverses object fields
func (*Base) ObjectFields(p ASTPass, fields *ast.ObjectFields, ctx Context) {
	for i := range *fields {
		p.ObjectField(p, &(*fields)[i], ctx)
	}
}

// Apply traverses that kind of node
func (*Base) Apply(p ASTPass, node *ast.Apply, ctx Context) {
	p.Visit(p, &node.Target, ctx)
	p.Arguments(p, &node.FodderLeft, &node.Arguments, &node.FodderRight, ctx)
	if node.TailStrict {
		p.Fodder(p, &node.TailStrictFodder, ctx)
	}
}

// ApplyBrace traverses that kind of node
func (*Base) ApplyBrace(p ASTPass, node *ast.ApplyBrace, ctx Context) {
	p.Visit(p, &node.Left, ctx)
	p.Visit(p, &node.Right, ctx)
}

// Array traverses that kind of node
func (*Base) Array(p ASTPass, node *ast.Array, ctx Context) {
	for i := range node.Elements {
		p.Visit(p, &node.Elements[i].Expr, ctx)
		p.Fodder(p, &node.Elements[i].CommaFodder, ctx)
	}
	p.Fodder(p, &node.CloseFodder, ctx)
}

// ArrayComp traverses that kind of node
func (*Base) ArrayComp(p ASTPass, node *ast.ArrayComp, ctx Context) {
	p.Visit(p, &node.Body, ctx)
	p.Fodder(p, &node.TrailingCommaFodder, ctx)
	p.ForSpec(p, &node.Spec, ctx)
	p.Fodder(p, &node.CloseFodder, ctx)
}

// Assert traverses that kind of node
func (*Base) Assert(p ASTPass, node *ast.Assert, ctx Context) {
	p.Visit(p, &node.Cond, ctx)
	if node.Message != nil {
		p.Fodder(p, &node.ColonFodder, ctx)
		p.Visit(p, &node.Message, ctx)
	}
	p.Fodder(p, &node.SemicolonFodder, ctx)
	p.Visit(p, &node.Rest, ctx)
}

// Binary traverses that kind of node
func (*Base) Binary(p ASTPass, node *ast.Binary, ctx Context) {
	p.Visit(p, &node.Left, ctx)
	p.Fodder(p, &node.OpFodder, ctx)
	p.Visit(p, &node.Right, ctx)
}

// Conditional traverses that kind of node
func (*Base) Conditional(p ASTPass, node *ast.Conditional, ctx Context) {
	p.Visit(p, &node.Cond, ctx)
	p.Fodder(p, &node.ThenFodder, ctx)
	p.Visit(p, &node.BranchTrue, ctx)
	if node.BranchFalse != nil {
		p.Fodder(p, &node.ElseFodder, ctx)
		p.Visit(p, &node.BranchFalse, ctx)
	}
}

// Dollar cannot descend any further
func (*Base) Dollar(p ASTPass, node *ast.Dollar, ctx Context) {
}

// Error traverses that kind of node
func (*Base) Error(p ASTPass, node *ast.Error, ctx Context) {
	p.Visit(p, &node.Expr, ctx)
}

// Function traverses that kind of node
func (*Base) Function(p ASTPass, node *ast.Function, ctx Context) {
	p.Parameters(p, &node.ParenLeftFodder, &node.Parameters, &node.ParenRightFodder, ctx)
	p.Visit(p, &node.Body, ctx)
}

// Import traverses that kind of node
func (*Base) Import(p ASTPass, node *ast.Import, ctx Context) {
	p.Fodder(p, &node.File.Fodder, ctx)
	p.LiteralString(p, node.File, ctx)
}

// ImportStr traverses that kind of node
func (*Base) ImportStr(p ASTPass, node *ast.ImportStr, ctx Context) {
	p.Fodder(p, &node.File.Fodder, ctx)
	p.LiteralString(p, node.File, ctx)
}

// ImportBin traverses that kind of node
func (*Base) ImportBin(p ASTPass, node *ast.ImportBin, ctx Context) {
	p.Fodder(p, &node.File.Fodder, ctx)
	p.LiteralString(p, node.File, ctx)
}

// Index traverses that kind of node
func (*Base) Index(p ASTPass, node *ast.Index, ctx Context) {
	p.Visit(p, &node.Target, ctx)
	p.Fodder(p, &node.LeftBracketFodder, ctx)
	if node.Id == nil {
		p.Visit(p, &node.Index, ctx)
		p.Fodder(p, &node.RightBracketFodder, ctx)
	}
}

// InSuper traverses that kind of node
func (*Base) InSuper(p ASTPass, node *ast.InSuper, ctx Context) {
	p.Visit(p, &node.Index, ctx)
}

// LiteralBoolean cannot descend any further
func (*Base) LiteralBoolean(p ASTPass, node *ast.LiteralBoolean, ctx Context) {
}

// LiteralNull cannot descend any further
func (*Base) LiteralNull(p ASTPass, node *ast.LiteralNull, ctx Context) {
}

// LiteralNumber cannot descend any further
func (*Base) LiteralNumber(p ASTPass, node *ast.LiteralNumber, ctx Context) {
}

// LiteralString cannot descend any further
func (*Base) LiteralString(p ASTPass, node *ast.LiteralString, ctx Context) {
}

// Local traverses that kind of node
func (*Base) Local(p ASTPass, node *ast.Local, ctx Context) {
	for i := range node.Binds {
		bind := &node.Binds[i]
		p.Fodder(p, &bind.VarFodder, ctx)
		if bind.Fun != nil {
			p.Parameters(p, &bind.Fun.ParenLeftFodder, &bind.Fun.Parameters, &bind.Fun.ParenRightFodder, ctx)
		}
		p.Fodder(p, &bind.EqFodder, ctx)
		p.Visit(p, &bind.Body, ctx)
		p.Fodder(p, &bind.CloseFodder, ctx)
	}
	p.Visit(p, &node.Body, ctx)
}

// Object traverses that kind of node
func (*Base) Object(p ASTPass, node *ast.Object, ctx Context) {
	p.ObjectFields(p, &node.Fields, ctx)
	p.Fodder(p, &node.CloseFodder, ctx)
}

// ObjectComp traverses that kind of node
func (*Base) ObjectComp(p ASTPass, node *ast.ObjectComp, ctx Context) {
	p.ObjectFields(p, &node.Fields, ctx)
	p.ForSpec(p, &node.Spec, ctx)
	p.Fodder(p, &node.CloseFodder, ctx)
}

// Parens traverses that kind of node
func (*Base) Parens(p ASTPass, node *ast.Parens, ctx Context) {
	p.Visit(p, &node.Inner, ctx)
	p.Fodder(p, &node.CloseFodder, ctx)
}

// Self cannot descend any further
func (*Base) Self(p ASTPass, node *ast.Self, ctx Context) {
}

// Slice traverses that kind of node
func (*Base) Slice(p ASTPass, node *ast.Slice, ctx Context) {
	p.Visit(p, &node.Target, ctx)
	p.Fodder(p, &node.LeftBracketFodder, ctx)
	if node.BeginIndex != nil {
		p.Visit(p, &node.BeginIndex, ctx)
	}
	p.Fodder(p, &node.EndColonFodder, ctx)
	if node.EndIndex != nil {
		p.Visit(p, &node.EndIndex, ctx)
	}
	p.Fodder(p, &node.StepColonFodder, ctx)
	if node.Step != nil {
		p.Visit(p, &node.Step, ctx)
	}
	p.Fodder(p, &node.RightBracketFodder, ctx)
}

// SuperIndex traverses that kind of node
func (*Base) SuperIndex(p ASTPass, node *ast.SuperIndex, ctx Context) {
	p.Fodder(p, &node.DotFodder, ctx)
	if node.Id == nil {
		p.Visit(p, &node.Index, ctx)
	}
	p.Fodder(p, &node.IDFodder, ctx)
}

// Unary traverses that kind of node
func (*Base) Unary(p ASTPass, node *ast.Unary, ctx Context) {
	p.Visit(p, &node.Expr, ctx)
}

// Var cannot descend any further
func (*Base) Var(p ASTPass, node *ast.Var, ctx Context) {
}

// Visit traverses into an arbitrary node type
func (*Base) Visit(p ASTPass, node *ast.Node, ctx Context) {

	f := *(*node).OpenFodder()
	p.Fodder(p, &f, ctx)
	*(*node).OpenFodder() = f

	switch node := (*node).(type) {
	case *ast.Apply:
		p.Apply(p, node, ctx)
	case *ast.ApplyBrace:
		p.ApplyBrace(p, node, ctx)
	case *ast.Array:
		p.Array(p, node, ctx)
	case *ast.ArrayComp:
		p.ArrayComp(p, node, ctx)
	case *ast.Assert:
		p.Assert(p, node, ctx)
	case *ast.Binary:
		p.Binary(p, node, ctx)
	case *ast.Conditional:
		p.Conditional(p, node, ctx)
	case *ast.Dollar:
		p.Dollar(p, node, ctx)
	case *ast.Error:
		p.Error(p, node, ctx)
	case *ast.Function:
		p.Function(p, node, ctx)
	case *ast.Import:
		p.Import(p, node, ctx)
	case *ast.ImportStr:
		p.ImportStr(p, node, ctx)
	case *ast.ImportBin:
		p.ImportBin(p, node, ctx)
	case *ast.Index:
		p.Index(p, node, ctx)
	case *ast.InSuper:
		p.InSuper(p, node, ctx)
	case *ast.LiteralBoolean:
		p.LiteralBoolean(p, node, ctx)
	case *ast.LiteralNull:
		p.LiteralNull(p, node, ctx)
	case *ast.LiteralNumber:
		p.LiteralNumber(p, node, ctx)
	case *ast.LiteralString:
		p.LiteralString(p, node, ctx)
	case *ast.Local:
		p.Local(p, node, ctx)
	case *ast.Object:
		p.Object(p, node, ctx)
	case *ast.ObjectComp:
		p.ObjectComp(p, node, ctx)
	case *ast.Parens:
		p.Parens(p, node, ctx)
	case *ast.Self:
		p.Self(p, node, ctx)
	case *ast.Slice:
		p.Slice(p, node, ctx)
	case *ast.SuperIndex:
		p.SuperIndex(p, node, ctx)
	case *ast.Unary:
		p.Unary(p, node, ctx)
	case *ast.Var:
		p.Var(p, node, ctx)
	}
}

// BaseContext just returns nil.
func (*Base) BaseContext(ASTPass) Context {
	return nil
}

// File processes a whole Jsonnet file
func (*Base) File(p ASTPass, node *ast.Node, finalFodder *ast.Fodder) {
	ctx := p.BaseContext(p)
	p.Visit(p, node, ctx)
	p.Fodder(p, finalFodder, ctx)
}
