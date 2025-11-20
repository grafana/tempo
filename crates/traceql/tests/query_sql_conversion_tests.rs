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

fn test_sql_conversion([input, expected]: [&str; 2]) {
    // Strip comments
    let query = strip_comments(input);

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

    assert_eq!(sql, strip_comments(expected));
}
