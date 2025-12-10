/// Converter from TraceQL AST to DataFusion SQL
///
/// Translates TraceQL queries into SQL queries against the spans view.
use super::ast::*;
use std::fmt::Write;

/// Represents which level of the trace hierarchy a filter applies to
#[derive(Debug, Clone, Copy, PartialEq)]
enum FilterLevel {
    Trace,    // Applied at traces table level
    Resource, // Applied after ResourceSpans unnest
    Span,     // Applied after Spans unnest
}

/// Classified filter expressions for each hierarchy level
#[derive(Debug, Default)]
struct ClassifiedFilters {
    trace_filters: Vec<String>,
    resource_filters: Vec<String>,
    span_filters: Vec<String>,
}

impl ClassifiedFilters {
    fn new() -> Self {
        Self::default()
    }
}

/// Context for writing field paths
#[derive(Copy, Clone)]
enum FieldContext {
    Trace,
    Resource,
    Span,
}

/// Convert a TraceQL query to SQL
pub fn traceql_to_sql(query: &TraceQLQuery) -> Result<String, ConversionError> {
    let mut sql = String::new();

    // Check if we have a select operation (projection)
    let select_fields = query.pipeline.iter().find_map(|op| {
        if let PipelineOp::Select { fields } = op {
            Some(fields)
        } else {
            None
        }
    });

    // Convert the main query
    match &query.query {
        QueryExpr::SpanFilter(filter) => {
            write_span_filter_query(&mut sql, filter, select_fields.map(|v| v.as_slice()))?;
        }
        QueryExpr::Structural { parent, child } => {
            write_structural_query(&mut sql, parent, child)?;
        }
        QueryExpr::Union(filters) => {
            write_union_query(&mut sql, filters)?;
        }
    }

    // Handle pipeline operations (skip select as it's handled above)
    let non_select_ops: Vec<_> = query
        .pipeline
        .iter()
        .filter(|op| !matches!(op, PipelineOp::Select { .. }))
        .collect();

    if !non_select_ops.is_empty() {
        // Wrap the query in a CTE for aggregation
        let base_query = sql;
        sql = String::new();
        writeln!(&mut sql, "WITH base_spans AS (")?;
        writeln!(&mut sql, "{}", base_query)?;
        writeln!(&mut sql, ")")?;

        for (i, op) in non_select_ops.iter().enumerate() {
            if i == 0 {
                write_pipeline_op(&mut sql, op, "base_spans")?;
            } else {
                // For multiple pipeline ops, would need to chain them
                return Err(ConversionError::Unsupported(
                    "Multiple pipeline operations not yet supported".to_string(),
                ));
            }
        }

        // Add having condition if present
        if let Some(having) = &query.having {
            write!(&mut sql, " HAVING ")?;
            // Determine the aggregation column name based on the pipeline operation
            let agg_column = match non_select_ops.first() {
                Some(PipelineOp::Count { .. }) => "count",
                Some(PipelineOp::Rate { .. }) => "rate",
                Some(PipelineOp::Avg { .. }) => "avg",
                Some(PipelineOp::Sum { .. }) => "sum",
                Some(PipelineOp::Min { .. }) => "min",
                Some(PipelineOp::Max { .. }) => "max",
                _ => {
                    return Err(ConversionError::Unsupported(
                        "No aggregation pipeline operation".to_string(),
                    ))
                }
            };
            write!(&mut sql, "{} {} ", agg_column, having.op)?;
            write_value(&mut sql, &having.value)?;
        }
    }

    Ok(sql)
}

/// Helper to determine most specific level (Span > Resource > Trace)
fn most_specific_level(a: FilterLevel, b: FilterLevel) -> FilterLevel {
    match (a, b) {
        (FilterLevel::Span, _) | (_, FilterLevel::Span) => FilterLevel::Span,
        (FilterLevel::Resource, _) | (_, FilterLevel::Resource) => FilterLevel::Resource,
        _ => FilterLevel::Trace,
    }
}

