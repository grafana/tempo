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
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/pass"
)

// FixIndentation is a formatter pass that changes the indentation of new line
// fodder so that it follows the nested structure of the code.
type FixIndentation struct {
	pass.Base
	column  int
	Options Options
}

// indent is the representation of the indentation level.  The field lineUp is
// what is generally used to indent after a new line.  The field base is used to
// help derive a new Indent struct when the indentation level increases.  lineUp
// is generally > base.
//
// In the following case (where spaces are replaced with underscores):
// ____foobar(1,
// ___________2)
//
// At the AST representing the 2, the indent has base == 4 and lineUp == 11.
type indent struct {
	base   int
	lineUp int
}

// setIndents sets the indentation values within the fodder elements.
// The last one gets a special indentation value, all the others are set to the same thing.
func (c *FixIndentation) setIndents(
	fodder ast.Fodder, allButLastIndent int, lastIndent int) {

	// First count how many there are.
	count := 0
	for _, f := range fodder {
		if f.Kind != ast.FodderInterstitial {
			count++
		}
	}
	// Now set the indents.
	i := 0
	for index := range fodder {
		f := &fodder[index]
		if f.Kind != ast.FodderInterstitial {
			if i+1 < count {
				f.Indent = allButLastIndent
			} else {
				if i != count-1 {
					panic("Shouldn't get here")
				}
				f.Indent = lastIndent
			}
			i++
		}
	}
}

// fill sets the indentation on the fodder elements and adjusts the c.column
// counter as if it was printed.
// To understand fodder, crowded, separateToken, see the documentation of
// unparse.fill.
// allButLastIndent is the new indentation value for all but the final fodder
// element.
// lastIndent is the new indentation value for the final fodder element.
func (c *FixIndentation) fillLast(
	fodder ast.Fodder, crowded bool, separateToken bool,
	allButLastIndent int, lastIndent int) {
	c.setIndents(fodder, allButLastIndent, lastIndent)

	// A model of unparser.fill that just keeps track of the
	// c.column counter.
	for _, fod := range fodder {

		switch fod.Kind {
		case ast.FodderParagraph:
			c.column = fod.Indent
			crowded = false

		case ast.FodderLineEnd:
			c.column = fod.Indent
			crowded = false

		case ast.FodderInterstitial:
			if crowded {
				c.column++
			}
			c.column += len(fod.Comment[0])
			crowded = true
		}
	}
	if separateToken && crowded {
		c.column++
	}
}

// fill is like fillLast but where the final and prior fodder get the same
// currIndent.
func (c *FixIndentation) fill(
	fodder ast.Fodder, crowded bool, separateToken bool, indent int) {
	c.fillLast(fodder, crowded, separateToken, indent, indent)
}

// newIndent calculates the indentation of sub-expressions.
// If the first sub-expression is on the same line as the current node, then subsequent
// ones will be lined up, otherwise subsequent ones will be on the next line indented
// by 'indent'.
func (c *FixIndentation) newIndent(firstFodder ast.Fodder, old indent, lineUp int) indent {
	if len(firstFodder) == 0 || firstFodder[0].Kind == ast.FodderInterstitial {
		return indent{old.base, lineUp}
	}
	// Reset
	return indent{old.base + c.Options.Indent, old.base + c.Options.Indent}
}

// Calculate the indentation of sub-expressions.
// If the first sub-expression is on the same line as the current node, then
// subsequent ones will be lined up and further indentations in their
// subexpressions will be based from this c.column.
func (c *FixIndentation) newIndentStrong(firstFodder ast.Fodder, old indent, lineUp int) indent {
	if len(firstFodder) == 0 || firstFodder[0].Kind == ast.FodderInterstitial {
		return indent{lineUp, lineUp}
	}
	// Reset
	return indent{old.base + c.Options.Indent, old.base + c.Options.Indent}
}

