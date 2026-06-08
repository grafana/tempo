// Package sqlbuilders provides SQL builder library-specific checkers for SELECT * detection.
package sqlbuilders

import (
	"go/ast"
	"go/token"
	"strings"
)

// SQLBoilerChecker checks github.com/volatiletech/sqlboiler for SELECT * patterns.
type SQLBoilerChecker struct{}

// NewSQLBoilerChecker creates a new SQLBoilerChecker.
func NewSQLBoilerChecker() *SQLBoilerChecker {
	return &SQLBoilerChecker{}
}

// Name returns the name of this checker.
func (c *SQLBoilerChecker) Name() string {
	return "sqlboiler"
}

// IsApplicable checks if the call might be from sqlboiler.
func (c *SQLBoilerChecker) IsApplicable(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// SQLBoiler methods
	sqlboilerMethods := []string{
		"All", "One", "Count", "Exists",
		"Select", "Load", "Reload",
	}

	for _, method := range sqlboilerMethods {
		if sel.Sel.Name == method {
			return true
		}
	}

	// Check for qm (query mods) package
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "qm" {
			return true
		}
	}

	return false
}

// CheckSelectStar checks for SELECT * in sqlboiler calls.
func (c *SQLBoilerChecker) CheckSelectStar(call *ast.CallExpr) *SelectStarViolation {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name

	// Check qm.Select("*")
	if methodName == "Select" {
		// Check if caller is qm package
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "qm" {
			for _, arg := range call.Args {
				if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value := strings.Trim(lit.Value, "`\"")
					if value == "*" {
						return &SelectStarViolation{
							Pos:     call.Pos(),
							End:     call.End(),
							Message: "SQLBoiler qm.Select(\"*\") - explicitly specify columns",
							Builder: "sqlboiler",
							Context: "explicit_star",
						}
					}
				}
			}
		}
	}

	return nil
}

// CheckChainedCalls checks method chains for SELECT * patterns.
func (c *SQLBoilerChecker) CheckChainedCalls(call *ast.CallExpr) []*SelectStarViolation {
	var violations []*SelectStarViolation

	// SQLBoiler uses query mods passed to model methods
	// Example: models.Users(qm.Select("*")).All(ctx, db)

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return violations
	}

	// Check terminal methods
	if sel.Sel.Name == "All" || sel.Sel.Name == "One" {
		// Look for the model call that might have query mods
		if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			// Check query mod arguments for Select("*")
			hasSelect := false
			for _, arg := range innerCall.Args {
				// Check if this is a qm.Select call
				if callExpr, ok := arg.(*ast.CallExpr); ok {
					if innerSel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if innerSel.Sel.Name == "Select" {
							hasSelect = true
							// Check for "*"
							for _, selectArg := range callExpr.Args {
								if lit, ok := selectArg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
									value := strings.Trim(lit.Value, "`\"")
									if value == "*" {
										violations = append(violations, &SelectStarViolation{
											Pos:     callExpr.Pos(),
											End:     callExpr.End(),
											Message: "SQLBoiler qm.Select(\"*\") - specify columns explicitly",
											Builder: "sqlboiler",
											Context: "explicit_star",
										})
									}
								}
							}
						}
					}
				}
			}

			// If no Select query mod, it defaults to SELECT *
			if !hasSelect && len(innerCall.Args) == 0 {
				// Model().All() without query mods = SELECT *
				violations = append(violations, &SelectStarViolation{
					Pos:     innerCall.Pos(),
					End:     call.End(),
					Message: "SQLBoiler model().All() without qm.Select() defaults to SELECT *",
					Builder: "sqlboiler",
					Context: "implicit_star",
				})
			}
		}
	}

	return violations
}
