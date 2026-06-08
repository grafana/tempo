// Package sqlbuilders provides SQL builder library-specific checkers for SELECT * detection.
package sqlbuilders

import (
	"go/ast"
	"go/token"
	"strings"
)

// SquirrelChecker checks github.com/Masterminds/squirrel for SELECT * patterns.
type SquirrelChecker struct{}

// NewSquirrelChecker creates a new SquirrelChecker.
func NewSquirrelChecker() *SquirrelChecker {
	return &SquirrelChecker{}
}

// Name returns the name of this checker.
func (c *SquirrelChecker) Name() string {
	return "squirrel"
}

// IsApplicable checks if the call might be from Squirrel.
func (c *SquirrelChecker) IsApplicable(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Squirrel methods to check
	squirrelMethods := []string{
		"Select", "Columns", "Column",
		"SelectBuilder", "InsertBuilder", "UpdateBuilder", "DeleteBuilder",
	}

	for _, method := range squirrelMethods {
		if sel.Sel.Name == method {
			return true
		}
	}

	// Check for squirrel package prefix
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "squirrel" || ident.Name == "sq" {
			return true
		}
	}

	return false
}

// CheckSelectStar checks for SELECT * in Squirrel calls.
func (c *SquirrelChecker) CheckSelectStar(call *ast.CallExpr) *SelectStarViolation {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name

	// Check squirrel.Select("*") or builder.Select("*")
	if methodName == "Select" {
		// Empty Select() means SELECT *
		if len(call.Args) == 0 {
			return &SelectStarViolation{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: "Squirrel Select() without columns defaults to SELECT * - add specific columns",
				Builder: "squirrel",
				Context: "empty_select",
			}
		}

		// Check for "*" in arguments
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				if value == "*" || value == "" {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "Squirrel Select(\"*\") - explicitly specify columns",
						Builder: "squirrel",
						Context: "explicit_star",
					}
				}
			}
		}
	}

	// Check Columns("*") or Column("*")
	if methodName == "Columns" || methodName == "Column" {
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				if value == "*" {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "Squirrel Columns(\"*\") - explicitly specify columns",
						Builder: "squirrel",
						Context: "explicit_star",
					}
				}
			}
		}
	}

	return nil
}

// CheckChainedCalls checks method chains for SELECT * patterns.
func (c *SquirrelChecker) CheckChainedCalls(call *ast.CallExpr) []*SelectStarViolation {
	var violations []*SelectStarViolation

	// Traverse the call chain
	current := call
	hasSelect := false
	hasColumns := false
	var selectCall *ast.CallExpr

	for current != nil {
		sel, ok := current.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		switch sel.Sel.Name {
		case "Select":
			hasSelect = true
			selectCall = current
			// Check if Select has arguments
			if len(current.Args) > 0 {
				hasColumns = true
				// Check for "*" argument
				for _, arg := range current.Args {
					if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						value := strings.Trim(lit.Value, "`\"")
						if value == "*" {
							violations = append(violations, &SelectStarViolation{
								Pos:     current.Pos(),
								End:     current.End(),
								Message: "Squirrel Select(\"*\") in chain - specify columns explicitly",
								Builder: "squirrel",
								Context: "chained_star",
							})
						}
					}
				}
			}
		case "Columns", "Column":
			hasColumns = true
			// Check for "*" in Columns/Column
			for _, arg := range current.Args {
				if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value := strings.Trim(lit.Value, "`\"")
					if value == "*" {
						violations = append(violations, &SelectStarViolation{
							Pos:     current.Pos(),
							End:     current.End(),
							Message: "Squirrel Columns(\"*\") in chain - specify columns explicitly",
							Builder: "squirrel",
							Context: "chained_star",
						})
					}
				}
			}
		case "From", "Where", "Join", "LeftJoin", "RightJoin", "InnerJoin":
			// Terminal methods - check if we have Select without columns
			if hasSelect && !hasColumns && selectCall != nil && len(selectCall.Args) == 0 {
				violations = append(violations, &SelectStarViolation{
					Pos:     selectCall.Pos(),
					End:     selectCall.End(),
					Message: "Squirrel Select() without columns in chain defaults to SELECT *",
					Builder: "squirrel",
					Context: "empty_select_chain",
				})
			}
		}

		// Move to the next call in the chain
		if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			current = innerCall
		} else {
			break
		}
	}

	return violations
}