// Calculate the indentation of sub-expressions.
// If the first sub-expression is on the same line as the current node, then
// subsequent ones will be lined up, otherwise subseqeuent ones will be on the
// next line with no additional currIndent.
func (c *FixIndentation) align(firstFodder ast.Fodder, old indent, lineUp int) indent {
	if len(firstFodder) == 0 || firstFodder[0].Kind == ast.FodderInterstitial {
		return indent{old.base, lineUp}
	}
	// Reset
	return old
}

// alignStrong calculates the indentation of sub-expressions.
// If the first sub-expression is on the same line as the current node, then
// subsequent ones will be lined up and further indentations in their
// subexpresssions will be based from this c.column.  Otherwise, subseqeuent ones
// will be on the next line with no additional currIndent.
func (c *FixIndentation) alignStrong(firstFodder ast.Fodder, old indent, lineUp int) indent {
	if len(firstFodder) == 0 || firstFodder[0].Kind == ast.FodderInterstitial {
		return indent{lineUp, lineUp}
	}
	// Reset
	return old
}

/** Does the given fodder contain at least one new line? */
func (c *FixIndentation) hasNewLines(fodder ast.Fodder) bool {
	for _, f := range fodder {
		if f.Kind != ast.FodderInterstitial {
			return true
		}
	}
	return false
}

// specs indents comprehension forspecs.
func (c *FixIndentation) specs(spec *ast.ForSpec, currIndent indent) {
	if spec.Outer != nil {
		c.specs(spec.Outer, currIndent)
	}
	c.fill(spec.ForFodder, true, true, currIndent.lineUp)
	c.column += 3 // for
	c.fill(spec.VarFodder, true, true, currIndent.lineUp)
	c.column += len(spec.VarName)
	c.fill(spec.InFodder, true, true, currIndent.lineUp)
	c.column += 2 // in
	newIndent := c.newIndent(*openFodder(spec.Expr), currIndent, c.column)
	c.Visit(spec.Expr, newIndent, true)
	for _, cond := range spec.Conditions {
		c.fill(cond.IfFodder, true, true, currIndent.lineUp)
		c.column += 2 // if
		newIndent := c.newIndent(*openFodder(spec.Expr), currIndent, c.column)
		c.Visit(spec.Expr, newIndent, true)
	}
}

func (c *FixIndentation) params(fodderL ast.Fodder, params []ast.Parameter,
	trailingComma bool, fodderR ast.Fodder, currIndent indent) {
	c.fill(fodderL, false, false, currIndent.lineUp)
	c.column++ // (
	var firstInside ast.Fodder
	gotFodder := false
	for _, param := range params {
		firstInside = param.NameFodder
		gotFodder = true
		break
	}
	if !gotFodder {
		firstInside = fodderR
	}
	newIndent := c.newIndent(firstInside, currIndent, c.column)
	first := true
	for _, param := range params {
		if !first {
			c.column++ // ','
		}
		c.fill(param.NameFodder, !first, true, newIndent.lineUp)
		c.column += len(param.Name)
		if param.DefaultArg != nil {
			c.fill(param.EqFodder, false, false, newIndent.lineUp)
			// default arg, no spacing: x=e
			c.column++
			c.Visit(param.DefaultArg, newIndent, false)
		}
		c.fill(param.CommaFodder, false, false, newIndent.lineUp)
		first = false
	}
	if trailingComma {
		c.column++
	}
	c.fillLast(fodderR, false, false, newIndent.lineUp, currIndent.lineUp)
	c.column++ // )
}

func (c *FixIndentation) fieldParams(field ast.ObjectField, currIndent indent) {
	m := field.Method
	if m != nil {
		c.params(m.ParenLeftFodder, m.Parameters, m.TrailingComma,
			m.ParenRightFodder, currIndent)
	}
}

