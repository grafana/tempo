# Multi-Level SQL Generation with Filter Pushdown - Implementation Plan

## Problem Statement

The current TraceQL-to-SQL converter generates queries like:
```sql
SELECT * FROM spans WHERE <filters>
```

The `spans` view performs three UNNEST operations on the entire dataset before applying filters:
1. `traces` → UNNEST `ResourceSpans`
2. → UNNEST `ScopeSpans`
3. → UNNEST `Spans`

This means **every trace is fully unpacked** regardless of which traces/resources match the filter criteria, causing massive performance overhead.

## Solution Approach

Generate inline SQL with filters pushed down to appropriate UNNEST levels:

```sql
WITH unnest_resources AS (
    SELECT "TraceID", UNNEST(t.rs) as resource
    FROM traces t
    WHERE <trace-level-filters>  -- ⭐ Filter traces early
),
filtered_resources AS (
    SELECT * FROM unnest_resources
    WHERE <resource-level-filters>  -- ⭐ Filter resources before span unnesting
),
unnest_scopespans AS (
    SELECT "TraceID", resource, UNNEST(resource.ss) as scopespans
    FROM filtered_resources
),
unnest_spans AS (
    SELECT "TraceID", resource, UNNEST(scopespans."Spans") as span
    FROM unnest_scopespans
    WHERE <span-level-filters>  -- ⭐ Filter spans after unnesting
)
SELECT <projection> FROM unnest_spans
```

---

## Phase 1: Predicate Classification

**File**: `crates/traceql/src/converter.rs`

### 1.1 Add Classification Types

Add after existing imports (~line 7):

```rust
/// Represents which level of the trace hierarchy a filter applies to
#[derive(Debug, Clone, PartialEq)]
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

    fn is_empty(&self) -> bool {
        self.trace_filters.is_empty()
            && self.resource_filters.is_empty()
            && self.span_filters.is_empty()
    }
}
```

### 1.2 Implement Predicate Classifier

Add new function to classify filter expressions:

```rust
/// Analyzes a filter expression and classifies predicates by hierarchy level
fn classify_filter_expression(
    expr: &Option<Expr>,
    sql_buffer: &mut String,
) -> Result<ClassifiedFilters, ConversionError> {
    let mut classified = ClassifiedFilters::new();

    if let Some(expr) = expr {
        classify_expr_recursive(expr, &mut classified, sql_buffer)?;
    }

    Ok(classified)
}

/// Recursively walks expression tree and classifies leaf predicates
fn classify_expr_recursive(
    expr: &Expr,
    classified: &mut ClassifiedFilters,
    sql_buffer: &mut String,
) -> Result<FilterLevel, ConversionError> {
    match expr {
        // Logical operators
        Expr::And(left, right) => {
            let left_level = classify_expr_recursive(left, classified, sql_buffer)?;
            let right_level = classify_expr_recursive(right, classified, sql_buffer)?;

            // AND can split predicates across levels
            // Each side goes to its appropriate level
            Ok(most_specific_level(left_level, right_level))
        }

        Expr::Or(left, right) => {
            // OR must keep both sides at the same (lowest common) level
            let left_level = classify_expr_recursive(left, classified, sql_buffer)?;
            let right_level = classify_expr_recursive(right, classified, sql_buffer)?;
            let common_level = least_specific_level(left_level, right_level);

            // Generate SQL for the entire OR expression
            let mut or_sql = String::new();
            or_sql.push('(');
            write_filter_expr(&mut or_sql, left)?;
            or_sql.push_str(" OR ");
            write_filter_expr(&mut or_sql, right)?;
            or_sql.push(')');

            // Add to appropriate level
            add_filter_to_level(classified, common_level, or_sql);
            Ok(common_level)
        }

        Expr::Not(inner) => {
            let level = classify_expr_recursive(inner, classified, sql_buffer)?;

            // Generate SQL with NOT
            let mut not_sql = String::new();
            not_sql.push_str("NOT (");
            write_filter_expr(&mut not_sql, inner)?;
            not_sql.push(')');

            add_filter_to_level(classified, level, not_sql);
            Ok(level)
        }

        // Leaf predicates (comparisons)
        Expr::Comparison { left, op, right } => {
            let level = determine_comparison_level(left)?;

            // Generate SQL for this comparison
            let mut comp_sql = String::new();
            write_comparison_expr(&mut comp_sql, left, op, right)?;

            add_filter_to_level(classified, level, comp_sql);
            Ok(level)
        }

        _ => Err(ConversionError::UnsupportedExpression(
            format!("Cannot classify expression: {:?}", expr)
        ))
    }
}

/// Determines which level a field reference belongs to
fn determine_comparison_level(field_ref: &FieldRef) -> Result<FilterLevel, ConversionError> {
    match field_ref {
        FieldRef::Intrinsic(intrinsic) => {
            match intrinsic.name.as_str() {
                // Trace-level intrinsics
                "rootServiceName" | "rootName" | "traceDuration" => {
                    Ok(FilterLevel::Trace)
                }
                // Span-level intrinsics
                "name" | "duration" | "status" | "kind" | "spanID" => {
                    Ok(FilterLevel::Span)
                }
                _ => Ok(FilterLevel::Span) // Default to span
            }
        }

        FieldRef::Attribute(attr) => {
            match attr.scope {
                Some(AttributeScope::Resource) => Ok(FilterLevel::Resource),
                Some(AttributeScope::Span) => Ok(FilterLevel::Span),
                None => Ok(FilterLevel::Span), // Unscoped defaults to span
            }
        }
    }
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

/// Adds a filter SQL string to the appropriate level
fn add_filter_to_level(classified: &mut ClassifiedFilters, level: FilterLevel, sql: String) {
    match level {
        FilterLevel::Trace => classified.trace_filters.push(sql),
        FilterLevel::Resource => classified.resource_filters.push(sql),
        FilterLevel::Span => classified.span_filters.push(sql),
    }
}
```

