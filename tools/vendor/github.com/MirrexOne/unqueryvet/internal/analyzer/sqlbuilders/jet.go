// Package sqlbuilders provides SQL builder library-specific checkers for SELECT * detection.
package sqlbuilders

import (
	"go/ast"
	"go/token"
	"strings"
)

// JetChecker checks github.com/go-jet/jet for SELECT * patterns.
type JetChecker struct{}

// NewJetChecker creates a new JetChecker.
func NewJetChecker() *JetChecker {
	return &JetChecker{}
}

// Name returns the name of this checker.
func (c *JetChecker) Name() string {
	return "jet"
}

// IsApplicable checks if the call might be from jet.
func (c *JetChecker) IsApplicable(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		// Check for direct SELECT call
		if ident, ok := call.Fun.(*ast.Ident); ok {
			return ident.Name == "SELECT" || ident.Name == "RawStatement"
		}
		return false
	}

	// jet methods to check
	jetMethods := []string{
		"SELECT", "FROM", "WHERE",
		"AllColumns", "Star",
		"RawStatement", "Raw",
	}

	for _, method := range jetMethods {
		if sel.Sel.Name == method {
			return true
		}
	}

	return false
}

// CheckSelectStar checks for SELECT * in jet calls.
func (c *JetChecker) CheckSelectStar(call *ast.CallExpr) *SelectStarViolation {
	// Check for direct SELECT function call
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if ident.Name == "SELECT" {
			// Check arguments for AllColumns or STAR
			for _, arg := range call.Args {
				if c.isAllColumnsOrStar(arg) {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "Jet SELECT with AllColumns/STAR - specify columns explicitly",
						Builder: "jet",
						Context: "explicit_star",
					}
				}
			}
		}

		if ident.Name == "RawStatement" {
			// Check for SELECT * in raw statement
			for _, arg := range call.Args {
				if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value := strings.Trim(lit.Value, "`\"")
					upperValue := strings.ToUpper(value)
					if strings.Contains(upperValue, "SELECT *") {
						return &SelectStarViolation{
							Pos:     call.Pos(),
							End:     call.End(),
							Message: "Jet RawStatement with SELECT * - specify columns explicitly",
							Builder: "jet",
							Context: "raw_select_star",
						}
					}
				}
			}
		}
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name

	// Check for table.AllColumns
	if methodName == "AllColumns" {
		return &SelectStarViolation{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "Jet AllColumns fetches all columns (SELECT *) - specify columns explicitly",
			Builder: "jet",
			Context: "all_columns",
		}
	}

	// Check for STAR constant usage
	if methodName == "Star" {
		return &SelectStarViolation{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "Jet Star() - avoid SELECT * and specify columns explicitly",
			Builder: "jet",
			Context: "explicit_star",
		}
	}

	// Check RawStatement
	if methodName == "RawStatement" || methodName == "Raw" {
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				upperValue := strings.ToUpper(value)
				if strings.Contains(upperValue, "SELECT *") {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "Jet Raw/RawStatement with SELECT * - specify columns explicitly",
						Builder: "jet",
						Context: "raw_select_star",
					}
				}
			}
		}
	}

	return nil
}

// CheckChainedCalls checks method chains for SELECT * patterns.
func (c *JetChecker) CheckChainedCalls(call *ast.CallExpr) []*SelectStarViolation {
	var violations []*SelectStarViolation

	// Traverse the call chain looking for SELECT with AllColumns
	current := call
	for current != nil {
		// Check arguments for AllColumns
		for _, arg := range current.Args {
			if c.isAllColumnsOrStar(arg) {
				violations = append(violations, &SelectStarViolation{
					Pos:     current.Pos(),
					End:     current.End(),
					Message: "Jet SELECT chain contains AllColumns/STAR - specify columns explicitly",
					Builder: "jet",
					Context: "chained_star",
				})
			}
		}

		// Move to the next call in the chain
		sel, ok := current.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			current = innerCall
		} else {
			break
		}
	}

	return violations
}

// isAllColumnsOrStar checks if an expression represents AllColumns or STAR.
func (c *JetChecker) isAllColumnsOrStar(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// table.AllColumns or package.STAR
		return e.Sel.Name == "AllColumns" || e.Sel.Name == "STAR"
	case *ast.CallExpr:
		// Check if it's a call to AllColumns() or Star()
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			return sel.Sel.Name == "AllColumns" || sel.Sel.Name == "Star"
		}
	case *ast.Ident:
		// Direct STAR identifier
		return e.Name == "STAR" || e.Name == "AllColumns"
	}
	return false
}