/// Helper to determine least specific level (Trace < Resource < Span)
fn least_specific_level(a: FilterLevel, b: FilterLevel) -> FilterLevel {
    match (a, b) {
        (FilterLevel::Trace, _) | (_, FilterLevel::Trace) => FilterLevel::Trace,
        (FilterLevel::Resource, _) | (_, FilterLevel::Resource) => FilterLevel::Resource,
        _ => FilterLevel::Span,
    }
}

/// Determines which level a field reference belongs to
fn determine_field_level(field: &FieldRef) -> Result<FilterLevel, ConversionError> {
    match field.scope {
        FieldScope::Resource => Ok(FilterLevel::Resource),
        FieldScope::Span => Ok(FilterLevel::Span),
        FieldScope::Unscoped => Ok(FilterLevel::Span), // Default unscoped to span
        FieldScope::Intrinsic => {
            // Classify intrinsics by whether they're trace-level or span-level
            match field.name.as_str() {
                "rootServiceName" | "rootName" | "traceDuration" => Ok(FilterLevel::Trace),
                _ => Ok(FilterLevel::Span), // Most intrinsics are span-level
            }
        }
    }
}

/// Adds a filter SQL string to the appropriate level
fn add_filter_to_level(classified: &mut ClassifiedFilters, level: FilterLevel, sql: String) {
    match level {
        FilterLevel::Trace => classified.trace_filters.push(sql),
        FilterLevel::Resource => classified.resource_filters.push(sql),
        FilterLevel::Span => classified.span_filters.push(sql),
    }
}

/// Writes SQL for a filter expression at a specific context level
fn write_filter_expr(
    sql: &mut String,
    expr: &Expr,
    context: FieldContext,
) -> Result<(), ConversionError> {
    match expr {
        Expr::BinaryOp { left, op, right } => {
            write!(sql, "(")?;
            write_filter_expr(sql, left, context)?;
            let sql_op = match op {
                BinaryOperator::And => "AND",
                BinaryOperator::Or => "OR",
            };
            write!(sql, " {} ", sql_op)?;
            write_filter_expr(sql, right, context)?;
            write!(sql, ")")?;
        }
        Expr::UnaryOp { op, expr } => {
            let sql_op = match op {
                UnaryOperator::Not => "NOT",
            };
            write!(sql, "{} (", sql_op)?;
            write_filter_expr(sql, expr, context)?;
            write!(sql, ")")?;
        }
        Expr::Comparison { field, op, value } => {
            write_comparison_with_context(sql, field, op, value, context)?;
        }
    }
    Ok(())
}

/// Analyzes a filter expression and classifies predicates by hierarchy level
fn classify_filter_expression(expr: &Option<Expr>) -> Result<ClassifiedFilters, ConversionError> {
    let mut classified = ClassifiedFilters::new();

    if let Some(expr) = expr {
        classify_expr_recursive(expr, &mut classified)?;
    }

    Ok(classified)
}