---

## Phase 2: Field Path Rewriting

The flattened `spans` view uses columns like `"ResourceServiceName"`. The nested structure needs paths like `resource."Resource"."ServiceName"`.

### 2.1 Add Field Path Generators

```rust
/// Writes SQL path for a resource-scoped field
fn write_resource_field_path(
    sql: &mut String,
    field_name: &str,
) -> Result<(), ConversionError> {
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

/// Writes SQL path for a span-scoped field
fn write_span_field_path(
    sql: &mut String,
    field_name: &str,
) -> Result<(), ConversionError> {
    // Check dedicated span columns first
    let dedicated_column = match field_name {
        "http.method" => Some("span.\"HttpMethod\""),
        "http.url" => Some("span.\"HttpUrl\""),
        "http.status_code" => Some("span.\"HttpStatusCode\""),
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

/// Writes SQL path for a span intrinsic field
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
        _ => return Err(ConversionError::UnsupportedIntrinsic(intrinsic_name.to_string())),
    };

    sql.push_str(column);
    Ok(())
}

/// Writes SQL path for a trace-level field (used in trace WHERE clause)
fn write_trace_field_path(
    sql: &mut String,
    intrinsic_name: &str,
) -> Result<(), ConversionError> {
    let column = match intrinsic_name {
        "traceID" => "t.\"TraceID\"",
        "startTime" => "t.\"StartTimeUnixNano\"",
        "endTime" => "t.\"EndTimeUnixNano\"",
        "duration" => "t.\"DurationNano\"",
        "rootServiceName" => "t.\"RootServiceName\"",
        "rootName" => "t.\"RootSpanName\"",
        _ => return Err(ConversionError::UnsupportedIntrinsic(intrinsic_name.to_string())),
    };

    sql.push_str(column);
    Ok(())
}
```

### 2.2 Update Comparison Writer

Modify the existing `write_comparison_expr` to use appropriate path based on context:

```rust
fn write_comparison_expr(
    sql: &mut String,
    field: &FieldRef,
    op: &ComparisonOp,
    value: &Value,
    context: FieldContext, // NEW: track where we are
) -> Result<(), ConversionError> {
    match field {
        FieldRef::Attribute(attr) => {
            match (context, attr.scope) {
                (FieldContext::Resource, Some(AttributeScope::Resource)) => {
                    write_resource_field_path(sql, &attr.name)?;
                }
                (FieldContext::Span, Some(AttributeScope::Span)) => {
                    write_span_field_path(sql, &attr.name)?;
                }
                _ => {
                    // Fallback for unscoped or mismatched context
                    write_span_field_path(sql, &attr.name)?;
                }
            }
        }
        FieldRef::Intrinsic(intrinsic) => {
            match context {
                FieldContext::Trace => write_trace_field_path(sql, &intrinsic.name)?,
                _ => write_span_intrinsic_path(sql, &intrinsic.name)?,
            }
        }
    }

    // Write operator and value
    write_comparison_op(sql, op)?;
    write_value(sql, value)?;

    Ok(())
}

#[derive(Copy, Clone)]
enum FieldContext {
    Trace,
    Resource,
    Span,
}
```

