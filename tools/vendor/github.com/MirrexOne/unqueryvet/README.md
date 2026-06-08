# unqueryvet

[![Go Report Card](https://goreportcard.com/badge/github.com/MirrexOne/unqueryvet)](https://goreportcard.com/report/github.com/MirrexOne/unqueryvet)
[![GoDoc](https://godoc.org/github.com/MirrexOne/unqueryvet?status.svg)](https://godoc.org/github.com/MirrexOne/unqueryvet)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

unqueryvet is a Go static analysis tool (linter) that detects `SELECT *` usage in SQL queries and SQL builders, encouraging explicit column selection for better performance, maintainability, and API stability.

## Features

- **Detects `SELECT *` in string literals** - Finds problematic queries in your Go code
- **Constants and variables support** - Detects `SELECT *` in const and var declarations
- **String concatenation analysis** - Detects `SELECT *` in concatenated strings like `"SELECT * " + "FROM users"`
- **Format string analysis** - Detects `SELECT *` in `fmt.Sprintf`, `log.Printf`, and other format functions
- **Aliased wildcard detection** - Catches `SELECT t.*`, `SELECT alias.*` patterns
- **Subquery detection** - Finds `SELECT *` inside subqueries and nested queries
- **SQL Builder support** - Works with 8 popular SQL builders: Squirrel, GORM, SQLx, Ent, PGX, Bun, SQLBoiler, Jet
- **Auto-fix suggestions** - Provides suggested fixes for detected violations
- **File and function filtering** - Ignore specific files or functions using glob patterns
- **Configurable severity** - Set diagnostic severity to "error" or "warning"
- **Highly configurable** - Extensive configuration options for different use cases
- **Supports `//nolint:unqueryvet`** - Standard Go linting suppression
- **golangci-lint integration** - Works seamlessly with golangci-lint
- **Zero false positives** - Smart pattern recognition for acceptable `SELECT *` usage
- **Fast and lightweight** - Built on golang.org/x/tools/go/analysis

## Why avoid `SELECT *`?

- **Performance**: Selecting unnecessary columns wastes network bandwidth and memory
- **Maintainability**: Schema changes can break your application unexpectedly  
- **Security**: May expose sensitive data that shouldn't be returned
- **API Stability**: Adding new columns can break clients that depend on column order

## Informative Error Messages

Unqueryvet provides context-specific messages that explain WHY you should avoid `SELECT *`:

```go
// Basic queries
query := "SELECT * FROM users"
// avoid SELECT * - explicitly specify needed columns for better performance, maintainability and stability

// Aliased wildcards
query := "SELECT t.* FROM users t"
// avoid SELECT alias.* - explicitly specify columns like t.id, t.name for better maintainability

// String concatenation
query := "SELECT * " + "FROM users"
// avoid SELECT * in concatenated string - explicitly specify needed columns

// Format strings
query := fmt.Sprintf("SELECT * FROM %s", tableName)
// avoid SELECT * in format string - explicitly specify needed columns

// Subqueries
query := "SELECT id FROM (SELECT * FROM users)"
// avoid SELECT * in subquery - explicitly specify needed columns

// SQL Builders
query := squirrel.Select("*").From("users")
// avoid SELECT * in SQL builder - explicitly specify columns to prevent unnecessary data transfer and schema change issues

// Empty Select()
query := squirrel.Select()
// SQL builder Select() without columns defaults to SELECT * - add specific columns with .Columns() method
```

## Quick Start

### As a standalone tool

```bash
go install github.com/MirrexOne/unqueryvet/cmd/unqueryvet@latest
unqueryvet ./...
```

### With golangci-lint (Recommended)

Add to your `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - unqueryvet

  settings:
    unqueryvet:
      check-sql-builders: true
      # By default, no functions are ignored - minimal configuration
      # ignored-functions:
      #   - "fmt.Printf"
      #   - "log.Printf"  
      # allowed-patterns:
      #   - "SELECT \\* FROM information_schema\\..*" 
      #   - "SELECT \\* FROM pg_catalog\\..*"
```

## Examples

### Problematic code (will trigger warnings)

```go
// Constants with SELECT *
const QueryUsers = "SELECT * FROM users"

// Variables with SELECT *
var QueryOrders = "SELECT * FROM orders"

// String literals with SELECT *
query := "SELECT * FROM users"
rows, err := db.Query("SELECT * FROM orders WHERE status = ?", "active")

// Aliased wildcards
query := "SELECT t.* FROM users t"
query := "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id"

// String concatenation
query := "SELECT * " + "FROM users " + "WHERE id = ?"

// Format strings
query := fmt.Sprintf("SELECT * FROM %s WHERE id = %d", table, id)

// Subqueries
query := "SELECT id FROM (SELECT * FROM users)"
query := "SELECT * FROM users WHERE id IN (SELECT * FROM orders)"

// SQL builders with SELECT *
query := squirrel.Select("*").From("products")
query := builder.Select().Columns("*").From("inventory")
```

### Good code (recommended)

```go
// Constants with explicit columns
const QueryUsers = "SELECT id, name, email FROM users"

// Variables with explicit columns
var QueryOrders = "SELECT id, status, total FROM orders"

// String literals with explicit column selection
query := "SELECT id, name, email FROM users"
rows, err := db.Query("SELECT id, total FROM orders WHERE status = ?", "active")

// SQL builders with explicit columns
query := squirrel.Select("id", "name", "price").From("products")
query := builder.Select().Columns("id", "quantity", "location").From("inventory")
```

### Acceptable SELECT * usage (won't trigger warnings)

```go
// System/meta queries
"SELECT * FROM information_schema.tables"
"SELECT * FROM pg_catalog.pg_tables"

// Aggregate functions
"SELECT COUNT(*) FROM users"
"SELECT MAX(*) FROM scores" 

// With nolint suppression
query := "SELECT * FROM debug_table" //nolint:unqueryvet
```

## Configuration

Unqueryvet is highly configurable to fit your project's needs:

```yaml
version: "2"

linters:
  settings:
    unqueryvet:
      # Enable/disable SQL builder checking (default: true)
      check-sql-builders: true

      # Enable/disable aliased wildcard detection like SELECT t.* (default: true)
      check-aliased-wildcard: true

      # Enable/disable string concatenation analysis (default: true)
      check-string-concat: true

      # Enable/disable format string analysis like fmt.Sprintf (default: true)
      check-format-strings: true

      # Enable/disable strings.Builder analysis (default: true)
      check-string-builder: true

      # Enable/disable subquery analysis (default: true)
      check-subqueries: true

      # Diagnostic severity: "error" or "warning" (default: "warning")
      severity: warning

      # SQL builder libraries to check (all enabled by default)
      sql-builders:
        squirrel: true
        gorm: true
        sqlx: true
        ent: true
        pgx: true
        bun: true
        sqlboiler: true
        jet: true

      # Patterns for files to ignore (glob patterns)
      # ignored-files:
      #   - "*_test.go"
      #   - "testdata/**"
      #   - "mock_*.go"

      # Functions to ignore (regex patterns)
      # ignored-functions:
      #   - "debug\\..*"
      #   - "test.*"

      # Default allowed patterns (automatically included):
      # - COUNT(*), MAX(*), MIN(*) functions
      # - information_schema, pg_catalog, sys schema queries
      # You can add more patterns if needed:
      # allowed-patterns:
      #   - "SELECT \\* FROM temp_.*"
```

## Supported SQL Builders

Unqueryvet supports 8 popular SQL builders out of the box:

| Library | Package | Detection |
|---------|---------|-----------|
| **Squirrel** | `github.com/Masterminds/squirrel` | `Select("*")`, `Columns("*")` |
| **GORM** | `gorm.io/gorm` | `Select("*")`, raw queries |
| **SQLx** | `github.com/jmoiron/sqlx` | `Select()`, raw queries |
| **Ent** | `entgo.io/ent` | Query builder patterns |
| **PGX** | `github.com/jackc/pgx` | `Query()`, `QueryRow()` |
| **Bun** | `github.com/uptrace/bun` | `NewSelect()`, raw queries |
| **SQLBoiler** | `github.com/volatiletech/sqlboiler` | Generated query methods |
| **Jet** | `github.com/go-jet/jet` | `SELECT()`, `STAR` |

Each checker can be individually enabled/disabled via configuration.

## Auto-Fix Suggestions

Unqueryvet provides automatic fix suggestions for detected violations. When used with editors that support LSP or with `golangci-lint --fix`, you can quickly fix issues:

```go
// Before (violation detected)
query := "SELECT * FROM users"

// After auto-fix (with TODO placeholder)
query := "SELECT id, /* TODO: specify columns */ FROM users"

// SQL builder before
squirrel.Select("*").From("users")

// SQL builder after auto-fix
squirrel.Select("id", /* TODO: specify columns */).From("users")
```

The auto-fix adds `/* TODO: specify columns */` as a reminder to manually specify the columns you actually need.

## Integration Examples

### GitHub Actions

```yaml
name: Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v5
    - uses: actions/setup-go@v6
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v8
      with:
        version: latest
        args: --enable unqueryvet
```

## Command Line Options

When used as a standalone tool:

```bash
# Check all packages
unqueryvet ./...

# Check specific packages
unqueryvet ./cmd/... ./internal/...

# With custom config file
unqueryvet -config=.unqueryvet.yml ./...

# Verbose output
unqueryvet -v ./...
```

## Performance

Unqueryvet is designed to be fast and lightweight:

- **Parallel processing**: Analyzes multiple files concurrently
- **Incremental analysis**: Only analyzes changed files when possible
- **Minimal memory footprint**: Efficient AST traversal
- **Smart caching**: Reuses analysis results when appropriate

## Advanced Usage

### Custom Patterns

You can define custom regex patterns for acceptable `SELECT *` usage:

```yaml
allowed-patterns:
  # Allow SELECT * from temporary tables
  - "SELECT \\* FROM temp_\\w+"
  # Allow SELECT * in migration scripts  
  - "SELECT \\* FROM.*-- migration"
  # Allow SELECT * for specific schemas
  - "SELECT \\* FROM audit\\..+"
```

### Integration with Custom SQL Builders

For custom SQL builders, Unqueryvet looks for these patterns:

```go
// Method chaining
builder.Select("*")          // Direct SELECT *
builder.Select().Columns("*") // Chained SELECT *

// Variable tracking  
query := builder.Select()    // Empty select
// If no .Columns() call follows, triggers warning
```

### Running Tests

```bash
go test ./...
go test -race ./...
go test -bench=. ./...
```

### Development Setup

```bash
git clone https://github.com/MirrexOne/unqueryvet.git
cd unqueryvet
go mod tidy
go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Bug Reports**: [GitHub Issues](https://github.com/MirrexOne/unqueryvet/issues)
