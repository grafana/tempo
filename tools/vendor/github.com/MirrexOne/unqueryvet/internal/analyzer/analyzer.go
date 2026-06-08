// Package analyzer provides the SQL static analysis implementation for detecting SELECT * usage.
package analyzer

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/MirrexOne/unqueryvet/internal/analyzer/sqlbuilders"
	"github.com/MirrexOne/unqueryvet/pkg/config"
)

const (
	// selectKeyword is the SQL SELECT method name in builders
	selectKeyword = "Select"
	// columnKeyword is the SQL Column method name in builders
	columnKeyword = "Column"
	// columnsKeyword is the SQL Columns method name in builders
	columnsKeyword = "Columns"
	// defaultWarningMessage is the standard warning for SELECT * usage
	defaultWarningMessage = "avoid SELECT * - explicitly specify needed columns for better performance, maintainability and stability"
)

// Precompiled regex patterns for performance
var (
	// aliasedWildcardPattern matches SELECT alias.* patterns like "SELECT t.*", "SELECT u.*, o.*"
	aliasedWildcardPattern = regexp.MustCompile(`(?i)SELECT\s+(?:[A-Za-z_][A-Za-z0-9_]*\s*\.\s*\*\s*,?\s*)+`)

	// subquerySelectStarPattern matches SELECT * in subqueries like "(SELECT * FROM ...)"
	subquerySelectStarPattern = regexp.MustCompile(`(?i)\(\s*SELECT\s+\*`)
)

// NewAnalyzer creates the Unqueryvet analyzer with enhanced logic for production use
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "unqueryvet",
		Doc:      "detects SELECT * in SQL queries and SQL builders, preventing performance issues and encouraging explicit column selection",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// NewAnalyzerWithSettings creates analyzer with provided settings for golangci-lint integration
func NewAnalyzerWithSettings(s config.UnqueryvetSettings) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "unqueryvet",
		Doc:  "detects SELECT * in SQL queries and SQL builders, preventing performance issues and encouraging explicit column selection",
		Run: func(pass *analysis.Pass) (any, error) {
			return RunWithConfig(pass, &s)
		},
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// RunWithConfig performs analysis with provided configuration
// This is the main entry point for configured analysis
func RunWithConfig(pass *analysis.Pass, cfg *config.UnqueryvetSettings) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Use provided configuration or default if nil
	if cfg == nil {
		defaultSettings := config.DefaultSettings()
		cfg = &defaultSettings
	}

	// Create filter context for efficient filtering
	filter, err := NewFilterContext(cfg)
	if err != nil {
		// If filter creation fails, continue without filtering
		filter = nil
	}

	// Check if current file should be ignored
	if filter != nil && len(pass.Files) > 0 {
		fileName := pass.Fset.File(pass.Files[0].Pos()).Name()
		if filter.IsIgnoredFile(fileName) {
			return nil, nil
		}
	}

	// Create SQL builder registry for checking SQL builder patterns
	var builderRegistry *sqlbuilders.Registry
	if cfg.CheckSQLBuilders {
		builderRegistry = sqlbuilders.NewRegistry(&cfg.SQLBuilders)
	}

	// Define AST node types we're interested in
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),   // Function/method calls
		(*ast.File)(nil),       // Files (for SQL builder analysis)
		(*ast.AssignStmt)(nil), // Assignment statements for standalone literals
		(*ast.GenDecl)(nil),    // General declarations (const, var, type)
		(*ast.BinaryExpr)(nil), // Binary expressions for string concatenation
	}

	// Walk through all AST nodes and analyze them
	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			// Analyze SQL builders only if enabled in configuration
			if cfg.CheckSQLBuilders {
				analyzeSQLBuilders(pass, node)
			}
		case *ast.AssignStmt:
			// Check assignment statements for standalone SQL literals
			checkAssignStmt(pass, node, cfg)
		case *ast.GenDecl:
			// Check constant and variable declarations
			checkGenDecl(pass, node, cfg)
		case *ast.CallExpr:
			// Check if function should be ignored
			if filter != nil && filter.IsIgnoredFunction(node) {
				return
			}

			// Check format functions (fmt.Sprintf, etc.)
			if cfg.CheckFormatStrings && CheckFormatFunction(pass, node, cfg) {
				pass.Report(analysis.Diagnostic{
					Pos:     node.Pos(),
					Message: getDetailedWarningMessage("format_string"),
				})
				return
			}

			// Check SQL builder patterns
			if builderRegistry != nil && builderRegistry.HasCheckers() {
				violations := builderRegistry.Check(node)
				for _, v := range violations {
					pass.Report(analysis.Diagnostic{
						Pos:     v.Pos,
						End:     v.End,
						Message: v.Message,
					})
				}
				if len(violations) > 0 {
					return
				}
			}

			// Analyze function calls for SQL with SELECT * usage
			checkCallExpr(pass, node, cfg)

		case *ast.BinaryExpr:
			// Check string concatenation for SELECT *
			if cfg.CheckStringConcat && CheckConcatenation(pass, node, cfg) {
				pass.Report(analysis.Diagnostic{
					Pos:     node.Pos(),
					Message: getDetailedWarningMessage("concat"),
				})
			}
		}
	})

	return nil, nil
}

