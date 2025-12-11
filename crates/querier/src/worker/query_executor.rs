use datafusion::arrow::array::{RecordBatch, StringArray, UInt64Array};
use datafusion::execution::context::SessionContext;
use provider::register_local_tempo_table;
use std::path::PathBuf;

use crate::error::{QuerierError, Result};
use crate::tempopb::{SearchMetrics, SearchResponse, TraceSearchMetadata};

/// Query executor that uses DataFusion and VParquet4Reader to execute queries
#[derive(Debug, Clone)]
pub struct QueryExecutor {
    data_path: PathBuf,
}

/// Parameters for a search query
#[derive(Debug, Clone)]
pub struct SearchParams {
    pub query: Option<String>,
    pub limit: Option<u32>,
    pub start: Option<u32>,
    pub end: Option<u32>,
}

impl SearchParams {
    /// Parse search parameters from URL query string
    pub fn from_url(url: &url::Url) -> Self {
        let query = url
            .query_pairs()
            .find(|(k, _)| k == "q" || k == "query")
            .map(|(_, v)| v.to_string());

        let limit = url
            .query_pairs()
            .find(|(k, _)| k == "limit")
            .and_then(|(_, v)| v.parse().ok());

        let start = url
            .query_pairs()
            .find(|(k, _)| k == "start")
            .and_then(|(_, v)| v.parse().ok());

        let end = url
            .query_pairs()
            .find(|(k, _)| k == "end")
            .and_then(|(_, v)| v.parse().ok());

        Self {
            query,
            limit,
            start,
            end,
        }
    }
}

impl QueryExecutor {
    /// Create a new QueryExecutor
    pub fn new(data_path: PathBuf) -> Self {
        Self { data_path }
    }

    /// Execute a search query and return SearchResponse
    pub async fn search(&self, params: &SearchParams) -> Result<SearchResponse> {
        tracing::info!(
            query = ?params.query,
            limit = ?params.limit,
            "Executing search query"
        );

        // Create a new DataFusion context
        let ctx = SessionContext::new();

        // Register the spans table
        register_local_tempo_table(&ctx, self.data_path.to_string_lossy().to_string())
            .await
            .map_err(|e| QuerierError::QueryExecution(format!("Failed to register table: {}", e)))?;

        // Build SQL query
        let sql = self.build_sql_query(params)?;

        tracing::debug!(sql = %sql, "Executing SQL query");

        // Execute query
        let df = ctx
            .sql(&sql)
            .await
            .map_err(|e| QuerierError::QueryExecution(format!("SQL execution failed: {}", e)))?;

        let results = df
            .collect()
            .await
            .map_err(|e| QuerierError::QueryExecution(format!("Failed to collect results: {}", e)))?;

        // Convert to SearchResponse
        let response = self.convert_to_search_response(results)?;

        tracing::info!(
            traces_found = response.traces.len(),
            "Search query completed"
        );

        Ok(response)
    }

    /// Build SQL query from search parameters
    fn build_sql_query(&self, params: &SearchParams) -> Result<String> {
        let mut conditions = Vec::new();

        // Add query condition (simple span name matching for now)
        if let Some(ref query) = params.query {
            // For now, treat query as a simple name filter
            // TODO: Full TraceQL parsing
            let name = query.trim();
            if !name.is_empty() {
                conditions.push(format!("name = '{}'", name.replace('\'', "''")));
            }
        }

        // Add time range conditions
        if let Some(start) = params.start {
            let start_ns = (start as u64) * 1_000_000_000;
            conditions.push(format!("start_time_unix_nano >= {}", start_ns));
        }

        if let Some(end) = params.end {
            let end_ns = (end as u64) * 1_000_000_000;
            conditions.push(format!("start_time_unix_nano <= {}", end_ns));
        }

        // Build WHERE clause
        let where_clause = if conditions.is_empty() {
            String::new()
        } else {
            format!("WHERE {}", conditions.join(" AND "))
        };

        // Build LIMIT clause
        let limit_clause = if let Some(limit) = params.limit {
            format!("LIMIT {}", limit)
        } else {
            "LIMIT 100".to_string()
        };

        Ok(format!(
            "SELECT trace_id, name, start_time_unix_nano, duration_nano FROM spans {} {}",
            where_clause, limit_clause
        ))
    }

