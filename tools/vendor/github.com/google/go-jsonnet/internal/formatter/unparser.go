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
	"bytes"
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

type unparser struct {
	buf     bytes.Buffer
	options Options
}

func (u *unparser) write(str string) {
	u.buf.WriteString(str)
}

// fill Pretty-prints fodder.
// The crowded and separateToken params control whether single whitespace
// characters are added to keep tokens from joining together in the output.
// The intuition of crowded is that the caller passes true for crowded if the
// last thing printed would crowd whatever we're printing here.  For example, if
// we just printed a ',' then crowded would be true.  If we just printed a '('
// then crowded would be false because we don't want the space after the '('.
//
// If crowded is true, a space is printed after any fodder, unless
// separateToken is false or the fodder ended with a newline.
// If crowded is true and separateToken is false and the fodder begins with
// an interstitial, then the interstitial is prefixed with a single space, but
// there is no space after the interstitial.
// If crowded is false and separateToken is true then a space character
// is only printed when the fodder ended with an interstitial comment (which
// creates a crowded situation where there was not one before).
// If crowded is false and separateToken is false then no space is printed
// after or before the fodder, even if the last fodder was an interstitial.
func (u *unparser) fodderFill(fodder ast.Fodder, crowded bool, separateToken bool, final bool) {
	var lastIndent int
	for i, fod := range fodder {
		skipTrailing := final && (i == (len(fodder) - 1))
		switch fod.Kind {
		case ast.FodderParagraph:
			for i, l := range fod.Comment {
				// Do not indent empty lines (note: first line is never empty).
				if len(l) > 0 {
					// First line is already indented by previous fod.
					if i > 0 {
						for i := 0; i < lastIndent; i++ {
							u.write(" ")
						}
					}
					u.write(l)
				}
				u.write("\n")
			}
			if !skipTrailing {
				for i := 0; i < fod.Blanks; i++ {
					u.write("\n")
				}
				for i := 0; i < fod.Indent; i++ {
					u.write(" ")
				}
			}
			lastIndent = fod.Indent
			crowded = false

		case ast.FodderLineEnd:
			if len(fod.Comment) > 0 {
				u.write("  ")
				u.write(fod.Comment[0])
			}
			u.write("\n")
			if !skipTrailing {
				for i := 0; i < fod.Blanks; i++ {
					u.write("\n")
				}
				for i := 0; i < fod.Indent; i++ {
					u.write(" ")
				}
			}
			lastIndent = fod.Indent
			crowded = false

		case ast.FodderInterstitial:
			if crowded {
				u.write(" ")
			}
			u.write(fod.Comment[0])
			crowded = true
		}
	}
	if separateToken && crowded {
		u.write(" ")
	}
}

func (u *unparser) fill(fodder ast.Fodder, crowded bool, separateToken bool) {
	u.fodderFill(fodder, crowded, separateToken, false)
}

func (u *unparser) fillFinal(fodder ast.Fodder, crowded bool, separateToken bool) {
	u.fodderFill(fodder, crowded, separateToken, true)
}

func (u *unparser) unparseSpecs(spec *ast.ForSpec) {
	if spec.Outer != nil {
		u.unparseSpecs(spec.Outer)
	}
	u.fill(spec.ForFodder, true, true)
	u.write("for")
	u.fill(spec.VarFodder, true, true)
	u.write(string(spec.VarName))
	u.fill(spec.InFodder, true, true)
	u.write("in")
	u.unparse(spec.Expr, true)
	for _, cond := range spec.Conditions {
		u.fill(cond.IfFodder, true, true)
		u.write("if")
		u.unparse(cond.Expr, true)
	}
}