---

## Phase 3: Inline View Generator

### 3.1 Main View Generation Function

```rust
/// Generates inline CTE chain with filter pushdown
fn write_inline_spans_view_with_pushdown(
    sql: &mut String,
    filters: &ClassifiedFilters,
) -> Result<(), ConversionError> {
    sql.push_str("WITH unnest_resources AS (\n");
    sql.push_str("  SELECT t.\"TraceID\", UNNEST(t.rs) as resource\n");
    sql.push_str("  FROM traces t\n");

    // Apply trace-level filters
    if !filters.trace_filters.is_empty() {
        sql.push_str("  WHERE ");
        sql.push_str(&filters.trace_filters.join(" AND "));
        sql.push_str("\n");
    }
    sql.push_str(")");

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
    sql.push_str(")");

    // Unnest Spans with span-level filters
    sql.push_str(",\nunnest_spans AS (\n");
    sql.push_str("  SELECT \"TraceID\", resource, UNNEST(scopespans.\"Spans\") as span\n");
    sql.push_str("  FROM unnest_scopespans\n");

    if !filters.span_filters.is_empty() {
        sql.push_str("  WHERE ");
        sql.push_str(&filters.span_filters.join(" AND "));
        sql.push_str("\n");
    }
    sql.push_str(")\n");

    Ok(())
}
```

### 3.2 Final Projection

```rust
/// Writes the final SELECT with field mappings
fn write_final_projection(
    sql: &mut String,
    select_fields: Option<&[FieldRef]>,
) -> Result<(), ConversionError> {
    sql.push_str("SELECT ");

    if let Some(fields) = select_fields {
        // Explicit field list from SELECT clause
        for (i, field) in fields.iter().enumerate() {
            if i > 0 {
                sql.push_str(", ");
            }
            write_projection_field(sql, field)?;
        }
    } else {
        // Default: select all span fields
        sql.push_str("\"TraceID\", ");
        sql.push_str("span.\"SpanID\", ");
        sql.push_str("span.\"Name\", ");
        sql.push_str("span.\"Kind\", ");
        sql.push_str("span.\"StartTimeUnixNano\", ");
        sql.push_str("span.\"DurationNano\", ");
        sql.push_str("span.\"StatusCode\", ");
        sql.push_str("span.\"HttpMethod\", ");
        sql.push_str("span.\"HttpUrl\", ");
        sql.push_str("span.\"HttpStatusCode\", ");
        sql.push_str("attrs_to_map(span.\"Attrs\") AS \"Attrs\", ");
        sql.push_str("resource.\"Resource\".\"ServiceName\" AS \"ResourceServiceName\"");
        // Add other resource fields as needed
    }

    sql.push_str("\nFROM unnest_spans");

    Ok(())
}

fn write_projection_field(
    sql: &mut String,
    field: &FieldRef,
) -> Result<(), ConversionError> {
    match field {
        FieldRef::Attribute(attr) => {
            if attr.scope == Some(AttributeScope::Resource) {
                write_resource_field_path(sql, &attr.name)?;
            } else {
                write_span_field_path(sql, &attr.name)?;
            }
        }
        FieldRef::Intrinsic(intrinsic) => {
            write_span_intrinsic_path(sql, &intrinsic.name)?;
        }
    }
    Ok(())
}
```

---

## Phase 4: Update Query Writers

### 4.1 Modify `write_span_filter_query()`

Replace the existing implementation (~line 78-133):

```rust
fn write_span_filter_query(
    sql: &mut String,
    filter: &SpanFilter,
    select_fields: Option<&[FieldRef]>,
) -> Result<(), ConversionError> {
    // Classify the filter expression
    let mut temp_sql = String::new();
    let classified = classify_filter_expression(&filter.expr, &mut temp_sql)?;

    // Generate inline view with pushdown
    write_inline_spans_view_with_pushdown(sql, &classified)?;

    // Write final projection
    write_final_projection(sql, select_fields)?;

    Ok(())
}
```

### 4.2 Modify `write_union_query()`

Each union branch gets its own optimized query:

```rust
fn write_union_query(
    sql: &mut String,
    union: &Union,
    select_fields: Option<&[FieldRef]>,
) -> Result<(), ConversionError> {
    for (i, branch) in union.branches.iter().enumerate() {
        if i > 0 {
            sql.push_str("\nUNION\n");
        }

        // Each branch gets its own classification and pushdown
        let mut temp_sql = String::new();
        let classified = classify_filter_expression(&branch.expr, &mut temp_sql)?;

        write_inline_spans_view_with_pushdown(sql, &classified)?;
        write_final_projection(sql, select_fields)?;
    }

    Ok(())
}
```

