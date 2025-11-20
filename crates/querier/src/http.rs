use axum::{
    extract::{Path, Query, State},
    http::{header, HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing, Json, Router,
};
use serde::Deserialize;
use std::sync::Arc;
use thiserror::Error;

use crate::QuerierService;

/// HTTP handler struct that holds the shared querier service
#[derive(Clone)]
pub struct HttpHandler {
    querier: Arc<QuerierService>,
}

impl HttpHandler {
    pub fn new(querier: Arc<QuerierService>) -> Self {
        Self { querier }
    }

    /// Create the HTTP router with all endpoints
    pub fn router(self) -> Router {
        Router::new()
            // Trace by ID endpoints (v1 and v2)
            .route(
                "/querier/api/traces/:trace_id",
                routing::get(Self::trace_by_id_handler).post(Self::trace_by_id_handler),
            )
            .route(
                "/querier/api/v2/traces/:trace_id",
                routing::get(Self::trace_by_id_handler_v2).post(Self::trace_by_id_handler_v2),
            )
            // Search endpoints
            .route(
                "/querier/api/search",
                routing::get(Self::search_handler).post(Self::search_handler),
            )
            // Search tags endpoints (v1 and v2)
            .route(
                "/querier/api/search/tags",
                routing::get(Self::search_tags_handler).post(Self::search_tags_handler),
            )
            .route(
                "/querier/api/v2/search/tags",
                routing::get(Self::search_tags_v2_handler).post(Self::search_tags_v2_handler),
            )
            // Search tag values endpoints (v1 and v2)
            .route(
                "/querier/api/search/tag/:tag_name/values",
                routing::get(Self::search_tag_values_handler).post(Self::search_tag_values_handler),
            )
            .route(
                "/querier/api/v2/search/tag/:tag_name/values",
                routing::get(Self::search_tag_values_v2_handler)
                    .post(Self::search_tag_values_v2_handler),
            )
            // Metrics endpoints
            .route(
                "/querier/api/metrics/summary",
                routing::get(Self::span_metrics_summary_handler)
                    .post(Self::span_metrics_summary_handler),
            )
            .route(
                "/querier/api/metrics/query_range",
                routing::get(Self::query_range_handler).post(Self::query_range_handler),
            )
            .with_state(self)
    }

    /// Handler for GET/POST /querier/api/traces/{traceID}
    async fn trace_by_id_handler(
        State(_handler): State<Self>,
        Path(trace_id): Path<String>,
        Query(params): Query<TraceByIdParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            trace_id = %trace_id,
            params = ?params,
            "Processing trace_by_id request (v1)"
        );

        // TODO: Implement trace lookup logic
        // 1. Parse and validate trace ID
        // 2. Create TraceByIdRequest protobuf message
        // 3. Call querier service
        // 4. Format response based on Accept header

        Err(HttpError::NotImplemented(
            "trace_by_id endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/v2/traces/{traceID}
    async fn trace_by_id_handler_v2(
        State(_handler): State<Self>,
        Path(trace_id): Path<String>,
        Query(params): Query<TraceByIdParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            trace_id = %trace_id,
            params = ?params,
            "Processing trace_by_id request (v2)"
        );

        // TODO: Implement trace lookup logic with v2 format (protobuf support)
        // This version should support both JSON and Protobuf responses

        Err(HttpError::NotImplemented(
            "trace_by_id_v2 endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/search
    async fn search_handler(
        State(_handler): State<Self>,
        Query(params): Query<SearchParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            params = ?params,
            "Processing search request"
        );

        // TODO: Implement search logic
        // 1. Validate search parameters
        // 2. Create SearchRequest protobuf message
        // 3. Call querier service
        // 4. Format and return search results

        Err(HttpError::NotImplemented(
            "search endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/search/tags
    async fn search_tags_handler(
        State(_handler): State<Self>,
        Query(params): Query<SearchTagsParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            params = ?params,
            "Processing search_tags request (v1)"
        );

        // TODO: Implement search tags logic
        // 1. Create SearchTagsRequest protobuf message
        // 2. Call querier service
        // 3. Return list of searchable tags

        Err(HttpError::NotImplemented(
            "search_tags endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/v2/search/tags
    async fn search_tags_v2_handler(
        State(_handler): State<Self>,
        Query(params): Query<SearchTagsParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            params = ?params,
            "Processing search_tags request (v2)"
        );

        // TODO: Implement search tags v2 logic
        // Returns tags with additional metadata in v2 format

        Err(HttpError::NotImplemented(
            "search_tags_v2 endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/search/tag/{tagName}/values
    async fn search_tag_values_handler(
        State(_handler): State<Self>,
        Path(tag_name): Path<String>,
        Query(params): Query<SearchTagValuesParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            tag_name = %tag_name,
            params = ?params,
            "Processing search_tag_values request (v1)"
        );

        // TODO: Implement search tag values logic
        // 1. Create SearchTagValuesRequest protobuf message
        // 2. Call querier service
        // 3. Return list of values for the specified tag

        Err(HttpError::NotImplemented(
            "search_tag_values endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/v2/search/tag/{tagName}/values
    async fn search_tag_values_v2_handler(
        State(_handler): State<Self>,
        Path(tag_name): Path<String>,
        Query(params): Query<SearchTagValuesParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            tag_name = %tag_name,
            params = ?params,
            "Processing search_tag_values request (v2)"
        );

        // TODO: Implement search tag values v2 logic
        // Returns tag values with additional metadata in v2 format

        Err(HttpError::NotImplemented(
            "search_tag_values_v2 endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/metrics/summary
    async fn span_metrics_summary_handler(
        State(_handler): State<Self>,
        Query(params): Query<SpanMetricsSummaryParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            params = ?params,
            "Processing span_metrics_summary request"
        );

        // TODO: Implement span metrics summary logic
        // 1. Parse and validate metrics query parameters
        // 2. Call metrics-generator service
        // 3. Return aggregated metrics summary

        Err(HttpError::NotImplemented(
            "span_metrics_summary endpoint not yet implemented".to_string(),
        ))
    }

    /// Handler for GET/POST /querier/api/metrics/query_range
    async fn query_range_handler(
        State(_handler): State<Self>,
        Query(params): Query<QueryRangeParams>,
        headers: HeaderMap,
    ) -> Result<Response, HttpError> {
        let tenant_id = extract_tenant_id(&headers)?;

        tracing::info!(
            tenant_id = %tenant_id,
            params = ?params,
            "Processing query_range request"
        );

        // TODO: Implement metrics query range logic
        // 1. Validate query parameters (query, start, end, step)
        // 2. Parse TraceQL metrics query
        // 3. Call metrics-generator service
        // 4. Return time series data

        Err(HttpError::NotImplemented(
            "query_range endpoint not yet implemented".to_string(),
        ))
    }
}

// ============================================================================
// Request Parameter Structs
// ============================================================================

/// Query parameters for trace lookup
#[derive(Debug, Deserialize)]
pub struct TraceByIdParams {
    #[serde(default)]
    pub start: Option<i64>,
    #[serde(default)]
    pub end: Option<i64>,
    #[serde(default = "default_query_mode")]
    pub mode: String,
    #[serde(default)]
    pub allow_partial: bool,
}

/// Query parameters for search
#[derive(Debug, Deserialize)]
pub struct SearchParams {
    pub tags: Option<String>,
    pub min_duration: Option<u32>,
    pub max_duration: Option<u32>,
    pub limit: Option<u32>,
    pub start: Option<i64>,
    pub end: Option<i64>,
    pub q: Option<String>, // TraceQL query
}

/// Query parameters for search tags
#[derive(Debug, Deserialize)]
pub struct SearchTagsParams {
    pub start: Option<i64>,
    pub end: Option<i64>,
}

/// Query parameters for search tag values
#[derive(Debug, Deserialize)]
pub struct SearchTagValuesParams {
    pub start: Option<i64>,
    pub end: Option<i64>,
    pub q: Option<String>, // TraceQL filter query
}

/// Query parameters for span metrics summary
#[derive(Debug, Deserialize)]
pub struct SpanMetricsSummaryParams {
    pub q: String,
    pub start: Option<i64>,
    pub end: Option<i64>,
    pub groupby: Option<String>,
}

/// Query parameters for metrics query range
#[derive(Debug, Deserialize)]
pub struct QueryRangeParams {
    pub query: String,
    pub start: i64,
    pub end: i64,
    pub step: Option<f64>,
}

fn default_query_mode() -> String {
    "all".to_string()
}

// ============================================================================
// Error Handling
// ============================================================================

#[derive(Debug, Error)]
pub enum HttpError {
    #[error("Missing tenant ID header (x-scope-orgid)")]
    MissingTenantId,

    #[error("Invalid tenant ID format")]
    InvalidTenantId,

    #[error("Not implemented: {0}")]
    NotImplemented(String),

    #[error("Invalid request: {0}")]
    InvalidRequest(String),

    #[error("Internal error: {0}")]
    Internal(String),
}

impl IntoResponse for HttpError {
    fn into_response(self) -> Response {
        let (status, message) = match &self {
            HttpError::MissingTenantId | HttpError::InvalidTenantId => {
                (StatusCode::UNAUTHORIZED, self.to_string())
            }
            HttpError::NotImplemented(_) => (StatusCode::NOT_IMPLEMENTED, self.to_string()),
            HttpError::InvalidRequest(_) => (StatusCode::BAD_REQUEST, self.to_string()),
            HttpError::Internal(_) => (StatusCode::INTERNAL_SERVER_ERROR, self.to_string()),
        };

        let body = Json(serde_json::json!({
            "error": message
        }));

        (status, body).into_response()
    }
}

// ============================================================================
// Tenant Extraction
// ============================================================================

const TENANT_HEADER_KEY: &str = "x-scope-orgid";
const DEFAULT_TENANT_ID: &str = "single-tenant";

/// Extract tenant ID from HTTP headers
pub fn extract_tenant_id(headers: &HeaderMap) -> Result<String, HttpError> {
    let tenant_id = headers
        .get(TENANT_HEADER_KEY)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .unwrap_or_else(|| DEFAULT_TENANT_ID.to_string());

    // Validate tenant ID format (alphanumeric with hyphens/underscores)
    if !tenant_id
        .chars()
        .all(|c| c.is_alphanumeric() || c == '-' || c == '_')
    {
        return Err(HttpError::InvalidTenantId);
    }

    Ok(tenant_id)
}

// ============================================================================
// Response Formatting
// ============================================================================

#[derive(Debug)]
pub enum ResponseFormat {
    Json,
    Protobuf,
}

impl ResponseFormat {
    pub fn from_headers(headers: &HeaderMap) -> Self {
        headers
            .get(header::ACCEPT)
            .and_then(|v| v.to_str().ok())
            .map(|s| {
                if s.contains("application/protobuf") || s.contains("application/x-protobuf") {
                    ResponseFormat::Protobuf
                } else {
                    ResponseFormat::Json
                }
            })
            .unwrap_or(ResponseFormat::Json)
    }
}