func (u *unparser) unparseParams(fodderL ast.Fodder, params []ast.Parameter, trailingComma bool, fodderR ast.Fodder) {
	u.fill(fodderL, false, false)
	u.write("(")
	first := true
	for _, param := range params {
		if !first {
			u.write(",")
		}
		u.fill(param.NameFodder, !first, true)
		u.unparseID(param.Name)
		if param.DefaultArg != nil {
			u.fill(param.EqFodder, false, false)
			u.write("=")
			u.unparse(param.DefaultArg, false)
		}
		u.fill(param.CommaFodder, false, false)
		first = false
	}
	if trailingComma {
		u.write(",")
	}
	u.fill(fodderR, false, false)
	u.write(")")
}

func (u *unparser) unparseFieldParams(field ast.ObjectField) {
	m := field.Method
	if m != nil {
		u.unparseParams(m.ParenLeftFodder, m.Parameters, m.TrailingComma,
			m.ParenRightFodder)
	}
}

func (u *unparser) unparseFields(fields ast.ObjectFields, crowded bool) {
	first := true
	for _, field := range fields {
		if !first {
			u.write(",")
		}

		// An aux function so we don't repeat ourselves for the 3 kinds of
		// basic field.
		unparseFieldRemainder := func(field ast.ObjectField) {
			u.unparseFieldParams(field)
			u.fill(field.OpFodder, false, false)
			if field.SuperSugar {
				u.write("+")
			}
			switch field.Hide {
			case ast.ObjectFieldInherit:
				u.write(":")
			case ast.ObjectFieldHidden:
				u.write("::")
			case ast.ObjectFieldVisible:
				u.write(":::")
			}
			u.unparse(field.Expr2, true)
		}

		switch field.Kind {
		case ast.ObjectLocal:
			u.fill(field.Fodder1, !first || crowded, true)
			u.write("local")
			u.fill(field.Fodder2, true, true)
			u.unparseID(*field.Id)
			u.unparseFieldParams(field)
			u.fill(field.OpFodder, true, true)
			u.write("=")
			u.unparse(field.Expr2, true)

		case ast.ObjectFieldID:
			u.fill(field.Fodder1, !first || crowded, true)
			u.unparseID(*field.Id)
			unparseFieldRemainder(field)

		case ast.ObjectFieldStr:
			u.unparse(field.Expr1, !first || crowded)
			unparseFieldRemainder(field)

		case ast.ObjectFieldExpr:
			u.fill(field.Fodder1, !first || crowded, true)
			u.write("[")
			u.unparse(field.Expr1, false)
			u.fill(field.Fodder2, false, false)
			u.write("]")
			unparseFieldRemainder(field)

		case ast.ObjectAssert:
			u.fill(field.Fodder1, !first || crowded, true)
			u.write("assert")
			u.unparse(field.Expr2, true)
			if field.Expr3 != nil {
				u.fill(field.OpFodder, true, true)
				u.write(":")
				u.unparse(field.Expr3, true)
			}
		}

		first = false
		u.fill(field.CommaFodder, false, false)
	}

}

func (u *unparser) unparseID(id ast.Identifier) {
	u.write(string(id))
}