/// Recursively walks expression tree and classifies leaf predicates
fn classify_expr_recursive(
    expr: &Expr,
    classified: &mut ClassifiedFilters,
) -> Result<FilterLevel, ConversionError> {
    match expr {
        Expr::BinaryOp { left, op, right } => {
            match op {
                BinaryOperator::And => {
                    // AND can split predicates across levels
                    let left_level = classify_expr_recursive(left, classified)?;
                    let right_level = classify_expr_recursive(right, classified)?;
                    Ok(most_specific_level(left_level, right_level))
                }
                BinaryOperator::Or => {
                    // OR must keep both sides at the same (lowest common) level
                    let left_level = classify_expr_recursive(left, classified)?;
                    let right_level = classify_expr_recursive(right, classified)?;
                    let common_level = least_specific_level(left_level, right_level);

                    // Generate SQL for the entire OR expression
                    let mut or_sql = String::new();
                    let context = match common_level {
                        FilterLevel::Trace => FieldContext::Trace,
                        FilterLevel::Resource => FieldContext::Resource,
                        FilterLevel::Span => FieldContext::Span,
                    };
                    write_filter_expr(&mut or_sql, expr, context)?;

                    add_filter_to_level(classified, common_level, or_sql);
                    Ok(common_level)
                }
            }
        }
        Expr::UnaryOp { expr: inner, .. } => {
            // For NOT, classify the inner expression first
            let level = classify_expr_recursive(inner, classified)?;

            // Generate SQL with NOT
            let mut not_sql = String::new();
            let context = match level {
                FilterLevel::Trace => FieldContext::Trace,
                FilterLevel::Resource => FieldContext::Resource,
                FilterLevel::Span => FieldContext::Span,
            };
            write_filter_expr(&mut not_sql, expr, context)?;

            add_filter_to_level(classified, level, not_sql);
            Ok(level)
        }
        Expr::Comparison { field, op, value } => {
            let level = determine_field_level(field)?;

            // Generate SQL for this comparison
            let mut comp_sql = String::new();
            let context = match level {
                FilterLevel::Trace => FieldContext::Trace,
                FilterLevel::Resource => FieldContext::Resource,
                FilterLevel::Span => FieldContext::Span,
            };
            write_comparison_with_context(&mut comp_sql, field, op, value, context)?;

            add_filter_to_level(classified, level, comp_sql);
            Ok(level)
        }
    }
}

/// Writes SQL path for a resource-scoped field in the nested structure
fn write_resource_field_path(sql: &mut String, field_name: &str) -> Result<(), ConversionError> {
    // Check dedicated resource columns first
    let dedicated_column = match field_name {
        "service.name" => Some("resource.\"Resource\".\"ServiceName\""),
        "cluster" => Some("resource.\"Resource\".\"Cluster\""),
        "namespace" => Some("resource.\"Resource\".\"Namespace\""),
        "pod" => Some("resource.\"Resource\".\"Pod\""),
        "container" => Some("resource.\"Resource\".\"Container\""),
        "k8s.cluster.name" => Some("resource.\"Resource\".\"K8sClusterName\""),
        "k8s.namespace.name" => Some("resource.\"Resource\".\"K8sNamespaceName\""),
        "k8s.pod.name" => Some("resource.\"Resource\".\"K8sPodName\""),
        "k8s.container.name" => Some("resource.\"Resource\".\"K8sContainerName\""),
        _ => None,
    };

    if let Some(column) = dedicated_column {
        sql.push_str(column);
    } else {
        // Generic attribute via attrs_to_map UDF
        sql.push_str("flatten(map_extract(attrs_to_map(resource.\"Resource\".\"Attrs\"), '");
        sql.push_str(field_name);
        sql.push_str("'))");
    }

    Ok(())
}

/// Writes SQL path for a span-scoped field in the nested structure
fn write_span_field_path(sql: &mut String, field_name: &str) -> Result<(), ConversionError> {
    // Check dedicated span columns first
    let dedicated_column = match field_name {
        "http.method" => Some("span.\"HttpMethod\""),
        "http.url" => Some("span.\"HttpUrl\""),
        "http.status_code" | "http.response_code" => Some("span.\"HttpStatusCode\""),
        _ => None,
    };

    if let Some(column) = dedicated_column {
        sql.push_str(column);
    } else {
        // Generic attribute via attrs_to_map UDF
        sql.push_str("flatten(map_extract(attrs_to_map(span.\"Attrs\"), '");
        sql.push_str(field_name);
        sql.push_str("'))");
    }

    Ok(())
}

