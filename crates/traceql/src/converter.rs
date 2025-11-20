/// Converter from TraceQL AST to DataFusion SQL
///
/// Translates TraceQL queries into SQL queries against the spans view.
use super::ast::*;
use std::fmt::Write;

/// Convert a TraceQL query to SQL
pub fn traceql_to_sql(query: &TraceQLQuery) -> Result<String, ConversionError> {
    let mut sql = String::new();

    // Convert the main query
    match &query.query {
        QueryExpr::SpanFilter(filter) => {
            write_span_filter_query(&mut sql, filter)?;
        }
        QueryExpr::Structural { parent, child } => {
            write_structural_query(&mut sql, parent, child)?;
        }
    }

    // Handle pipeline operations
    if !query.pipeline.is_empty() {
        // Wrap the query in a CTE for aggregation
        let base_query = sql;
        sql = String::new();
        writeln!(&mut sql, "WITH base_spans AS (")?;
        writeln!(&mut sql, "{}", base_query)?;
        writeln!(&mut sql, ")")?;

        for (i, op) in query.pipeline.iter().enumerate() {
            if i == 0 {
                write_pipeline_op(&mut sql, op, "base_spans")?;
            } else {
                // For multiple pipeline ops, would need to chain them
                return Err(ConversionError::Unsupported(
                    "Multiple pipeline operations not yet supported".to_string(),
                ));
            }
        }
    }

    Ok(sql)
}

/// Write a simple span filter query
fn write_span_filter_query(sql: &mut String, filter: &SpanFilter) -> Result<(), ConversionError> {
    writeln!(sql, "SELECT * FROM spans")?;

    if let Some(expr) = &filter.expr {
        write!(sql, "WHERE ")?;
        write_expr(sql, expr)?;
    }

    Ok(())
}

/// Write a structural query (parent >> child)
fn write_structural_query(
    sql: &mut String,
    parent: &SpanFilter,
    child: &SpanFilter,
) -> Result<(), ConversionError> {
    // Use nested set model to find parent-child relationships
    writeln!(sql, "SELECT child.* FROM spans parent")?;
    writeln!(sql, "INNER JOIN spans child")?;
    writeln!(
        sql,
        "  ON child.\"NestedSetLeft\" > parent.\"NestedSetLeft\""
    )?;
    writeln!(
        sql,
        "  AND child.\"NestedSetRight\" < parent.\"NestedSetRight\""
    )?;
    writeln!(sql, "  AND child.\"TraceID\" = parent.\"TraceID\"")?;

    let mut conditions = Vec::new();

    // Add parent filter
    if let Some(expr) = &parent.expr {
        let mut parent_filter = String::new();
        write_expr_with_prefix(&mut parent_filter, expr, "parent")?;
        conditions.push(parent_filter);
    }

    // Add child filter
    if let Some(expr) = &child.expr {
        let mut child_filter = String::new();
        write_expr_with_prefix(&mut child_filter, expr, "child")?;
        conditions.push(child_filter);
    }

    if !conditions.is_empty() {
        writeln!(sql, "WHERE {}", conditions.join(" AND "))?;
    }

    Ok(())
}

/// Write an expression
fn write_expr(sql: &mut String, expr: &Expr) -> Result<(), ConversionError> {
    write_expr_with_prefix(sql, expr, "")
}

/// Write an expression with an optional table prefix
fn write_expr_with_prefix(
    sql: &mut String,
    expr: &Expr,
    prefix: &str,
) -> Result<(), ConversionError> {
    match expr {
        Expr::BinaryOp { left, op, right } => {
            write!(sql, "(")?;
            write_expr_with_prefix(sql, left, prefix)?;
            write!(sql, " {} ", op)?;
            write_expr_with_prefix(sql, right, prefix)?;
            write!(sql, ")")?;
        }
        Expr::UnaryOp { op, expr } => {
            write!(sql, "{}", op)?;
            write!(sql, "(")?;
            write_expr_with_prefix(sql, expr, prefix)?;
            write!(sql, ")")?;
        }
        Expr::Comparison { field, op, value } => {
            write_comparison(sql, field, op, value, prefix)?;
        }
    }
    Ok(())
}

