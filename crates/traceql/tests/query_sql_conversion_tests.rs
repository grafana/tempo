use test_each_file::test_each_file;
use traceql::traceql_to_sql_string;

/// Strip comments starting with # from the query
fn strip_comments(content: &str) -> String {
    content
        .lines()
        .filter(|line| !line.trim_start().starts_with('#'))
        .collect::<Vec<_>>()
        .join("\n")
        .trim()
        .to_string()
}

test_each_file! { for ["tql", "sql"] in "./crates/traceql/queries" => test_sql_conversion }

fn test_sql_conversion(content: &str) {
    // Strip comments
    let query = strip_comments(content);

    // Skip empty queries
    if query.is_empty() {
        return;
    }

    // Convert TraceQL to SQL
    let result = traceql_to_sql_string(&query);

    // Assert that conversion succeeded
    assert!(
        result.is_ok(),
        "Failed to convert query to SQL: {}\nQuery: {}",
        result.unwrap_err(),
        query
    );

    let sql = result.unwrap();

    // Verify the SQL is not empty
    assert!(
        !sql.is_empty(),
        "Generated SQL is empty for query: {}",
        query
    );

    // Verify the SQL contains SELECT statement
    assert!(
        sql.to_uppercase().contains("SELECT"),
        "Generated SQL does not contain SELECT statement: {}\nQuery: {}",
        sql,
        query
    );
}