/// Writes SQL path for a span intrinsic field in the nested structure
fn write_span_intrinsic_path(
    sql: &mut String,
    intrinsic_name: &str,
) -> Result<(), ConversionError> {
    let column = match intrinsic_name {
        "name" => "span.\"Name\"",
        "duration" => "span.\"DurationNano\"",
        "status" => "span.\"StatusCode\"",
        "kind" => "span.\"Kind\"",
        "spanID" => "span.\"SpanID\"",
        "parentSpanID" => "span.\"ParentSpanID\"",
        "traceID" => "\"TraceID\"", // Available at all levels
        _ => {
            return Err(ConversionError::Unsupported(format!(
                "Unsupported span intrinsic: {}",
                intrinsic_name
            )))
        }
    };

    sql.push_str(column);
    Ok(())
}

/// Writes SQL path for a trace-level field (used in trace WHERE clause)
fn write_trace_field_path(sql: &mut String, intrinsic_name: &str) -> Result<(), ConversionError> {
    let column = match intrinsic_name {
        "traceID" => "t.\"TraceID\"",
        "startTime" => "t.\"StartTimeUnixNano\"",
        "endTime" => "t.\"EndTimeUnixNano\"",
        "duration" => "t.\"DurationNano\"",
        "rootServiceName" => "t.\"RootServiceName\"",
        "rootName" => "t.\"RootSpanName\"",
        _ => {
            return Err(ConversionError::Unsupported(format!(
                "Unsupported trace intrinsic: {}",
                intrinsic_name
            )))
        }
    };

    sql.push_str(column);
    Ok(())
}

/// Write a comparison expression with context-appropriate field paths
fn write_comparison_with_context(
    sql: &mut String,
    field: &FieldRef,
    op: &ComparisonOperator,
    value: &Value,
    context: FieldContext,
) -> Result<(), ConversionError> {
    // Determine if this is a list attribute
    let is_list_attr = match field.scope {
        FieldScope::Span => !matches!(
            field.name.as_str(),
            "http.method" | "http.url" | "http.status_code" | "http.response_code"
        ),
        FieldScope::Resource => !matches!(
            field.name.as_str(),
            "service.name"
                | "cluster"
                | "namespace"
                | "pod"
                | "container"
                | "k8s.cluster.name"
                | "k8s.namespace.name"
                | "k8s.pod.name"
                | "k8s.container.name"
        ),
        FieldScope::Unscoped => true,
        FieldScope::Intrinsic => false,
    };

    // Write the field path based on context
    match field.scope {
        FieldScope::Resource => {
            write_resource_field_path(sql, &field.name)?;
        }
        FieldScope::Span | FieldScope::Unscoped => {
            write_span_field_path(sql, &field.name)?;
        }
        FieldScope::Intrinsic => match context {
            FieldContext::Trace => write_trace_field_path(sql, &field.name)?,
            _ => write_span_intrinsic_path(sql, &field.name)?,
        },
    }

    // Write operator and value with appropriate handling for list attributes
    match op {
        ComparisonOperator::Eq if is_list_attr => {
            let field_sql = sql.clone();
            sql.clear();
            write!(sql, "list_contains({}, ", field_sql)?;
            write_value(sql, value)?;
            write!(sql, ")")?;
        }
        ComparisonOperator::NotEq if is_list_attr => {
            let field_sql = sql.clone();
            sql.clear();
            write!(sql, "NOT list_contains({}, ", field_sql)?;
            write_value(sql, value)?;
            write!(sql, ")")?;
        }
        ComparisonOperator::Regex if is_list_attr => {
            let field_sql = sql.clone();
            sql.clear();
            write!(sql, "array_to_string({}, ',') ~ ", field_sql)?;
            write_value(sql, value)?;
        }
        ComparisonOperator::NotRegex if is_list_attr => {
            let field_sql = sql.clone();
            sql.clear();
            write!(sql, "array_to_string({}, ',') !~ ", field_sql)?;
            write_value(sql, value)?;
        }
        ComparisonOperator::Regex => {
            write!(sql, " ~ ")?;
            write_value(sql, value)?;
        }
        ComparisonOperator::NotRegex => {
            write!(sql, " !~ ")?;
            write_value(sql, value)?;
        }
        _ => {
            write!(sql, " {} ", op)?;
            write_value(sql, value)?;
        }
    }

    Ok(())
}