/// Write a comparison expression
fn write_comparison(
    sql: &mut String,
    field: &FieldRef,
    op: &ComparisonOperator,
    value: &Value,
    prefix: &str,
) -> Result<(), ConversionError> {
    let table_prefix = if prefix.is_empty() {
        String::new()
    } else {
        format!("{}.", prefix)
    };

    // Check if this is a span or resource attribute that returns a list
    let is_list_attr = match field.scope {
        FieldScope::Span => {
            // Span attributes except dedicated columns
            !matches!(field.name.as_str(), "http.method" | "http.url" | "http.status_code" | "http.response_code")
        }
        FieldScope::Resource => {
            // Resource attributes except dedicated columns
            !matches!(field.name.as_str(),
                "service.name" | "cluster" | "namespace" | "pod" | "container" |
                "k8s.cluster.name" | "k8s.namespace.name" | "k8s.pod.name" | "k8s.container.name"
            )
        }
        _ => false
    };

    // Map TraceQL field to SQL column
    let sql_field = field_to_sql(field, &table_prefix)?;

    // Write the comparison
    match op {
        ComparisonOperator::Eq if is_list_attr => {
            // Use list_contains for span and resource attributes
            write!(sql, "list_contains({}, ", sql_field)?;
            write_value(sql, value)?;
            write!(sql, ")")?;
        }
        ComparisonOperator::NotEq if is_list_attr => {
            // Use NOT list_contains for span and resource attributes
            write!(sql, "NOT list_contains({}, ", sql_field)?;
            write_value(sql, value)?;
            write!(sql, ")")?;
        }
        ComparisonOperator::Regex if is_list_attr => {
            // TODO: Use proper list aggregation instead of array_to_string
            // This workaround concatenates all list elements which may produce false matches
            // Should use a list iteration function when available in DataFusion
            write!(sql, "array_to_string({}, ',') ~ ", sql_field)?;
            write_value(sql, value)?;
        }
        ComparisonOperator::NotRegex if is_list_attr => {
            // TODO: Use proper list aggregation instead of array_to_string
            // This workaround concatenates all list elements which may produce false matches
            // Should use a list iteration function when available in DataFusion
            write!(sql, "array_to_string({}, ',') !~ ", sql_field)?;
            write_value(sql, value)?;
        }
        ComparisonOperator::Regex => {
            // Use DataFusion regex match
            write!(sql, "{} ~ ", sql_field)?;
            write_value(sql, value)?;
        }
        ComparisonOperator::NotRegex => {
            write!(sql, "{} !~ ", sql_field)?;
            write_value(sql, value)?;
        }
        _ => {
            write!(sql, "{} {} ", sql_field, op)?;
            write_value(sql, value)?;
        }
    }

    Ok(())
}

/// Convert a TraceQL field reference to SQL
fn field_to_sql(field: &FieldRef, table_prefix: &str) -> Result<String, ConversionError> {
    match &field.scope {
        FieldScope::Span => {
            // span.* attributes go to Attrs map or dedicated columns
            match field.name.as_str() {
                "http.method" => Ok(format!("{}\"HttpMethod\"", table_prefix)),
                "http.url" => Ok(format!("{}\"HttpUrl\"", table_prefix)),
                "http.status_code" | "http.response_code" => {
                    Ok(format!("{}\"HttpStatusCode\"", table_prefix))
                }
                _ => {
                    // Generic attribute access via map
                    // Use map_extract to get the list for this key
                    Ok(format!("flatten(map_extract({}\"Attrs\", '{}'))", table_prefix, field.name))
                }
            }
        }
        FieldScope::Resource => {
            // resource.* attributes go to ResourceAttrs map or dedicated columns
            match field.name.as_str() {
                "service.name" => Ok(format!("{}\"ResourceServiceName\"", table_prefix)),
                "cluster" => Ok(format!("{}\"ResourceCluster\"", table_prefix)),
                "namespace" => Ok(format!("{}\"ResourceNamespace\"", table_prefix)),
                "pod" => Ok(format!("{}\"ResourcePod\"", table_prefix)),
                "container" => Ok(format!("{}\"ResourceContainer\"", table_prefix)),
                "k8s.cluster.name" => Ok(format!("{}\"ResourceK8sClusterName\"", table_prefix)),
                "k8s.namespace.name" => Ok(format!("{}\"ResourceK8sNamespaceName\"", table_prefix)),
                "k8s.pod.name" => Ok(format!("{}\"ResourceK8sPodName\"", table_prefix)),
                "k8s.container.name" => Ok(format!("{}\"ResourceK8sContainerName\"", table_prefix)),
                _ => {
                    // Generic resource attribute access via ResourceAttrs map
                    // Use map_extract to get the list for this key
                    Ok(format!("flatten(map_extract({}\"ResourceAttrs\", '{}'))", table_prefix, field.name))
                }
            }
        }
        FieldScope::Intrinsic => {
            // Intrinsic fields map directly to columns
            match field.name.as_str() {
                "name" => Ok(format!("{}\"Name\"", table_prefix)),
                "duration" => Ok(format!("{}\"DurationNano\"", table_prefix)),
                "status" => Ok(format!("{}\"StatusCode\"", table_prefix)),
                "kind" => Ok(format!("{}\"Kind\"", table_prefix)),
                "nestedSetParent" => {
                    // Special case: nestedSetParent is calculated from nested set model
                    // A span's parent count is based on how many ancestors it has
                    // For now, we'll use a simplified version
                    Err(ConversionError::Unsupported(
                        "nestedSetParent intrinsic not yet fully supported".to_string(),
                    ))
                }
                _ => {
                    // Unknown intrinsic, try as-is
                    Ok(format!("{}\"{}\"", table_prefix, field.name))
                }
            }
        }
        FieldScope::Unscoped => {
            // Unscoped field .* - try both span and resource attributes
            // For now, just try as span attribute
            Ok(format!("{}\"Attrs\"['{}']", table_prefix, field.name))
        }
    }
}

