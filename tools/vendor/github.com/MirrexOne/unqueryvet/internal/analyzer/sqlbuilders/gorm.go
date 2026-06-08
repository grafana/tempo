// Package sqlbuilders provides SQL builder library-specific checkers for SELECT * detection.
package sqlbuilders

import (
	"go/ast"
	"go/token"
	"strings"
)

// GORMChecker checks gorm.io/gorm for SELECT * patterns.
type GORMChecker struct{}

// NewGORMChecker creates a new GORMChecker.
func NewGORMChecker() *GORMChecker {
	return &GORMChecker{}
}

// Name returns the name of this checker.
func (c *GORMChecker) Name() string {
	return "gorm"
}

// IsApplicable checks if the call might be from GORM.
func (c *GORMChecker) IsApplicable(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// GORM methods to check
	gormMethods := []string{
		"Select", "Find", "First", "Last", "Take", "Scan",
		"Model", "Table", "Raw", "Exec", "Pluck",
		"Preload", "Joins", "Where", "Or", "Not",
	}

	for _, method := range gormMethods {
		if sel.Sel.Name == method {
			return true
		}
	}

	// Check for gorm package or DB type
	if ident, ok := sel.X.(*ast.Ident); ok {
		lowerName := strings.ToLower(ident.Name)
		if lowerName == "gorm" || lowerName == "db" {
			return true
		}
	}

	return false
}

// CheckSelectStar checks for SELECT * in GORM calls.
func (c *GORMChecker) CheckSelectStar(call *ast.CallExpr) *SelectStarViolation {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	methodName := sel.Sel.Name

	// Check db.Select("*")
	if methodName == "Select" {
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				if value == "*" {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "GORM Select(\"*\") - explicitly specify columns",
						Builder: "gorm",
						Context: "explicit_star",
					}
				}
				// Check for SELECT * in raw SQL inside Select
				upperValue := strings.ToUpper(value)
				if strings.Contains(upperValue, "SELECT *") {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "GORM Select() contains SELECT * - specify columns explicitly",
						Builder: "gorm",
						Context: "raw_select_star",
					}
				}
			}
		}
	}

	// Check db.Raw("SELECT * FROM ...")
	if methodName == "Raw" {
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				upperValue := strings.ToUpper(value)
				if strings.Contains(upperValue, "SELECT *") {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "GORM Raw() with SELECT * - specify columns explicitly",
						Builder: "gorm",
						Context: "raw_select_star",
					}
				}
			}
		}
	}

	// Check db.Exec("SELECT * FROM ...")
	if methodName == "Exec" {
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				value := strings.Trim(lit.Value, "`\"")
				upperValue := strings.ToUpper(value)
				if strings.Contains(upperValue, "SELECT *") {
					return &SelectStarViolation{
						Pos:     call.Pos(),
						End:     call.End(),
						Message: "GORM Exec() with SELECT * - specify columns explicitly",
						Builder: "gorm",
						Context: "raw_select_star",
					}
				}
			}
		}
	}

	return nil
}

// CheckChainedCalls checks method chains for SELECT * patterns.
func (c *GORMChecker) CheckChainedCalls(call *ast.CallExpr) []*SelectStarViolation {
	var violations []*SelectStarViolation

	// Track chain state
	hasModel := false
	hasSelect := false
	var modelCall *ast.CallExpr

	// Traverse the call chain
	current := call
	for current != nil {
		sel, ok := current.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		switch sel.Sel.Name {
		case "Model", "Table":
			hasModel = true
			modelCall = current
		case "Select":
			hasSelect = true
			// Check for "*" argument
			for _, arg := range current.Args {
				if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					value := strings.Trim(lit.Value, "`\"")
					if value == "*" {
						violations = append(violations, &SelectStarViolation{
							Pos:     current.Pos(),
							End:     current.End(),
							Message: "GORM Select(\"*\") in chain - specify columns explicitly",
							Builder: "gorm",
							Context: "chained_star",
						})
					}
				}
			}
		case "Find", "First", "Last", "Take", "Scan":
			// Terminal methods - check if we have Model without Select
			if hasModel && !hasSelect && modelCall != nil {
				violations = append(violations, &SelectStarViolation{
					Pos:     modelCall.Pos(),
					End:     current.End(),
					Message: "GORM Model() with Find/First without Select() defaults to SELECT *",
					Builder: "gorm",
					Context: "implicit_star",
				})
			}
		case "Preload":
			// Preload often uses SELECT * for eager loading - this is common but worth flagging
			if len(current.Args) > 0 {
				// Only flag if there's no second argument (no custom SQL)
				if len(current.Args) == 1 {
					// Could add a warning about Preload SELECT *
				}
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