func (u *unparser) unparse(expr ast.Node, crowded bool) {

	if leftRecursive(expr) == nil {
		u.fill(*expr.OpenFodder(), crowded, true)
	}

	switch node := expr.(type) {
	case *ast.Apply:
		u.unparse(node.Target, crowded)
		u.fill(node.FodderLeft, false, false)
		u.write("(")
		first := true
		for _, arg := range node.Arguments.Positional {
			if !first {
				u.write(",")
			}
			space := !first
			u.unparse(arg.Expr, space)
			u.fill(arg.CommaFodder, false, false)
			first = false
		}
		for _, arg := range node.Arguments.Named {
			if !first {
				u.write(",")
			}
			space := !first
			u.fill(arg.NameFodder, space, true)
			u.unparseID(arg.Name)
			space = false
			u.write("=")
			u.unparse(arg.Arg, space)
			u.fill(arg.CommaFodder, false, false)
			first = false
		}
		if node.TrailingComma {
			u.write(",")
		}
		u.fill(node.FodderRight, false, false)
		u.write(")")
		if node.TailStrict {
			u.fill(node.TailStrictFodder, true, true)
			u.write("tailstrict")
		}

	case *ast.ApplyBrace:
		u.unparse(node.Left, crowded)
		u.unparse(node.Right, true)

	case *ast.Array:
		u.write("[")
		first := true
		for _, element := range node.Elements {
			if !first {
				u.write(",")
			}
			u.unparse(element.Expr, !first || u.options.PadArrays)
			u.fill(element.CommaFodder, false, false)
			first = false
		}
		if node.TrailingComma {
			u.write(",")
		}
		u.fill(node.CloseFodder, len(node.Elements) > 0, u.options.PadArrays)
		u.write("]")

	case *ast.ArrayComp:
		u.write("[")
		u.unparse(node.Body, u.options.PadArrays)
		u.fill(node.TrailingCommaFodder, false, false)
		if node.TrailingComma {
			u.write(",")
		}
		u.unparseSpecs(&node.Spec)
		u.fill(node.CloseFodder, true, u.options.PadArrays)
		u.write("]")

	case *ast.Assert:
		u.write("assert")
		u.unparse(node.Cond, true)
		if node.Message != nil {
			u.fill(node.ColonFodder, true, true)
			u.write(":")
			u.unparse(node.Message, true)
		}
		u.fill(node.SemicolonFodder, false, false)
		u.write(";")
		u.unparse(node.Rest, true)

	case *ast.Binary:
		u.unparse(node.Left, crowded)
		u.fill(node.OpFodder, true, true)
		u.write(node.Op.String())
		u.unparse(node.Right, true)

	case *ast.Conditional:
		u.write("if")
		u.unparse(node.Cond, true)
		u.fill(node.ThenFodder, true, true)
		u.write("then")
		u.unparse(node.BranchTrue, true)
		if node.BranchFalse != nil {
			u.fill(node.ElseFodder, true, true)
			u.write("else")
			u.unparse(node.BranchFalse, true)
		}

	case *ast.Dollar:
		u.write("$")

	case *ast.Error:
		u.write("error")
		u.unparse(node.Expr, true)

	case *ast.Function:
		u.write("function")
		u.unparseParams(node.ParenLeftFodder, node.Parameters, node.TrailingComma, node.ParenRightFodder)
		u.unparse(node.Body, true)

	case *ast.Import:
		u.write("import")
		u.unparse(node.File, true)

	case *ast.ImportStr:
		u.write("importstr")
		u.unparse(node.File, true)

	case *ast.ImportBin:
		u.write("importbin")
		u.unparse(node.File, true)

	case *ast.Index:
		u.unparse(node.Target, crowded)
		u.fill(node.LeftBracketFodder, false, false) // Can also be DotFodder
		if node.Id != nil {
			u.write(".")
			u.fill(node.RightBracketFodder, false, false) // IdFodder
			u.unparseID(*node.Id)
		} else {
			u.write("[")
			u.unparse(node.Index, false)
			u.fill(node.RightBracketFodder, false, false)
			u.write("]")
		}

	case *ast.Slice:
		u.unparse(node.Target, crowded)
		u.fill(node.LeftBracketFodder, false, false)
		u.write("[")
		if node.BeginIndex != nil {
			u.unparse(node.BeginIndex, false)
		}
		u.fill(node.EndColonFodder, false, false)
		u.write(":")
		if node.EndIndex != nil {
			u.unparse(node.EndIndex, false)
		}
		if node.Step != nil || len(node.StepColonFodder) > 0 {
			u.fill(node.StepColonFodder, false, false)
			u.write(":")
			if node.Step != nil {
				u.unparse(node.Step, false)
			}
		}
		u.fill(node.RightBracketFodder, false, false)
		u.write("]")

	case *ast.InSuper:
		u.unparse(node.Index, true)
		u.fill(node.InFodder, true, true)
		u.write("in")
		u.fill(node.SuperFodder, true, true)
		u.write("super")

	case *ast.Local:
		u.write("local")
		if len(node.Binds) == 0 {
			panic("INTERNAL ERROR: local with no binds")
		}
		first := true
		for _, bind := range node.Binds {
			if !first {
				u.write(",")
			}
			first = false
			u.fill(bind.VarFodder, true, true)
			u.unparseID(bind.Variable)
			if bind.Fun != nil {
				u.unparseParams(bind.Fun.ParenLeftFodder,
					bind.Fun.Parameters,
					bind.Fun.TrailingComma,
					bind.Fun.ParenRightFodder)
			}
			u.fill(bind.EqFodder, true, true)
			u.write("=")
			u.unparse(bind.Body, true)
			u.fill(bind.CloseFodder, false, false)
		}
		u.write(";")
		u.unparse(node.Body, true)

	case *ast.LiteralBoolean:
		if node.Value {
			u.write("true")
		} else {
			u.write("false")
		}

	case *ast.LiteralNumber:
		u.write(node.OriginalString)

	case *ast.LiteralString:
		switch node.Kind {
		case ast.StringDouble:
			u.write("\"")
			// The original escape codes are still in the string.
			u.write(node.Value)
			u.write("\"")
		case ast.StringSingle:
			u.write("'")
			// The original escape codes are still in the string.
			u.write(node.Value)
			u.write("'")
		case ast.StringBlock:
			u.write("|||")
			if node.Value[len(node.Value)-1] != '\n' {
				u.write("-")
			}
			u.write("\n")
			if node.Value[0] != '\n' {
				u.write(node.BlockIndent)
			}
			for i, r := range node.Value {
				// Formatter always outputs in unix mode.
				if r == '\r' {
					continue
				}
				u.write(string(r))
				if r == '\n' && (i+1 < len(node.Value)) && node.Value[i+1] != '\n' {
					u.write(node.BlockIndent)
				}
			}
			if node.Value[len(node.Value)-1] != '\n' {
				u.write("\n")
			}
			u.write(node.BlockTermIndent)
			u.write("|||")
		case ast.VerbatimStringDouble:
			u.write("@\"")
			// Escapes were processed by the parser, so put them back in.
			for _, r := range node.Value {
				if r == '"' {
					u.write("\"\"")
				} else {
					u.write(string(r))
				}
			}
			u.write("\"")
		case ast.VerbatimStringSingle:
			u.write("@'")
			// Escapes were processed by the parser, so put them back in.
			for _, r := range node.Value {
				if r == '\'' {
					u.write("''")
				} else {
					u.write(string(r))
				}
			}
			u.write("'")
		}

	case *ast.LiteralNull:
		u.write("null")

	case *ast.Object:
		u.write("{")
		u.unparseFields(node.Fields, u.options.PadObjects)
		if node.TrailingComma {
			u.write(",")
		}
		u.fill(node.CloseFodder, len(node.Fields) > 0, u.options.PadObjects)
		u.write("}")

	case *ast.ObjectComp:
		u.write("{")
		u.unparseFields(node.Fields, u.options.PadObjects)
		if node.TrailingComma {
			u.write(",")
		}
		u.unparseSpecs(&node.Spec)
		u.fill(node.CloseFodder, true, u.options.PadObjects)
		u.write("}")

	case *ast.Parens:
		u.write("(")
		u.unparse(node.Inner, false)
		u.fill(node.CloseFodder, false, false)
		u.write(")")

	case *ast.Self:
		u.write("self")

	case *ast.SuperIndex:
		u.write("super")
		u.fill(node.DotFodder, false, false)
		if node.Id != nil {
			u.write(".")
			u.fill(node.IDFodder, false, false)
			u.unparseID(*node.Id)
		} else {
			u.write("[")
			u.unparse(node.Index, false)
			u.fill(node.IDFodder, false, false)
			u.write("]")
		}
	case *ast.Var:
		u.unparseID(node.Id)

	case *ast.Unary:
		u.write(node.Op.String())
		u.unparse(node.Expr, false)

	default:
		panic(fmt.Sprintf("INTERNAL ERROR: Unknown AST: %T", expr))
	}
}

func (u *unparser) string() string {
	return u.buf.String()
}