### 4.3 Modify `write_structural_query()`

Parent-child joins need careful handling:

```rust
fn write_structural_query(
    sql: &mut String,
    structural: &Structural,
    select_fields: Option<&[FieldRef]>,
) -> Result<(), ConversionError> {
    // Classify parent and child filters separately
    let mut temp_sql = String::new();
    let parent_classified = classify_filter_expression(&structural.parent.expr, &mut temp_sql)?;
    let child_classified = classify_filter_expression(&structural.child.expr, &mut temp_sql)?;

    // Generate parent spans CTE
    sql.push_str("WITH ");
    write_inline_spans_view_with_pushdown(sql, &parent_classified)?;
    sql.push_str(", parent_spans AS (\n");
    sql.push_str("  SELECT * FROM unnest_spans\n");
    sql.push_str(")\n");

    // Generate child spans CTE (reuse unnest names with prefix)
    sql.push_str(", child_unnest_resources AS (\n");
    sql.push_str("  SELECT t.\"TraceID\", UNNEST(t.rs) as resource FROM traces t\n");
    // ... similar to write_inline_spans_view_with_pushdown but prefixed
    sql.push_str("), child_spans AS (\n");
    sql.push_str("  SELECT * FROM child_unnest_spans\n");
    sql.push_str(")\n");

    // Join parent and child on nested set relationships
    sql.push_str("SELECT child_spans.* FROM parent_spans\n");
    sql.push_str("INNER JOIN child_spans\n");
    sql.push_str("  ON child_spans.\"TraceID\" = parent_spans.\"TraceID\"\n");
    sql.push_str("  AND child_spans.\"NestedSetLeft\" > parent_spans.\"NestedSetLeft\"\n");
    sql.push_str("  AND child_spans.\"NestedSetRight\" < parent_spans.\"NestedSetRight\"\n");

    Ok(())
}
```

### 4.4 Handle Aggregations

Aggregations already use a CTE wrapper, just update the inner query:

```rust
fn write_aggregation_query(
    sql: &mut String,
    agg: &Aggregation,
    filter: &SpanFilter,
) -> Result<(), ConversionError> {
    sql.push_str("WITH base_spans AS (\n");

    // Use pushdown for the inner query
    let mut temp_sql = String::new();
    let classified = classify_filter_expression(&filter.expr, &mut temp_sql)?;
    write_inline_spans_view_with_pushdown(sql, &classified)?;
    write_final_projection(sql, None)?;

    sql.push_str("\n)\n");

    // Aggregation on top
    sql.push_str("SELECT COUNT(*) as count FROM base_spans");

    Ok(())
}
```

---

## Phase 5: Testing Strategy

### 5.1 Unit Tests

Add to `crates/traceql/src/converter.rs`:

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_classify_resource_filter() {
        let filter = parse_traceql("{ resource.service.name = 'api' }").unwrap();
        let classified = classify_filter_expression(&filter.expr, &mut String::new()).unwrap();

        assert_eq!(classified.resource_filters.len(), 1);
        assert_eq!(classified.span_filters.len(), 0);
        assert_eq!(classified.trace_filters.len(), 0);
    }

    #[test]
    fn test_classify_mixed_and_filter() {
        let filter = parse_traceql("{ resource.service.name = 'api' && span.http.method = 'GET' }").unwrap();
        let classified = classify_filter_expression(&filter.expr, &mut String::new()).unwrap();

        // AND allows splitting across levels
        assert_eq!(classified.resource_filters.len(), 1);
        assert_eq!(classified.span_filters.len(), 1);
    }

    #[test]
    fn test_classify_mixed_or_filter() {
        let filter = parse_traceql("{ resource.service.name = 'api' || span.http.method = 'GET' }").unwrap();
        let classified = classify_filter_expression(&filter.expr, &mut String::new()).unwrap();

        // OR must stay at span level (lowest common level)
        assert_eq!(classified.span_filters.len(), 1);
        assert_eq!(classified.resource_filters.len(), 0);
    }

    #[test]
    fn test_inline_view_generation() {
        let filters = ClassifiedFilters {
            trace_filters: vec![],
            resource_filters: vec!["resource.\"Resource\".\"ServiceName\" = 'api'".to_string()],
            span_filters: vec!["span.\"HttpMethod\" = 'GET'".to_string()],
        };

        let mut sql = String::new();
        write_inline_spans_view_with_pushdown(&mut sql, &filters).unwrap();

        // Verify CTE structure
        assert!(sql.contains("WITH unnest_resources AS"));
        assert!(sql.contains("filtered_resources AS"));
        assert!(sql.contains("WHERE resource.\"Resource\".\"ServiceName\" = 'api'"));
        assert!(sql.contains("WHERE span.\"HttpMethod\" = 'GET'"));
    }
}
```

### 5.2 Integration Tests

Update existing tests in `crates/tests/benches/traceql_queries.rs`:

```rust
#[test]
fn test_resource_filter_optimization() {
    let query = "{ resource.service.name = 'loki-querier' }";
    let sql = traceql_to_sql(query).unwrap();

    // Verify resource filter is pushed down
    assert!(sql.contains("filtered_resources"));
    assert!(!sql.contains("FROM spans")); // Should not use view

    // Run and compare results with Go implementation
    let results = execute_query(&sql).unwrap();
    assert_eq!(results.len(), expected_count);
}
```

### 5.3 Performance Benchmarks

Add benchmarks to verify improvements:

```rust
#[bench]
fn bench_resource_filter_with_pushdown(b: &mut Bencher) {
    let query = "{ resource.service.name = 'loki-querier' }";
    b.iter(|| {
        execute_traceql_query(query)
    });
}