/// Generates inline CTE chain with filter pushdown
/// Returns the name of the final CTE to select from
fn write_inline_spans_view_with_pushdown(
    sql: &mut String,
    filters: &ClassifiedFilters,
    include_with_keyword: bool,
) -> Result<&'static str, ConversionError> {
    if include_with_keyword {
        sql.push_str("WITH ");
    }
    sql.push_str("unnest_resources AS (\n");
    sql.push_str("  SELECT t.\"TraceID\", UNNEST(t.rs) as resource\n");
    sql.push_str("  FROM traces t\n");

    // Apply trace-level filters
    if !filters.trace_filters.is_empty() {
        sql.push_str("  WHERE ");
        sql.push_str(&filters.trace_filters.join(" AND "));
        sql.push('\n');
    }
    sql.push(')');

    // Conditionally add resource filtering CTE
    let resource_source = if !filters.resource_filters.is_empty() {
        sql.push_str(",\nfiltered_resources AS (\n");
        sql.push_str("  SELECT * FROM unnest_resources\n");
        sql.push_str("  WHERE ");
        sql.push_str(&filters.resource_filters.join(" AND "));
        sql.push_str("\n)");
        "filtered_resources"
    } else {
        "unnest_resources"
    };

    // Unnest ScopeSpans
    sql.push_str(",\nunnest_scopespans AS (\n");
    sql.push_str("  SELECT \"TraceID\", resource, UNNEST(resource.ss) as scopespans\n");
    sql.push_str(&format!("  FROM {}\n", resource_source));
    sql.push(')');

    // Unnest Spans
    sql.push_str(",\nunnest_spans AS (\n");
    sql.push_str("  SELECT \"TraceID\", resource, UNNEST(scopespans.\"Spans\") as span\n");
    sql.push_str("  FROM unnest_scopespans\n");
    sql.push(')');

    // Add span-level filtering in a separate CTE if needed
    let source_cte = if !filters.span_filters.is_empty() {
        sql.push_str(",\nfiltered_spans AS (\n");
        sql.push_str("  SELECT * FROM unnest_spans\n");
        sql.push_str("  WHERE ");
        sql.push_str(&filters.span_filters.join(" AND "));
        sql.push_str("\n)");
        "filtered_spans"
    } else {
        "unnest_spans"
    };

    Ok(source_cte)
}

/// Writes the final SELECT with field mappings
fn write_final_projection(
    sql: &mut String,
    select_fields: Option<&[FieldRef]>,
    source: &str,
) -> Result<(), ConversionError> {
    sql.push_str("\nSELECT ");

    if let Some(fields) = select_fields {
        // Explicit field list from SELECT clause
        for (i, field) in fields.iter().enumerate() {
            if i > 0 {
                sql.push_str(", ");
            }
            write_projection_field(sql, field)?;
        }
    } else {
        // Default: select all span fields with proper aliases
        sql.push_str("\"TraceID\" AS \"TraceID\", ");
        sql.push_str("span.\"SpanID\" AS \"SpanID\", ");
        sql.push_str("span.\"Name\" AS \"Name\", ");
        sql.push_str("span.\"Kind\" AS \"Kind\", ");
        sql.push_str("span.\"ParentSpanID\" AS \"ParentSpanID\", ");
        sql.push_str("span.\"StartTimeUnixNano\" AS \"StartTimeUnixNano\", ");
        sql.push_str("span.\"DurationNano\" AS \"DurationNano\", ");
        sql.push_str("span.\"StatusCode\" AS \"StatusCode\", ");
        sql.push_str("span.\"HttpMethod\" AS \"HttpMethod\", ");
        sql.push_str("span.\"HttpUrl\" AS \"HttpUrl\", ");
        sql.push_str("span.\"HttpStatusCode\" AS \"HttpStatusCode\", ");
        // sql.push_str("attrs_to_map(span.\"Attrs\") AS \"Attrs\", ");
    }

    write!(sql, "\nFROM {}", source)?;

    Ok(())
}