/// Write a value to SQL
fn write_value(sql: &mut String, value: &Value) -> Result<(), ConversionError> {
    match value {
        Value::String(s) => {
            // Escape single quotes in SQL strings
            let escaped = s.replace('\'', "''");
            write!(sql, "'{}'", escaped)?;
        }
        Value::Integer(i) => {
            write!(sql, "{}", i)?;
        }
        Value::Float(f) => {
            write!(sql, "{}", f)?;
        }
        Value::Bool(b) => {
            write!(sql, "{}", b)?;
        }
        Value::Duration(d) => {
            // Convert duration to nanoseconds for comparison with DurationNano
            write!(sql, "{}", d.to_nanos())?;
        }
        Value::Status(s) => {
            // Map status to StatusCode integer values (OTLP spec)
            let code = match s {
                Status::Unset => 0,
                Status::Ok => 1,
                Status::Error => 2,
            };
            write!(sql, "{}", code)?;
        }
        Value::SpanKind(k) => {
            // Map span kind to Kind integer values (OTLP spec)
            let kind_code = match k {
                SpanKind::Unspecified => 0,
                SpanKind::Internal => 1,
                SpanKind::Server => 2,
                SpanKind::Client => 3,
                SpanKind::Producer => 4,
                SpanKind::Consumer => 5,
            };
            write!(sql, "{}", kind_code)?;
        }
    }
    Ok(())
}

/// Write a pipeline operation
fn write_pipeline_op(
    sql: &mut String,
    op: &PipelineOp,
    source: &str,
) -> Result<(), ConversionError> {
    match op {
        PipelineOp::Rate => {
            // Rate calculates spans per second
            // This is a simplified version - real rate would need time windows
            writeln!(sql, "SELECT COUNT(*) as rate FROM {}", source)?;
        }
        PipelineOp::Count => {
            writeln!(sql, "SELECT COUNT(*) as count FROM {}", source)?;
        }
        PipelineOp::Avg { field } => {
            writeln!(sql, "SELECT AVG({}) as avg FROM {}", field, source)?;
        }
        PipelineOp::Sum { field } => {
            writeln!(sql, "SELECT SUM({}) as sum FROM {}", field, source)?;
        }
        PipelineOp::Min { field } => {
            writeln!(sql, "SELECT MIN({}) as min FROM {}", field, source)?;
        }
        PipelineOp::Max { field } => {
            writeln!(sql, "SELECT MAX({}) as max FROM {}", field, source)?;
        }
    }
    Ok(())
}

/// Conversion errors
#[derive(Debug)]
pub enum ConversionError {
    Unsupported(String),
    FormatError(std::fmt::Error),
}

impl From<std::fmt::Error> for ConversionError {
    fn from(e: std::fmt::Error) -> Self {
        ConversionError::FormatError(e)
    }
}

impl std::fmt::Display for ConversionError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ConversionError::Unsupported(msg) => write!(f, "Unsupported feature: {}", msg),
            ConversionError::FormatError(e) => write!(f, "Format error: {}", e),
        }
    }
}

impl std::error::Error for ConversionError {}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser::parse;

    #[test]
    fn test_empty_filter() {
        let query = parse("{ }").unwrap();
        let sql = traceql_to_sql(&query).unwrap();
        assert!(sql.contains("SELECT * FROM spans"));
    }

    #[test]
    fn test_span_attribute() {
        let query = parse(r#"{ span.http.method = "GET" }"#).unwrap();
        let sql = traceql_to_sql(&query).unwrap();
        assert!(sql.contains("HttpMethod"));
        assert!(sql.contains("GET"));
    }

    #[test]
    fn test_duration() {
        let query = parse("{ duration > 100ms }").unwrap();
        let sql = traceql_to_sql(&query).unwrap();
        assert!(sql.contains("DurationNano"));
        assert!(sql.contains("100000000")); // 100ms in nanos
    }

    #[test]
    fn test_and_operation() {
        let query =
            parse(r#"{ span.http.method = "POST" && span.http.status_code = 500 }"#).unwrap();
        let sql = traceql_to_sql(&query).unwrap();
        assert!(sql.contains("HttpMethod"));
        assert!(sql.contains("HttpStatusCode"));
        assert!(sql.contains("&&"));
    }
}
