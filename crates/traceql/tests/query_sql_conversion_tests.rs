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
    assert!(!query.is_empty());

    // Convert TraceQL to SQL
    let sql = traceql_to_sql_string(&query).expect("Conversion failed");

    assert_eq!(sql.trim(), strip_comments(expected).trim());
}