// run performs the main analysis of Go code files for SELECT * usage
func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Define AST node types we're interested in
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),   // Function/method calls
		(*ast.File)(nil),       // Files (for SQL builder analysis)
		(*ast.AssignStmt)(nil), // Assignment statements for standalone literals
		(*ast.GenDecl)(nil),    // General declarations (const, var)
	}

	// Always use default settings since passing settings through ResultOf doesn't work reliably
	defaultSettings := config.DefaultSettings()
	cfg := &defaultSettings

	// Walk through all AST nodes and analyze them
	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			// Analyze SQL builders only if enabled in configuration
			if cfg.CheckSQLBuilders {
				analyzeSQLBuilders(pass, node)
			}
		case *ast.AssignStmt:
			// Check assignment statements for standalone SQL literals
			checkAssignStmt(pass, node, cfg)
		case *ast.GenDecl:
			// Check constant and variable declarations
			checkGenDecl(pass, node, cfg)
		case *ast.CallExpr:
			// Analyze function calls for SQL with SELECT * usage
			checkCallExpr(pass, node, cfg)
		}
	})

	return nil, nil
}

// checkAssignStmt checks assignment statements for standalone SQL literals
func checkAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, cfg *config.UnqueryvetSettings) {
	// Check right-hand side expressions for string literals with SELECT *
	for _, expr := range stmt.Rhs {
		// Only check direct string literals, not function calls
		if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			content := normalizeSQLQuery(lit.Value)
			if isSelectStarQuery(content, cfg) {
				pass.Report(analysis.Diagnostic{
					Pos:     lit.Pos(),
					Message: getWarningMessage(),
				})
			}
		}
	}
}

// checkGenDecl checks general declarations (const, var) for SELECT * in SQL queries
func checkGenDecl(pass *analysis.Pass, decl *ast.GenDecl, cfg *config.UnqueryvetSettings) {
	// Only check const and var declarations
	if decl.Tok != token.CONST && decl.Tok != token.VAR {
		return
	}

	// Iterate through all specifications in the declaration
	for _, spec := range decl.Specs {
		// Type assert to ValueSpec (const/var specifications)
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Check all values in the specification
		for _, value := range valueSpec.Values {
			// Only check direct string literals
			if lit, ok := value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				content := normalizeSQLQuery(lit.Value)
				if isSelectStarQuery(content, cfg) {
					pass.Report(analysis.Diagnostic{
						Pos:     lit.Pos(),
						Message: getWarningMessage(),
					})
				}
			}
		}
	}
}

// checkCallExpr analyzes function calls for SQL with SELECT * usage
// Includes checking arguments and SQL builders
func checkCallExpr(pass *analysis.Pass, call *ast.CallExpr, cfg *config.UnqueryvetSettings) {
	// Check SQL builders for SELECT * in arguments
	if cfg.CheckSQLBuilders && isSQLBuilderSelectStar(call) {
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			Message: getDetailedWarningMessage("sql_builder"),
		})
		return
	}

	// Check function call arguments for strings with SELECT *
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			content := normalizeSQLQuery(lit.Value)
			if isSelectStarQuery(content, cfg) {
				pass.Report(analysis.Diagnostic{
					Pos:     lit.Pos(),
					Message: getWarningMessage(),
				})
			}
		}
	}
}