#[bench]
fn bench_mixed_filter_with_pushdown(b: &mut Bencher) {
    let query = "{ resource.service.name = 'loki-querier' && span.http.method = 'GET' }";
    b.iter(|| {
        execute_traceql_query(query)
    });
}
```

---

## Phase 6: Rollout Plan

Since this is a PoC, full rewrite is acceptable:

1. **Backup current converter.rs** (for reference)
2. **Implement Phase 1** (classification) - verify with unit tests
3. **Implement Phase 2** (field paths) - verify paths are correct
4. **Implement Phase 3** (view generation) - verify SQL structure
5. **Implement Phase 4** (query writers) - update all query types
6. **Run integration tests** - ensure correctness vs Go reference
7. **Run benchmarks** - verify performance improvements
8. **Document changes** - update any relevant docs

---

## Expected Outcomes

### Performance Improvements

- **Resource-only queries**: 5-20x faster (skip 2 unnest levels)
- **Mixed resource+span queries**: 2-10x faster (skip 1 unnest level partially)
- **Union queries**: Shared filtering eliminates redundant work
- **Memory usage**: Scales with filtered data, not total dataset

### SQL Query Size

- Generated SQL will be longer (inline CTEs vs view reference)
- But logically equivalent and more performant
- DataFusion can optimize the inline query directly

### Compatibility

- Same public API: `traceql_to_sql()` signature unchanged
- All existing tests should pass with same results
- Performance improvements are transparent to callers

---

## Implementation Checklist

- [ ] Phase 1: Add classification types and implement predicate classifier
- [ ] Phase 2: Implement field path rewriting for nested structure
- [ ] Phase 3: Implement inline view generator with filter injection
- [ ] Phase 4: Update all query writers (span_filter, union, structural, aggregation)
- [ ] Phase 5: Add unit tests for classification logic
- [ ] Phase 5: Add integration tests for correctness
- [ ] Phase 5: Add benchmarks for performance verification
- [ ] Phase 6: Run full test suite and verify against Go implementation
- [ ] Document changes and optimization approach

---

## Key Implementation Notes

### attrs_to_map UDF Placement

The `attrs_to_map` UDF should be applied at the appropriate level:
- For resource filters: Apply in resource filter expressions
- For span filters: Apply in span filter expressions
- In final projection: Apply to both if needed for SELECT

### Handling List Fields

Remember that attributes are stored as `List<String>` after attrs_to_map conversion, so comparisons need:
```sql
list_contains(flatten(map_extract(attrs_to_map(...), 'key')), 'value')
```

### Dedicated Column Optimization

Always prefer dedicated columns over generic attributes:
- Resource: ServiceName, Cluster, Namespace, Pod, K8s fields
- Span: HttpMethod, HttpUrl, HttpStatusCode, Name, Duration, Status

These are indexed and faster than map lookups.

### Error Handling

Be careful with:
- Missing attributes (NULL handling)
- Type mismatches in comparisons
- Empty list handling in list_contains

---

This plan provides complete implementation details for the multi-level filter pushdown optimization in the TraceQL converter.
