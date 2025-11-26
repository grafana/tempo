/// TraceQL query language support
///
/// This module provides parsing and conversion of TraceQL queries to DataFusion SQL.
///
/// # Architecture
///
/// 1. **Lexer** (`lexer.rs`) - Tokenizes TraceQL query strings
/// 2. **Parser** (`parser.rs`) - Builds an Abstract Syntax Tree (AST) from tokens
/// 3. **AST** (`ast.rs`) - Defines the TraceQL AST structures
/// 4. **Converter** (`converter.rs`) - Converts TraceQL AST to DataFusion SQL
///
/// # Usage
///
/// ```rust,ignore
/// use tempo_datafusion::traceql;
///
/// // Parse a TraceQL query
/// let sql = traceql::traceql_to_sql_string(r#"{ span.http.method = "GET" }"#)?;
///
/// // Execute the generated SQL against DataFusion
/// let df = ctx.sql(&sql).await?;
/// ```
pub mod ast;
pub mod converter;
pub mod lexer;
pub mod parser;

use anyhow::{Context, Result};

/// Parse a TraceQL query string and convert it to SQL
///
/// This is the main entry point for TraceQL query processing.
///
/// # Arguments
///
/// * `traceql_query` - The TraceQL query string (e.g., `{ span.http.method = "GET" }`)
///
/// # Returns
///
/// A SQL query string that can be executed against the DataFusion context
///
/// # Errors
///
/// Returns an error if the TraceQL query cannot be parsed or converted
pub fn traceql_to_sql_string(traceql_query: &str) -> Result<String> {
    // Parse the TraceQL query
    let ast = parser::parse(traceql_query)
        .with_context(|| format!("Failed to parse TraceQL query: {}", traceql_query))?;

    // Convert to SQL
    let sql =
        converter::traceql_to_sql(&ast).with_context(|| "Failed to convert TraceQL to SQL")?;

    Ok(sql)
}

/// Detect if a query string is TraceQL or SQL
///
/// TraceQL queries start with '{' (after optional whitespace)
pub fn is_traceql_query(query: &str) -> bool {
    query.trim_start().starts_with('{')
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_traceql_query() {
        assert!(is_traceql_query("{ }"));
        assert!(is_traceql_query("  { span.http.method = \"GET\" }"));
        assert!(is_traceql_query("{ } | rate()"));

        assert!(!is_traceql_query("SELECT * FROM spans"));
        assert!(!is_traceql_query("  SELECT COUNT(*) FROM traces"));
    }

    #[test]
    fn test_end_to_end_simple() {
        let sql = traceql_to_sql_string("{ }").unwrap();
        // Should generate inline CTEs with unnest operations
        assert!(sql.contains("WITH unnest_resources"));
        assert!(sql.contains("FROM unnest_spans"));
    }

    #[test]
    fn test_end_to_end_with_filter() {
        let sql = traceql_to_sql_string(r#"{ span.http.method = "GET" }"#).unwrap();
        assert!(sql.contains("HttpMethod"));
        assert!(sql.contains("'GET'"));
    }

    #[test]
    fn test_end_to_end_with_pipeline() {
        let sql = traceql_to_sql_string("{ } | rate()").unwrap();
        assert!(sql.contains("COUNT(*)"));
    }
}