// fields indents fields within an object.
// indent is the indent of the first field
// crowded is whether the first field is crowded (see unparser.fill)
func (c *FixIndentation) fields(fields ast.ObjectFields, currIndent indent, crowded bool) {
	newIndent := currIndent.lineUp
	for i, field := range fields {
		if i > 0 {
			c.column++ // ','
		}

		// An aux function so we don't repeat ourselves for the 3 kinds of
		// basic field.
		unparseFieldRemainder := func(field ast.ObjectField) {
			c.fieldParams(field, currIndent)
			c.fill(field.OpFodder, false, false, newIndent)
			if field.SuperSugar {
				c.column++
			}
			switch field.Hide {
			case ast.ObjectFieldInherit:
				c.column++
			case ast.ObjectFieldHidden:
				c.column += 2
			case ast.ObjectFieldVisible:
				c.column += 3
			}
			c.Visit(field.Expr2,
				c.newIndent(*openFodder(field.Expr2), currIndent, c.column),
				true)
		}

		switch field.Kind {
		case ast.ObjectLocal:
			c.fill(field.Fodder1, i > 0 || crowded, true, currIndent.lineUp)
			c.column += 5 // local
			c.fill(field.Fodder2, true, true, currIndent.lineUp)
			c.column += len(*field.Id)
			c.fieldParams(field, currIndent)
			c.fill(field.OpFodder, true, true, currIndent.lineUp)
			c.column++ // =
			newIndent2 := c.newIndent(*openFodder(field.Expr2), currIndent, c.column)
			c.Visit(field.Expr2, newIndent2, true)

		case ast.ObjectFieldID:
			c.fill(field.Fodder1, i > 0 || crowded, true, newIndent)
			c.column += len(*field.Id)
			unparseFieldRemainder(field)

		case ast.ObjectFieldStr:
			c.Visit(field.Expr1, currIndent, i > 0 || crowded)
			unparseFieldRemainder(field)

		case ast.ObjectFieldExpr:
			c.fill(field.Fodder1, i > 0 || crowded, true, newIndent)
			c.column++ // [
			c.Visit(field.Expr1, currIndent, false)
			c.fill(field.Fodder2, false, false, newIndent)
			c.column++ // ]
			unparseFieldRemainder(field)

		case ast.ObjectAssert:
			c.fill(field.Fodder1, i > 0 || crowded, true, newIndent)
			c.column += 6 // assert
			// + 1 for the space after the assert
			newIndent2 := c.newIndent(*openFodder(field.Expr2), currIndent, c.column+1)
			c.Visit(field.Expr2, currIndent, true)
			if field.Expr3 != nil {
				c.fill(field.OpFodder, true, true, newIndent2.lineUp)
				c.column++ // ":"
				c.Visit(field.Expr3, newIndent2, true)
			}
		}
		c.fill(field.CommaFodder, false, false, newIndent)
	}
}