/// Write a projection field with appropriate path
fn write_projection_field(sql: &mut String, field: &FieldRef) -> Result<(), ConversionError> {
    match field.scope {
        FieldScope::Resource => {
            write_resource_field_path(sql, &field.name)?;
        }
        FieldScope::Span | FieldScope::Unscoped => {
            write_span_field_path(sql, &field.name)?;
        }
        FieldScope::Intrinsic => {
            write_span_intrinsic_path(sql, &field.name)?;
        }
    }
    Ok(())
}

/// Write a simple span filter query
fn write_span_filter_query(
    sql: &mut String,
    filter: &SpanFilter,
    select_fields: Option<&[FieldRef]>,
) -> Result<(), ConversionError> {
    // Classify the filter expression
    let classified = classify_filter_expression(&filter.expr)?;

    // Generate inline view with pushdown
    let source = write_inline_spans_view_with_pushdown(sql, &classified, true)?;

    // Write final projection
    write_final_projection(sql, select_fields, source)?;

    Ok(())
}

/// Write a structural query (parent >> child)
fn write_structural_query(
    sql: &mut String,
    parent: &SpanFilter,
    child: &SpanFilter,
) -> Result<(), ConversionError> {
    // Classify parent and child filters separately
    let parent_classified = classify_filter_expression(&parent.expr)?;
    let child_classified = classify_filter_expression(&child.expr)?;

    // Generate parent spans CTE
    write_inline_spans_view_with_pushdown(sql, &parent_classified, true)?;
    sql.push_str(", parent_spans AS (\n");
    sql.push_str("  SELECT * FROM unnest_spans\n");
    sql.push_str(")\n");

    // Generate child spans CTE with prefixed names to avoid conflicts
    sql.push_str(", child_unnest_resources AS (\n");
    sql.push_str("  SELECT t.\"TraceID\", UNNEST(t.rs) as resource\n");
    sql.push_str("  FROM traces t\n");
    if !child_classified.trace_filters.is_empty() {
        sql.push_str("  WHERE ");
        sql.push_str(&child_classified.trace_filters.join(" AND "));
        sql.push('\n');
    }
    sql.push(')');

    let child_resource_source = if !child_classified.resource_filters.is_empty() {
        sql.push_str(",\nchild_filtered_resources AS (\n");
        sql.push_str("  SELECT * FROM child_unnest_resources\n");
        sql.push_str("  WHERE ");
        sql.push_str(&child_classified.resource_filters.join(" AND "));
        sql.push_str("\n)");
        "child_filtered_resources"
    } else {
        "child_unnest_resources"
    };

    sql.push_str(",\nchild_unnest_scopespans AS (\n");
    sql.push_str("  SELECT \"TraceID\", resource, UNNEST(resource.ss) as scopespans\n");
    sql.push_str(&format!("  FROM {}\n", child_resource_source));
    sql.push(')');

    sql.push_str(",\nchild_unnest_spans AS (\n");
    sql.push_str("  SELECT \"TraceID\", resource, UNNEST(scopespans.\"Spans\") as span\n");
    sql.push_str("  FROM child_unnest_scopespans\n");
    sql.push(')');

    // Add span-level filtering for child spans if needed
    let child_source = if !child_classified.span_filters.is_empty() {
        sql.push_str(",\nchild_filtered_spans AS (\n");
        sql.push_str("  SELECT * FROM child_unnest_spans\n");
        sql.push_str("  WHERE ");
        sql.push_str(&child_classified.span_filters.join(" AND "));
        sql.push_str("\n)");
        "child_filtered_spans"
    } else {
        "child_unnest_spans"
    };

    sql.push_str(",\nchild_spans AS (\n");
    sql.push_str("  SELECT \"TraceID\", ");
    sql.push_str("span.\"SpanID\", ");
    sql.push_str("span.\"Name\", ");
    sql.push_str("span.\"NestedSetLeft\", ");
    sql.push_str("span.\"NestedSetRight\" ");
    writeln!(sql, "FROM {}", child_source)?;
    sql.push_str(")\n");

    // Join parent and child on nested set relationships
    sql.push_str("SELECT child_spans.* FROM parent_spans\n");
    sql.push_str("INNER JOIN child_spans\n");
    sql.push_str("  ON child_spans.\"TraceID\" = parent_spans.\"TraceID\"\n");
    sql.push_str("  AND child_spans.\"NestedSetLeft\" > parent_spans.\"NestedSetLeft\"\n");
    sql.push_str("  AND child_spans.\"NestedSetRight\" < parent_spans.\"NestedSetRight\"\n");

    Ok(())
}

