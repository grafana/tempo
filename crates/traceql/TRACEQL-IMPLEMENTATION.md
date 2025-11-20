# TraceQL Implementation

This document describes the TraceQL implementation in tempo-datafusion.

## Overview

TraceQL is a query language for distributed traces, inspired by PromQL and LogQL. This implementation adds TraceQL support to tempo-datafusion by parsing TraceQL queries and converting them to DataFusion SQL queries.

## Architecture

The implementation is split into 4 modules located in `src/traceql/`:

### 1. AST Module (`ast.rs`)
Defines the Abstract Syntax Tree structures for TraceQL:
- `TraceQLQuery` - Root query structure
- `QueryExpr` - Can be a simple span filter or structural query
- `SpanFilter` - Expression inside `{ }`
- `Expr` - Expression types (binary ops, unary ops, comparisons)
- `FieldRef` - Field references with scopes (span, resource, intrinsic, unscoped)
- `Value` - Value types (string, integer, float, bool, duration, status, span kind)
- `PipelineOp` - Pipeline operations (rate, count, avg, sum, min, max)

### 2. Lexer Module (`lexer.rs`)
Tokenizes TraceQL query strings:
- Handles operators: `=`, `!=`, `>`, `>=`, `<`, `<=`, `=~`, `!~`, `&&`, `||`, `!`, `>>`
- Parses literals: strings, integers, floats, booleans
- Recognizes duration units: `ns`, `us`, `ms`, `s`, `m`, `h`
- Handles delimiters: `{`, `}`, `(`, `)`, `.`, `|`

### 3. Parser Module (`parser.rs`)
Builds AST from tokens:
- Recursive descent parser with operator precedence
- Handles parenthesized expressions
- Supports both simple and structural queries
- Parses pipeline operations

### 4. Converter Module (`converter.rs`)
Converts TraceQL AST to DataFusion SQL:
- Maps TraceQL fields to SQL columns
- Converts duration values to nanoseconds
- Maps status/kind enums to integer codes
- Handles span attributes via dedicated columns or map access
- Wraps pipeline operations in CTEs

## Usage

### Command Line

Prefix TraceQL queries with `|`:

```bash
# Simple filter
cargo run -- --config local.toml --exec '|{ span.http.method = "GET" }'

# Duration filter
cargo run -- --config local.toml --exec '|{ duration > 100ms }'

# Complex query
cargo run -- --config local.toml --exec '|{ span.http.method = "POST" && duration > 1s }'

# With pipeline
cargo run -- --config local.toml --exec '|{ } | rate()'
```

### REPL

Start the REPL and type TraceQL queries with `|` prefix:

```bash
cargo run -- --config local.toml
datafusion> |{ span.http.method = "GET" }
```

## Field Mappings

### Span Attributes
- `span.http.method` → `HttpMethod` column
- `span.http.url` → `HttpUrl` column
- `span.http.status_code` → `HttpStatusCode` column
- `span.http.response_code` → `HttpStatusCode` column
- `span.*` → `Attrs['*']` map access

### Intrinsic Fields
- `name` → `Name` column
- `duration` → `DurationNano` column (converted to nanoseconds)
- `status` → `StatusCode` column (0=unset, 1=ok, 2=error)
- `kind` → `Kind` column (0=unspecified, 1=internal, 2=server, 3=client, 4=producer, 5=consumer)

### Unscoped Fields
- `.fieldname` → `Attrs['fieldname']` map access

## Supported Features

### Operators
- Comparison: `=`, `!=`, `>`, `>=`, `<`, `<=`
- Regex: `=~`, `!~`
- Logical: `&&`, `||`, `!`
- Structural: `>>`

### Value Types
- Strings: `"value"`
- Integers: `42`, `-10`
- Floats: `3.14`, `-0.5`
- Booleans: `true`, `false`
- Durations: `100ms`, `1s`, `5m`
- Status: `unset`, `ok`, `error`
- Span kinds: `unspecified`, `internal`, `server`, `client`, `producer`, `consumer`

### Pipeline Operations
- `rate()` - Count of spans
- `count()` - Count of spans

## Examples

### Find GET requests
```traceql
{ span.http.method = "GET" }
```
→ SQL: `SELECT * FROM spans WHERE "HttpMethod" = 'GET'`

### Find slow requests
```traceql
{ duration > 100ms }
```
→ SQL: `SELECT * FROM spans WHERE "DurationNano" > 100000000`

### Find errors
```traceql
{ status = error }
```
→ SQL: `SELECT * FROM spans WHERE "StatusCode" = 2`

### Complex query
```traceql
{ span.http.method = "POST" && (span.http.status_code >= 500 || duration > 2s) }
```
→ SQL: `SELECT * FROM spans WHERE ("HttpMethod" = 'POST' && (("HttpStatusCode" >= 500) || ("DurationNano" > 2000000000)))`

### With aggregation
```traceql
{ span.http.method = "GET" } | rate()
```
→ SQL:
```sql
WITH base_spans AS (
SELECT * FROM spans WHERE "HttpMethod" = 'GET'
)
SELECT COUNT(*) as rate FROM base_spans
```

### Structural query (parent-child)
```traceql
{ span.http.method = "GET" } >> { span.db.system = "postgresql" }
```
→ SQL: Uses nested set model to find database spans that are children of GET requests

## Limitations

- Resource attributes not yet supported
- `nestedSetParent` intrinsic not fully supported
- Multiple pipeline operations not yet supported
- Advanced aggregations (avg, sum, min, max by group) not yet implemented
- Trace-level operations not yet implemented

## Testing

Run tests with:
```bash
cargo test traceql
```

All tests should pass (18 tests total).

## Future Enhancements

1. Support resource attributes
2. Implement full nested set operations
3. Add more pipeline operations (by, quantile, etc.)
4. Support trace-level operations
5. Add span relationship queries (descendant, sibling, etc.)
6. Optimize SQL generation for better performance