    /// Convert DataFusion results to SearchResponse
    fn convert_to_search_response(&self, batches: Vec<RecordBatch>) -> Result<SearchResponse> {
        let mut traces = Vec::new();
        let mut inspected_traces = 0u32;

        for batch in batches {
            let num_rows = batch.num_rows();
            if num_rows == 0 {
                continue;
            }

            // Extract columns
            let trace_id_col = batch
                .column_by_name("trace_id")
                .ok_or_else(|| QuerierError::QueryExecution("Missing trace_id column".to_string()))?;
            let name_col = batch
                .column_by_name("name")
                .ok_or_else(|| QuerierError::QueryExecution("Missing name column".to_string()))?;
            let start_time_col = batch
                .column_by_name("start_time_unix_nano")
                .ok_or_else(|| QuerierError::QueryExecution("Missing start_time_unix_nano column".to_string()))?;
            let duration_col = batch
                .column_by_name("duration_nano")
                .ok_or_else(|| QuerierError::QueryExecution("Missing duration_nano column".to_string()))?;

            // Cast to concrete types
            let trace_ids = trace_id_col
                .as_any()
                .downcast_ref::<StringArray>()
                .ok_or_else(|| QuerierError::QueryExecution("Invalid trace_id type".to_string()))?;
            let names = name_col
                .as_any()
                .downcast_ref::<StringArray>()
                .ok_or_else(|| QuerierError::QueryExecution("Invalid name type".to_string()))?;
            let start_times = start_time_col
                .as_any()
                .downcast_ref::<UInt64Array>()
                .ok_or_else(|| QuerierError::QueryExecution("Invalid start_time type".to_string()))?;
            let durations = duration_col
                .as_any()
                .downcast_ref::<UInt64Array>()
                .ok_or_else(|| QuerierError::QueryExecution("Invalid duration type".to_string()))?;

            // Build trace metadata
            for i in 0..num_rows {
                let trace_id = trace_ids.value(i).to_string();
                let root_trace_name = names.value(i).to_string();
                let start_time_unix_nano = start_times.value(i);
                let duration_nano = durations.value(i);
                let duration_ms = (duration_nano / 1_000_000) as u32;

                let trace_metadata = TraceSearchMetadata {
                    trace_id: trace_id.clone(),
                    root_service_name: String::new(), // TODO: Extract from resource attributes
                    root_trace_name,
                    start_time_unix_nano,
                    duration_ms,
                    span_set: None,
                    span_sets: vec![],
                    service_stats: Default::default(),
                };

                traces.push(trace_metadata);
                inspected_traces += 1;
            }
        }

        Ok(SearchResponse {
            traces,
            metrics: Some(SearchMetrics {
                inspected_traces,
                inspected_bytes: 0, // TODO: Track bytes read
                total_blocks: 1,
                completed_jobs: 1,
                total_jobs: 1,
                total_block_bytes: 0,
                inspected_spans: inspected_traces as u64,
            }),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_search_params_from_url() {
        let url = url::Url::parse("http://localhost/api/search?query=my-span&limit=10").unwrap();
        let params = SearchParams::from_url(&url);

        assert_eq!(params.query, Some("my-span".to_string()));
        assert_eq!(params.limit, Some(10));
        assert_eq!(params.start, None);
        assert_eq!(params.end, None);
    }

    #[test]
    fn test_search_params_empty() {
        let url = url::Url::parse("http://localhost/api/search").unwrap();
        let params = SearchParams::from_url(&url);

        assert_eq!(params.query, None);
        assert_eq!(params.limit, None);
    }

    #[test]
    fn test_build_sql_query_with_name() {
        let executor = QueryExecutor::new(PathBuf::from("test.parquet"));
        let params = SearchParams {
            query: Some("test-span".to_string()),
            limit: Some(50),
            start: None,
            end: None,
        };

        let sql = executor.build_sql_query(&params).unwrap();
        assert!(sql.contains("WHERE name = 'test-span'"));
        assert!(sql.contains("LIMIT 50"));
    }

    #[test]
    fn test_build_sql_query_with_time_range() {
        let executor = QueryExecutor::new(PathBuf::from("test.parquet"));
        let params = SearchParams {
            query: None,
            limit: None,
            start: Some(1234567890),
            end: Some(1234567900),
        };

        let sql = executor.build_sql_query(&params).unwrap();
        assert!(sql.contains("start_time_unix_nano >= 1234567890000000000"));
        assert!(sql.contains("start_time_unix_nano <= 1234567900000000000"));
    }

    #[test]
    fn test_build_sql_query_default_limit() {
        let executor = QueryExecutor::new(PathBuf::from("test.parquet"));
        let params = SearchParams {
            query: None,
            limit: None,
            start: None,
            end: None,
        };

        let sql = executor.build_sql_query(&params).unwrap();
        assert!(sql.contains("LIMIT 100"));
    }
}