/// Write a union query ({ } || { } || ...)
fn write_union_query(sql: &mut String, filters: &[SpanFilter]) -> Result<(), ConversionError> {
    for (i, filter) in filters.iter().enumerate() {
        if i > 0 {
            writeln!(sql, "\nUNION\n")?;
        }

        // Each branch gets its own classification and pushdown
        let classified = classify_filter_expression(&filter.expr)?;
        let source = write_inline_spans_view_with_pushdown(sql, &classified, true)?;
        write_final_projection(sql, None, source)?;
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
                    Ok(format!(
                        "flatten(map_extract({}\"Attrs\", '{}'))",
                        table_prefix, field.name
                    ))
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
                    Ok(format!(
                        "flatten(map_extract({}\"ResourceAttrs\", '{}'))",
                        table_prefix, field.name
                    ))
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
                "rootServiceName" => {
                    // Trace-level field: service name of the root span
                    // When using root join, table_prefix will be "root."
                    Ok(format!("{}\"ResourceServiceName\"", table_prefix))
                }
                "rootName" => {
                    // Trace-level field: name of the root span
                    // When using root join, table_prefix will be "root."
                    Ok(format!("{}\"Name\"", table_prefix))
                }
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
            // For now, just try as span attribute using same logic as scoped span attributes
            Ok(format!(
                "flatten(map_extract({}\"Attrs\", '{}'))",
                table_prefix, field.name
            ))
        }
    }
}