// NormalizeSQLQuery normalizes SQL query for analysis with advanced escape sequence handling.
// Exported for testing purposes.
func NormalizeSQLQuery(query string) string {
	return normalizeSQLQuery(query)
}

func normalizeSQLQuery(query string) string {
	if len(query) < 2 {
		return query
	}

	first, last := query[0], query[len(query)-1]

	// 1. Handle different quote types with escape sequence processing
	if first == '"' && last == '"' {
		// For regular strings check for escape sequences
		if !strings.Contains(query, "\\") {
			query = trimQuotes(query)
		} else if unquoted, err := strconv.Unquote(query); err == nil {
			// Use standard Go unquoting for proper escape sequence handling
			query = unquoted
		} else {
			// Fallback: simple quote removal
			query = trimQuotes(query)
		}
	} else if first == '`' && last == '`' {
		// Raw strings - simply remove backticks
		query = trimQuotes(query)
	}

	// 2. Process comments line by line before normalization
	lines := strings.Split(query, "\n")
	var processedParts []string

	for _, line := range lines {
		// Remove comments from current line
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}

		// Add non-empty lines
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			processedParts = append(processedParts, trimmed)
		}
	}

	// 3. Reassemble query and normalize
	query = strings.Join(processedParts, " ")
	query = strings.ToUpper(query)
	query = strings.ReplaceAll(query, "\t", " ")
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")

	return strings.TrimSpace(query)
}

// trimQuotes removes first and last character (quotes)
func trimQuotes(query string) string {
	return query[1 : len(query)-1]
}

// IsSelectStarQuery determines if query contains SELECT * with enhanced allowed patterns support.
// Exported for testing purposes.
func IsSelectStarQuery(query string, cfg *config.UnqueryvetSettings) bool {
	return isSelectStarQuery(query, cfg)
}

func isSelectStarQuery(query string, cfg *config.UnqueryvetSettings) bool {
	// Check allowed patterns first - if query matches an allowed pattern, ignore it
	for _, pattern := range cfg.AllowedPatterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			return false
		}
	}

	upperQuery := strings.ToUpper(query)

	// Check for SELECT * in query (case-insensitive)
	if strings.Contains(upperQuery, "SELECT *") { //nolint:unqueryvet
		// Ensure this is actually an SQL query by checking for SQL keywords
		sqlKeywords := []string{"FROM", "WHERE", "JOIN", "GROUP", "ORDER", "HAVING", "UNION", "LIMIT"}
		for _, keyword := range sqlKeywords {
			if strings.Contains(upperQuery, keyword) {
				return true
			}
		}

		// Also check if it's just "SELECT *" without other keywords (still problematic)
		trimmed := strings.TrimSpace(upperQuery)
		if trimmed == "SELECT *" {
			return true
		}
	}

	// Check for SELECT alias.* patterns (e.g., SELECT t.*, SELECT u.*, o.*)
	if cfg.CheckAliasedWildcard && isSelectAliasStarQuery(query) {
		return true
	}

	// Check for SELECT * in subqueries (e.g., (SELECT * FROM ...))
	if cfg.CheckSubqueries && isSelectStarInSubquery(query) {
		return true
	}

	return false
}

// isSelectAliasStarQuery detects SELECT alias.* patterns like "SELECT t.*", "SELECT u.*, o.*"
func isSelectAliasStarQuery(query string) bool {
	return aliasedWildcardPattern.MatchString(query)
}

// isSelectStarInSubquery detects SELECT * in subqueries like "(SELECT * FROM ...)"
func isSelectStarInSubquery(query string) bool {
	return subquerySelectStarPattern.MatchString(query)
}

// getWarningMessage returns informative warning message
func getWarningMessage() string {
	return defaultWarningMessage
}

