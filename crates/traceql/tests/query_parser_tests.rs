use test_each_file::test_each_file;
use traceql::parser;

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

test_each_file! { for ["tql"] in "./crates/traceql/queries" => test_parse_query }

fn test_parse_query([input]: [&str; 1]) {
    // Strip comments
    let query = strip_comments(input);

    // Parse the query
    let result = parser::parse(&query);

    // Assert that parsing succeeded
    assert!(
        result.is_ok(),
        "Failed to parse query: {}\nQuery: {}",
        result.unwrap_err(),
        query
    );
}