/// Maps a TraceQL field name to the SQL column name used in the base_spans projection
fn map_field_to_column_name(field_name: &str) -> &str {
    match field_name {
        "status" => "\"StatusCode\"",
        "name" => "\"Name\"",
        "duration" => "\"DurationNano\"",
        "kind" => "\"Kind\"",
        _ => field_name,
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
        PipelineOp::Rate { group_by } => {
            // Rate calculates spans per second using 5-minute buckets
            // Use date_bin to create time buckets
            write!(sql, "SELECT ")?;

            // Add time bucket column (cast UInt64 to Int64 for to_timestamp_nanos)
            write!(
                sql,
                "date_bin(INTERVAL '5 minutes', to_timestamp_nanos(CAST(\"StartTimeUnixNano\" AS BIGINT)), TIMESTAMP '1970-01-01 00:00:00') as time_bucket"
            )?;

            // Add group by fields
            for field in group_by {
                let column_name = map_field_to_column_name(field);
                write!(sql, ", {}", column_name)?;
            }

            // Add rate calculation (count per 5 minutes)
            write!(sql, ", COUNT(*) as rate")?;

            writeln!(sql, " FROM {}", source)?;

            // Add GROUP BY clause
            write!(sql, "GROUP BY time_bucket")?;
            for field in group_by {
                let column_name = map_field_to_column_name(field);
                write!(sql, ", {}", column_name)?;
            }
            writeln!(sql)?;

            // Order by time bucket and group by fields
            write!(sql, "ORDER BY time_bucket")?;
            for field in group_by {
                let column_name = map_field_to_column_name(field);
                write!(sql, ", {}", column_name)?;
            }
        }
        PipelineOp::Count { group_by } => {
            write!(sql, "SELECT ")?;

            // Add group by fields
            if !group_by.is_empty() {
                for (i, field) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(field);
                    write!(sql, "{}", column_name)?;
                }
                write!(sql, ", ")?;
            }

            writeln!(sql, "COUNT(*) as count FROM {}", source)?;

            // Add GROUP BY clause if needed
            if !group_by.is_empty() {
                write!(sql, "GROUP BY ")?;
                for (i, field) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(field);
                    write!(sql, "{}", column_name)?;
                }
                writeln!(sql)?;
            }
        }
        PipelineOp::Avg { field, group_by } => {
            write!(sql, "SELECT ")?;

            // Add group by fields
            if !group_by.is_empty() {
                for f in group_by {
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}, ", column_name)?;
                }
            }

            let field_column = map_field_to_column_name(field);
            writeln!(sql, "AVG({}) as avg FROM {}", field_column, source)?;

            // Add GROUP BY clause if needed
            if !group_by.is_empty() {
                write!(sql, "GROUP BY ")?;
                for (i, f) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}", column_name)?;
                }
                writeln!(sql)?;
            }
        }
        PipelineOp::Sum { field, group_by } => {
            write!(sql, "SELECT ")?;

            // Add group by fields
            if !group_by.is_empty() {
                for f in group_by {
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}, ", column_name)?;
                }
            }

            let field_column = map_field_to_column_name(field);
            writeln!(sql, "SUM({}) as sum FROM {}", field_column, source)?;

            // Add GROUP BY clause if needed
            if !group_by.is_empty() {
                write!(sql, "GROUP BY ")?;
                for (i, f) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}", column_name)?;
                }
                writeln!(sql)?;
            }
        }
        PipelineOp::Min { field, group_by } => {
            write!(sql, "SELECT ")?;

            // Add group by fields
            if !group_by.is_empty() {
                for f in group_by {
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}, ", column_name)?;
                }
            }

            let field_column = map_field_to_column_name(field);
            writeln!(sql, "MIN({}) as min FROM {}", field_column, source)?;

            // Add GROUP BY clause if needed
            if !group_by.is_empty() {
                write!(sql, "GROUP BY ")?;
                for (i, f) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}", column_name)?;
                }
                writeln!(sql)?;
            }
        }
        PipelineOp::Max { field, group_by } => {
            write!(sql, "SELECT ")?;

            // Add group by fields
            if !group_by.is_empty() {
                for f in group_by {
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}, ", column_name)?;
                }
            }

            let field_column = map_field_to_column_name(field);
            writeln!(sql, "MAX({}) as max FROM {}", field_column, source)?;

            // Add GROUP BY clause if needed
            if !group_by.is_empty() {
                write!(sql, "GROUP BY ")?;
                for (i, f) in group_by.iter().enumerate() {
                    if i > 0 {
                        write!(sql, ", ")?;
                    }
                    let column_name = map_field_to_column_name(f);
                    write!(sql, "{}", column_name)?;
                }
                writeln!(sql)?;
            }
        }
        PipelineOp::Select { .. } => {
            // Select operations are handled in traceql_to_sql and should not reach here
            return Err(ConversionError::Unsupported(
                "Select operations should be handled before aggregation pipeline".to_string(),
            ));
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
        // Should generate inline CTEs with unnest operations
        assert!(sql.contains("WITH unnest_resources"));
        assert!(sql.contains("FROM unnest_spans"));
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
        assert!(sql.contains("AND"));
    }
}