// getDetailedWarningMessage returns context-specific warning message
func getDetailedWarningMessage(context string) string {
	switch context {
	case "sql_builder":
		return "avoid SELECT * in SQL builder - explicitly specify columns to prevent unnecessary data transfer and schema change issues"
	case "nested":
		return "avoid SELECT * in subquery - can cause performance issues and unexpected results when schema changes"
	case "empty_select":
		return "SQL builder Select() without columns defaults to SELECT * - add specific columns with .Columns() method"
	case "aliased_wildcard":
		return "avoid SELECT alias.* - explicitly specify columns like alias.id, alias.name for better maintainability"
	case "subquery":
		return "avoid SELECT * in subquery - specify columns explicitly to prevent issues when schema changes"
	case "concat":
		return "avoid SELECT * in concatenated query - explicitly specify needed columns"
	case "format_string":
		return "avoid SELECT * in format string - explicitly specify needed columns"
	default:
		return defaultWarningMessage
	}
}

// isSQLBuilderSelectStar checks SQL builder method calls for SELECT * usage
func isSQLBuilderSelectStar(call *ast.CallExpr) bool {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check that this is a Select method call
	if fun.Sel == nil || fun.Sel.Name != selectKeyword {
		return false
	}

	if len(call.Args) == 0 {
		return false
	}

	// Check Select method arguments for "*" or empty strings
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			value := strings.Trim(lit.Value, "`\"")
			// Consider both "*" and empty strings in Select() as problematic
			if value == "*" || value == "" {
				return true
			}
		}
	}

	return false
}

// analyzeSQLBuilders performs advanced SQL builder analysis
// Key logic for handling edge-cases like Select().Columns("*")
func analyzeSQLBuilders(pass *analysis.Pass, file *ast.File) {
	// Track SQL builder variables and their state
	builderVars := make(map[string]*ast.CallExpr) // Variables with empty Select() calls
	hasColumns := make(map[string]bool)           // Flag: were columns added for variable

	// First pass: find variables created with empty Select() calls
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Analyze assignments like: query := builder.Select()
			for i, expr := range node.Rhs {
				if call, ok := expr.(*ast.CallExpr); ok {
					if isEmptySelectCall(call) {
						// Found empty Select() call, remember the variable
						if i < len(node.Lhs) {
							if ident, ok := node.Lhs[i].(*ast.Ident); ok {
								builderVars[ident.Name] = call
								hasColumns[ident.Name] = false
							}
						}
					}
				}
			}
		}
		return true
	})

	// Second pass: check usage of Columns/Column methods
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				// Check calls to Columns() or Column() methods
				if sel.Sel != nil && (sel.Sel.Name == columnsKeyword || sel.Sel.Name == columnKeyword) {
					// Check for "*" in arguments
					if hasStarInColumns(node) {
						pass.Report(analysis.Diagnostic{
							Pos:     node.Pos(),
							Message: getDetailedWarningMessage("sql_builder"),
						})
					}

					// Update variable state - columns were added
					if ident, ok := sel.X.(*ast.Ident); ok {
						if _, exists := builderVars[ident.Name]; exists {
							if !hasStarInColumns(node) {
								hasColumns[ident.Name] = true
							}
						}
					}
				}
			}

			// Check call chains like builder.Select().Columns("*")
			if isSelectWithColumns(node) {
				if hasStarInColumns(node) {
					if sel, ok := node.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
						pass.Report(analysis.Diagnostic{
							Pos:     node.Pos(),
							Message: getDetailedWarningMessage("sql_builder"),
						})
					}
				}
				return true
			}
		}
		return true
	})

	// Final check: warn about builders with empty Select() without subsequent columns
	for varName, call := range builderVars {
		if !hasColumns[varName] {
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				Message: getDetailedWarningMessage("empty_select"),
			})
		}
	}
}

// isEmptySelectCall checks if call is an empty Select()
func isEmptySelectCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel != nil && sel.Sel.Name == selectKeyword && len(call.Args) == 0 {
			return true
		}
	}
	return false
}

// isSelectWithColumns checks call chains like Select().Columns()
func isSelectWithColumns(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel != nil && (sel.Sel.Name == columnsKeyword || sel.Sel.Name == columnKeyword) {
			// Check that previous call in chain is Select()
			if innerCall, ok := sel.X.(*ast.CallExpr); ok {
				return isEmptySelectCall(innerCall)
			}
		}
	}
	return false
}

// hasStarInColumns checks if call arguments contain "*" symbol
func hasStarInColumns(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			value := strings.Trim(lit.Value, "`\"")
			if value == "*" {
				return true
			}
		}
	}
	return false
}