// Visit has logic common to all nodes.
func (c *FixIndentation) Visit(expr ast.Node, currIndent indent, crowded bool) {
	separateToken := leftRecursive(expr) == nil
	c.fill(*expr.OpenFodder(), crowded, separateToken, currIndent.lineUp)
	switch node := expr.(type) {

	case *ast.Apply:
		initFodder := *openFodder(node.Target)
		newColumn := c.column
		if crowded {
			newColumn++
		}
		newIndent := c.align(initFodder, currIndent, newColumn)
		c.Visit(node.Target, newIndent, crowded)
		c.fill(node.FodderLeft, false, false, newIndent.lineUp)
		c.column++ // (
		firstFodder := node.FodderRight
		for _, arg := range node.Arguments.Named {
			firstFodder = arg.NameFodder
			break
		}
		for _, arg := range node.Arguments.Positional {
			firstFodder = *openFodder(arg.Expr)
			break
		}
		strongIndent := false
		// Need to use strong indent if any of the
		// arguments (except the first) are preceded by newlines.
		first := true
		for _, arg := range node.Arguments.Positional {
			if first {
				// Skip first element.
				first = false
				continue
			}
			if c.hasNewLines(*openFodder(arg.Expr)) {
				strongIndent = true
			}
		}
		for _, arg := range node.Arguments.Named {
			if first {
				// Skip first element.
				first = false
				continue
			}
			if c.hasNewLines(arg.NameFodder) {
				strongIndent = true
			}
		}
		var argIndent indent
		if strongIndent {
			argIndent = c.newIndentStrong(firstFodder, currIndent, c.column)
		} else {
			argIndent = c.newIndent(firstFodder, currIndent, c.column)
		}

		first = true
		for _, arg := range node.Arguments.Positional {
			if !first {
				c.column++ // ","
			}
			space := !first
			c.Visit(arg.Expr, argIndent, space)
			c.fill(arg.CommaFodder, false, false, argIndent.lineUp)
			first = false
		}
		for _, arg := range node.Arguments.Named {
			if !first {
				c.column++ // ","
			}
			space := !first
			c.fill(arg.NameFodder, space, false, argIndent.lineUp)
			c.column += len(arg.Name)
			c.column++ // "="
			c.Visit(arg.Arg, argIndent, false)
			c.fill(arg.CommaFodder, false, false, argIndent.lineUp)
			first = false
		}
		if node.TrailingComma {
			c.column++ // ","
		}
		c.fillLast(node.FodderRight, false, false, argIndent.lineUp, currIndent.base)
		c.column++ // )
		if node.TailStrict {
			c.fill(node.TailStrictFodder, true, true, currIndent.base)
			c.column += 10 // tailstrict
		}

	case *ast.ApplyBrace:
		initFodder := *openFodder(node.Left)
		newColumn := c.column
		if crowded {
			newColumn++
		}
		newIndent := c.align(initFodder, currIndent, newColumn)
		c.Visit(node.Left, newIndent, crowded)
		c.Visit(node.Right, newIndent, true)

	case *ast.Array:
		c.column++ // '['
		// First fodder element exists and is a newline
		var firstFodder ast.Fodder
		if len(node.Elements) > 0 {
			firstFodder = *openFodder(node.Elements[0].Expr)
		} else {
			firstFodder = node.CloseFodder
		}
		newColumn := c.column
		if c.Options.PadArrays {
			newColumn++
		}
		strongIndent := false
		// Need to use strong indent if there are not newlines before any of the sub-expressions
		for i, el := range node.Elements {
			if i == 0 {
				continue
			}
			if c.hasNewLines(*openFodder(el.Expr)) {
				strongIndent = true
			}
		}

		var newIndent indent
		if strongIndent {
			newIndent = c.newIndentStrong(firstFodder, currIndent, newColumn)
		} else {
			newIndent = c.newIndent(firstFodder, currIndent, newColumn)
		}

		for i, el := range node.Elements {
			if i > 0 {
				c.column++
			}
			c.Visit(el.Expr, newIndent, i > 0 || c.Options.PadArrays)
			c.fill(el.CommaFodder, false, false, newIndent.lineUp)
		}
		if node.TrailingComma {
			c.column++
		}

		// Handle penultimate newlines from expr.CloseFodder if there are any.
		c.fillLast(node.CloseFodder,
			len(node.Elements) > 0,
			c.Options.PadArrays,
			newIndent.lineUp,
			currIndent.base)
		c.column++ // ']'

	case *ast.ArrayComp:
		c.column++ // [
		newColumn := c.column
		if c.Options.PadArrays {
			newColumn++
		}
		newIndent :=
			c.newIndent(*openFodder(node.Body), currIndent, newColumn)
		c.Visit(node.Body, newIndent, c.Options.PadArrays)
		c.fill(node.TrailingCommaFodder, false, false, newIndent.lineUp)
		if node.TrailingComma {
			c.column++ // ','
		}
		c.specs(&node.Spec, newIndent)
		c.fillLast(node.CloseFodder, true, c.Options.PadArrays,
			newIndent.lineUp, currIndent.base)
		c.column++ // ]

	case *ast.Assert:

		c.column += 6 // assert
		// + 1 for the space after the assert
		newIndent := c.newIndent(*openFodder(node.Cond), currIndent, c.column+1)
		c.Visit(node.Cond, newIndent, true)
		if node.Message != nil {
			c.fill(node.ColonFodder, true, true, newIndent.lineUp)
			c.column++ // ":"
			c.Visit(node.Message, newIndent, true)
		}
		c.fill(node.SemicolonFodder, false, false, newIndent.lineUp)
		c.column++ // ";"
		c.Visit(node.Rest, currIndent, true)

	case *ast.Binary:
		firstFodder := *openFodder(node.Left)
		// Need to use strong indent in the case of
		/*
		   A
		   + B
		   or
		   A +
		   B
		*/

		innerColumn := c.column
		if crowded {
			innerColumn++
		}
		var newIndent indent
		if c.hasNewLines(node.OpFodder) || c.hasNewLines(*openFodder(node.Right)) {
			newIndent = c.alignStrong(firstFodder, currIndent, innerColumn)
		} else {
			newIndent = c.align(firstFodder, currIndent, innerColumn)
		}
		c.Visit(node.Left, newIndent, crowded)
		c.fill(node.OpFodder, true, true, newIndent.lineUp)
		c.column += len(node.Op.String())
		// Don't calculate a new indent for here, because we like being able to do:
		// true &&
		// true &&
		// true
		c.Visit(node.Right, newIndent, true)

	case *ast.Conditional:
		c.column += 2 // if
		condIndent := c.newIndent(*openFodder(node.Cond), currIndent, c.column+1)
		c.Visit(node.Cond, condIndent, true)
		c.fill(node.ThenFodder, true, true, currIndent.base)
		c.column += 4 // then
		trueIndent := c.newIndent(*openFodder(node.BranchTrue), currIndent, c.column+1)
		c.Visit(node.BranchTrue, trueIndent, true)
		if node.BranchFalse != nil {
			c.fill(node.ElseFodder, true, true, currIndent.base)
			c.column += 4 // else
			falseIndent := c.newIndent(*openFodder(node.BranchFalse), currIndent, c.column+1)
			c.Visit(node.BranchFalse, falseIndent, true)
		}

	case *ast.Dollar:
		c.column++ // $

	case *ast.Error:
		c.column += 5 // error
		newIndent := c.newIndent(*openFodder(node.Expr), currIndent, c.column+1)
		c.Visit(node.Expr, newIndent, true)

	case *ast.Function:
		c.column += 8 // function
		c.params(node.ParenLeftFodder, node.Parameters,
			node.TrailingComma, node.ParenRightFodder, currIndent)
		newIndent := c.newIndent(*openFodder(node.Body), currIndent, c.column+1)
		c.Visit(node.Body, newIndent, true)

	case *ast.Import:
		c.column += 6 // import
		newIndent := c.newIndent(*openFodder(node.File), currIndent, c.column+1)
		c.Visit(node.File, newIndent, true)

	case *ast.ImportStr:
		c.column += 9 // importstr
		newIndent := c.newIndent(*openFodder(node.File), currIndent, c.column+1)
		c.Visit(node.File, newIndent, true)

	case *ast.ImportBin:
		c.column += 9 // importbin
		newIndent := c.newIndent(*openFodder(node.File), currIndent, c.column+1)
		c.Visit(node.File, newIndent, true)

	case *ast.InSuper:
		c.Visit(node.Index, currIndent, crowded)
		c.fill(node.InFodder, true, true, currIndent.lineUp)
		c.column += 2 // in
		c.fill(node.SuperFodder, true, true, currIndent.lineUp)
		c.column += 5 // super

	case *ast.Index:
		c.Visit(node.Target, currIndent, crowded)
		c.fill(node.LeftBracketFodder, false, false, currIndent.lineUp) // Can also be DotFodder
		if node.Id != nil {
			c.column++ // "."
			newIndent := c.newIndent(node.RightBracketFodder, currIndent, c.column)
			c.fill(node.RightBracketFodder, false, false, newIndent.lineUp) // Can also be IdFodder
			c.column += len(*node.Id)
		} else {
			c.column++ // "["
			newIndent := c.newIndent(*openFodder(node.Index), currIndent, c.column)
			c.Visit(node.Index, newIndent, false)
			c.fillLast(node.RightBracketFodder, false, false, newIndent.lineUp, currIndent.base)
			c.column++ // "]"
		}

	case *ast.Slice:
		c.Visit(node.Target, currIndent, crowded)
		c.fill(node.LeftBracketFodder, false, false, currIndent.lineUp)
		c.column++ // "["
		var newIndent indent
		if node.BeginIndex != nil {
			newIndent = c.newIndent(*openFodder(node.BeginIndex), currIndent, c.column)
			c.Visit(node.BeginIndex, newIndent, false)
		}
		if node.EndIndex != nil {
			newIndent = c.newIndent(node.EndColonFodder, currIndent, c.column)
			c.fill(node.EndColonFodder, false, false, newIndent.lineUp)
			c.column++ // ":"
			c.Visit(node.EndIndex, newIndent, false)
		}
		if node.Step != nil {
			if node.EndIndex == nil {
				newIndent = c.newIndent(node.EndColonFodder, currIndent, c.column)
				c.fill(node.EndColonFodder, false, false, newIndent.lineUp)
				c.column++ // ":"
			}
			c.fill(node.StepColonFodder, false, false, newIndent.lineUp)
			c.column++ // ":"
			c.Visit(node.Step, newIndent, false)
		}
		if node.BeginIndex == nil && node.EndIndex == nil && node.Step == nil {
			newIndent = c.newIndent(node.EndColonFodder, currIndent, c.column)
			c.fill(node.EndColonFodder, false, false, newIndent.lineUp)
			c.column++ // ":"
		}
		c.column++ // "]"

	case *ast.Local:
		c.column += 5 // local
		if len(node.Binds) == 0 {
			panic("Not enough binds in local")
		}
		first := true
		newIndent := c.newIndent(node.Binds[0].VarFodder, currIndent, c.column+1)
		for _, bind := range node.Binds {
			if !first {
				c.column++ // ','
			}
			first = false
			c.fill(bind.VarFodder, true, true, newIndent.lineUp)
			c.column += len(bind.Variable)
			if bind.Fun != nil {
				c.params(bind.Fun.ParenLeftFodder,
					bind.Fun.Parameters,
					bind.Fun.TrailingComma,
					bind.Fun.ParenRightFodder,
					newIndent)
			}
			c.fill(bind.EqFodder, true, true, newIndent.lineUp)
			c.column++ // '='
			newIndent2 := c.newIndent(*openFodder(bind.Body), newIndent, c.column+1)
			c.Visit(bind.Body, newIndent2, true)
			c.fillLast(bind.CloseFodder, false, false, newIndent2.lineUp,
				currIndent.base)
		}
		c.column++ // ';'
		c.Visit(node.Body, currIndent, true)

	case *ast.LiteralBoolean:
		if node.Value {
			c.column += 4
		} else {
			c.column += 5
		}

	case *ast.LiteralNumber:
		c.column += len(node.OriginalString)

	case *ast.LiteralString:
		switch node.Kind {
		case ast.StringDouble:
			c.column += 2 + len(node.Value) // Include quotes
		case ast.StringSingle:
			c.column += 2 + len(node.Value) // Include quotes
		case ast.StringBlock:
			node.BlockIndent = strings.Repeat(" ", currIndent.base+c.Options.Indent)
			node.BlockTermIndent = strings.Repeat(" ", currIndent.base)
			c.column = currIndent.base // blockTermIndent
			c.column += 3              // always "|||" (never "|||-" because we're only accounting for block end)
		case ast.VerbatimStringSingle:
			c.column += 3 // Include @, start and end quotes
			for _, r := range node.Value {
				if r == '\'' {
					c.column += 2
				} else {
					c.column++
				}
			}
		case ast.VerbatimStringDouble:
			c.column += 3 // Include @, start and end quotes
			for _, r := range node.Value {
				if r == '"' {
					c.column += 2
				} else {
					c.column++
				}
			}
		}

	case *ast.LiteralNull:
		c.column += 4 // null

	case *ast.Object:
		c.column++ // '{'
		var firstFodder ast.Fodder
		if len(node.Fields) == 0 {
			firstFodder = node.CloseFodder
		} else {
			if node.Fields[0].Kind == ast.ObjectFieldStr {
				firstFodder = *openFodder(node.Fields[0].Expr1)
			} else {
				firstFodder = node.Fields[0].Fodder1
			}
		}
		newColumn := c.column
		if c.Options.PadObjects {
			newColumn++
		}
		newIndent := c.newIndent(firstFodder, currIndent, newColumn)
		c.fields(node.Fields, newIndent, c.Options.PadObjects)
		if node.TrailingComma {
			c.column++
		}
		c.fillLast(node.CloseFodder,
			len(node.Fields) > 0,
			c.Options.PadObjects,
			newIndent.lineUp,
			currIndent.base)
		c.column++ // '}'

	case *ast.ObjectComp:
		c.column++ // '{'
		var firstFodder ast.Fodder
		if len(node.Fields) == 0 {
			firstFodder = node.CloseFodder
		} else {
			if node.Fields[0].Kind == ast.ObjectFieldStr {
				firstFodder = *openFodder(node.Fields[0].Expr1)
			} else {
				firstFodder = node.Fields[0].Fodder1
			}
		}
		newColumn := c.column
		if c.Options.PadObjects {
			newColumn++
		}
		newIndent := c.newIndent(firstFodder, currIndent, newColumn)

		c.fields(node.Fields, newIndent, c.Options.PadObjects)
		if node.TrailingComma {
			c.column++ // ','
		}
		c.specs(&node.Spec, newIndent)
		c.fillLast(node.CloseFodder,
			true,
			c.Options.PadObjects,
			newIndent.lineUp,
			currIndent.base)
		c.column++ // '}'

	case *ast.Parens:
		c.column++ // (
		newIndent := c.newIndentStrong(*openFodder(node.Inner), currIndent, c.column)
		c.Visit(node.Inner, newIndent, false)
		c.fillLast(node.CloseFodder, false, false, newIndent.lineUp, currIndent.base)
		c.column++ // )

	case *ast.Self:
		c.column += 4 // self

	case *ast.SuperIndex:
		c.column += 5 // super
		c.fill(node.DotFodder, false, false, currIndent.lineUp)
		if node.Id != nil {
			c.column++ // ".";
			newIndent := c.newIndent(node.IDFodder, currIndent, c.column)
			c.fill(node.IDFodder, false, false, newIndent.lineUp)
			c.column += len(*node.Id)
		} else {
			c.column++ // "[";
			newIndent := c.newIndent(*openFodder(node.Index), currIndent, c.column)
			c.Visit(node.Index, newIndent, false)
			c.fillLast(node.IDFodder, false, false, newIndent.lineUp, currIndent.base)
			c.column++ // "]";
		}

	case *ast.Unary:
		c.column += len(node.Op.String())
		newIndent := c.newIndent(*openFodder(node.Expr), currIndent, c.column)
		_, leftIsDollar := leftRecursiveDeep(node.Expr).(*ast.Dollar)
		c.Visit(node.Expr, newIndent, leftIsDollar)

	case *ast.Var:
		c.column += len(node.Id)
	}

}

// VisitFile corrects the whole file including the final fodder.
func (c *FixIndentation) VisitFile(body ast.Node, finalFodder ast.Fodder) {
	c.Visit(body, indent{0, 0}, false)
	c.setIndents(finalFodder, 0, 0)
}
