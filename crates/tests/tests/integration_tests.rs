use test_each_file::test_each_file;
use tests::{execute_query, get_test_data_path, setup_context_with_block};
use traceql::traceql_to_sql_string;

const TEST_BLOCK_ID: &str = "b27b0e53-66a0-4505-afd6-434ae3cd4a10";
const TEST_TENANT_ID: &str = "single-tenant";

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

test_each_file! { #[tokio::test] async for ["tql"] in "./crates/traceql/queries" => test_traceql_query_execution }

async fn test_traceql_query_execution([input]: [&str; 1]) {
    // Strip comments
    let query = strip_comments(input);

    // Get the path to the test data file (for verification)
    let data_path = get_test_data_path(TEST_BLOCK_ID);

    // Verify test data exists
    assert!(
        data_path.exists(),
        "Test data file not found at {:?}",
        data_path
    );

    // Setup context with test block using object store
    let ctx = setup_context_with_block(TEST_BLOCK_ID, TEST_TENANT_ID)
        .await
        .expect("Failed to setup context");

    // Convert TraceQL to SQL
    let sql = traceql_to_sql_string(&query)
        .unwrap_or_else(|e| panic!("Failed to convert TraceQL to SQL: {}\nQuery: {}", e, query));

    // Execute the query
    let rows = execute_query(&ctx, &sql)
        .await
        .unwrap_or_else(|e| panic!("Failed to execute query: {}\nSQL: {}", e, sql));

    assert!(rows > 0, "Query '{}' returned 0 rows", query);
}

#[tokio::test]
async fn test_basic_queries() {
    // Get the path to the test data file (for verification)
    let data_path = get_test_data_path(TEST_BLOCK_ID);

    // Verify test data exists
    assert!(
        data_path.exists(),
        "Test data file not found at {:?}",
        data_path
    );

    // Setup context with test block using object store
    let ctx = setup_context_with_block(TEST_BLOCK_ID, TEST_TENANT_ID)
        .await
        .expect("Failed to setup context");

    // Test basic queries that should definitely work
    let test_cases = vec![
        ("name_match", "{ name = `distributor.ConsumeTraces` }"),
        ("status_match", "{ status = ok }"),
        (
            "simple_and",
            "{ name = `distributor.ConsumeTraces` && status = ok }",
        ),
    ];

    for (name, traceql) in test_cases {
        let sql = traceql_to_sql_string(traceql)
            .unwrap_or_else(|e| panic!("{}: Failed to convert TraceQL to SQL: {}", name, e));

        let rows = execute_query(&ctx, &sql)
            .await
            .unwrap_or_else(|e| panic!("{}: Failed to execute query: {}", name, e));

        println!("{}: {} rows", name, rows);
    }
}

#[tokio::test]
async fn test_no_match_queries() {
    // Get the path to the test data file (for verification)
    let data_path = get_test_data_path(TEST_BLOCK_ID);

    // Verify test data exists
    assert!(
        data_path.exists(),
        "Test data file not found at {:?}",
        data_path
    );

    // Setup context with test block using object store
    let ctx = setup_context_with_block(TEST_BLOCK_ID, TEST_TENANT_ID)
        .await
        .expect("Failed to setup context");

    // Test queries that should return no results
    let test_cases = vec![
        ("name_no_match", "{ name = `does-not-exist-xyz123` }"),
        (
            "status_no_match",
            "{ status = error && name = `does-not-exist-xyz123` }",
        ),
    ];

    for (name, traceql) in test_cases {
        let sql = traceql_to_sql_string(traceql)
            .unwrap_or_else(|e| panic!("{}: Failed to convert TraceQL to SQL: {}", name, e));

        let rows = execute_query(&ctx, &sql)
            .await
            .unwrap_or_else(|e| panic!("{}: Failed to execute query: {}", name, e));

        println!("{}: {} rows", name, rows);
        assert_eq!(rows, 0, "{} should return 0 rows", name);
    }
}
