# Tempo Querier Service - Rust Implementation Specification

## 1. Overview

This specification defines the implementation requirements for a Tempo Querier service in Rust that maintains interface compatibility with the existing Go implementation. The service coordinates distributed trace queries across ingester rings and storage backends, combining results while respecting tenant isolation and resource limits.

**Version:** 1.0
**Target Rust Edition:** 2021
**Protocol Buffers Version:** proto3

---

## 2. Architecture Overview

### 2.1 Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Querier Service                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ HTTP Server  │  │ gRPC Server  │  │Worker Client │      │
│  │  (axum)      │  │   (tonic)    │  │   (tonic)    │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │               │
│         └──────────────────┴──────────────────┘              │
│                            │                                  │
│                  ┌─────────▼─────────┐                       │
│                  │  Request Handler  │                       │
│                  │   & Validator     │                       │
│                  └─────────┬─────────┘                       │
│                            │                                  │
│         ┌──────────────────┼──────────────────┐             │
│         │                  │                  │              │
│    ┌────▼─────┐   ┌───────▼────────┐   ┌────▼─────┐       │
│    │Ingester  │   │  Storage       │   │Generator │       │
│    │Ring      │   │  Backend       │   │Ring      │       │
│    │Client    │   │  Client        │   │Client    │       │
│    └────┬─────┘   └───────┬────────┘   └────┬─────┘       │
│         │                  │                  │              │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
    [Ingesters]        [Object Store]     [Generators]
```

### 2.2 Data Flow

1. **Request Reception**: HTTP/gRPC request arrives with tenant context
2. **Authentication**: Extract and validate tenant ID from headers/metadata
3. **Authorization**: Check tenant limits and permissions
4. **Query Planning**: Determine query mode (ingesters, blocks, all)
5. **Parallel Execution**: Query ingesters and storage concurrently
6. **Result Combination**: Merge partial results respecting size limits
7. **Response Formatting**: Return JSON or Protobuf response

---

## 3. Protocol Buffer Service Definitions

### 3.1 Querier Service (Unary RPCs)

```protobuf
// File: proto/tempo.proto
syntax = "proto3";
package tempopb;

service Querier {
  rpc FindTraceByID(TraceByIDRequest) returns (TraceByIDResponse);
  rpc SearchRecent(SearchRequest) returns (SearchResponse);
  rpc SearchBlock(SearchBlockRequest) returns (SearchResponse);
  rpc SearchTags(SearchTagsRequest) returns (SearchTagsResponse);
  rpc SearchTagsV2(SearchTagsRequest) returns (SearchTagsV2Response);
  rpc SearchTagValues(SearchTagValuesRequest) returns (SearchTagValuesResponse);
  rpc SearchTagValuesV2(SearchTagValuesRequest) returns (SearchTagValuesV2Response);
}
```

**Rust Implementation Structure:**

```rust
use tonic::{Request, Response, Status};
use tokio::sync::RwLock;
use std::sync::Arc;

#[tonic::async_trait]
pub trait QuerierService: Send + Sync + 'static {
    async fn find_trace_by_id(
        &self,
        request: Request<TraceByIdRequest>,
    ) -> Result<Response<TraceByIdResponse>, Status>;

    async fn search_recent(
        &self,
        request: Request<SearchRequest>,
    ) -> Result<Response<SearchResponse>, Status>;

    async fn search_block(
        &self,
        request: Request<SearchBlockRequest>,
    ) -> Result<Response<SearchResponse>, Status>;

    async fn search_tags(
        &self,
        request: Request<SearchTagsRequest>,
    ) -> Result<Response<SearchTagsResponse>, Status>;

    async fn search_tags_v2(
        &self,
        request: Request<SearchTagsRequest>,
    ) -> Result<Response<SearchTagsV2Response>, Status>;

    async fn search_tag_values(
        &self,
        request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<SearchTagValuesResponse>, Status>;

    async fn search_tag_values_v2(
        &self,
        request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<SearchTagValuesV2Response>, Status>;
}
```

### 3.2 StreamingQuerier Service (Server Streaming RPCs)

```protobuf
service StreamingQuerier {
  rpc Search(SearchRequest) returns (stream SearchResponse);
  rpc SearchTags(SearchTagsRequest) returns (stream SearchTagsResponse);
  rpc SearchTagsV2(SearchTagsRequest) returns (stream SearchTagsV2Response);
  rpc SearchTagValues(SearchTagValuesRequest) returns (stream SearchTagValuesResponse);
  rpc SearchTagValuesV2(SearchTagValuesRequest) returns (stream SearchTagValuesV2Response);
  rpc MetricsQueryRange(QueryRangeRequest) returns (stream QueryRangeResponse);
  rpc MetricsQueryInstant(QueryInstantRequest) returns (stream QueryInstantResponse);
}
```

**Rust Implementation Structure:**

```rust
use tokio_stream::{Stream, StreamExt};
use std::pin::Pin;

type ResponseStream<T> = Pin<Box<dyn Stream<Item = Result<T, Status>> + Send>>;

#[tonic::async_trait]
pub trait StreamingQuerierService: Send + Sync + 'static {
    async fn search(
        &self,
        request: Request<SearchRequest>,
    ) -> Result<Response<ResponseStream<SearchResponse>>, Status>;

    async fn search_tags(
        &self,
        request: Request<SearchTagsRequest>,
    ) -> Result<Response<ResponseStream<SearchTagsResponse>>, Status>;

    async fn search_tags_v2(
        &self,
        request: Request<SearchTagsRequest>,
    ) -> Result<Response<ResponseStream<SearchTagsV2Response>>, Status>;

    async fn search_tag_values(
        &self,
        request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<ResponseStream<SearchTagValuesResponse>>, Status>;

    async fn search_tag_values_v2(
        &self,
        request: Request<SearchTagValuesRequest>,
    ) -> Result<Response<ResponseStream<SearchTagValuesV2Response>>, Status>;

    async fn metrics_query_range(
        &self,
        request: Request<QueryRangeRequest>,
    ) -> Result<Response<ResponseStream<QueryRangeResponse>>, Status>;

    async fn metrics_query_instant(
        &self,
        request: Request<QueryInstantRequest>,
    ) -> Result<Response<ResponseStream<QueryInstantResponse>>, Status>;
}
```

---

## 4. HTTP REST API Endpoints

### 4.1 Endpoint Definitions

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET/POST | `/querier/api/traces/{traceID}` | `trace_by_id_handler` | Retrieve trace by ID (v1) |
| GET/POST | `/querier/api/v2/traces/{traceID}` | `trace_by_id_handler_v2` | Retrieve trace by ID (v2, with protobuf support) |
| GET/POST | `/querier/api/search` | `search_handler` | Search recent traces |
| GET/POST | `/querier/api/search/tags` | `search_tags_handler` | Get searchable tags |
| GET/POST | `/querier/api/v2/search/tags` | `search_tags_v2_handler` | Get searchable tags (v2) |
| GET/POST | `/querier/api/search/tag/{tagName}/values` | `search_tag_values_handler` | Get values for a tag |
| GET/POST | `/querier/api/v2/search/tag/{tagName}/values` | `search_tag_values_v2_handler` | Get values for a tag (v2) |
| GET/POST | `/querier/api/metrics/summary` | `span_metrics_summary_handler` | Get span metrics summary |
| GET/POST | `/querier/api/metrics/query_range` | `query_range_handler` | Query metrics over time range |

### 4.2 HTTP Handler Implementation

```rust
use axum::{
    extract::{Path, Query, State},
    response::{IntoResponse, Response},
    http::{StatusCode, header},
    Json, Router,
};
use serde::{Deserialize, Serialize};

pub struct QuerierHttpHandler {
    querier: Arc<QuerierImpl>,
    config: QuerierConfig,
}

impl QuerierHttpHandler {
    pub fn router(self: Arc<Self>) -> Router {
        Router::new()
            .route("/querier/api/traces/:trace_id",
                axum::routing::get(Self::trace_by_id_handler)
                    .post(Self::trace_by_id_handler))
            .route("/querier/api/v2/traces/:trace_id",
                axum::routing::get(Self::trace_by_id_handler_v2)
                    .post(Self::trace_by_id_handler_v2))
            .route("/querier/api/search",
                axum::routing::get(Self::search_handler)
                    .post(Self::search_handler))
            .route("/querier/api/search/tags",
                axum::routing::get(Self::search_tags_handler)
                    .post(Self::search_tags_handler))
            .route("/querier/api/v2/search/tags",
                axum::routing::get(Self::search_tags_v2_handler)
                    .post(Self::search_tags_v2_handler))
            .route("/querier/api/search/tag/:tag_name/values",
                axum::routing::get(Self::search_tag_values_handler)
                    .post(Self::search_tag_values_handler))
            .route("/querier/api/v2/search/tag/:tag_name/values",
                axum::routing::get(Self::search_tag_values_v2_handler)
                    .post(Self::search_tag_values_v2_handler))
            .route("/querier/api/metrics/summary",
                axum::routing::get(Self::span_metrics_summary_handler)
                    .post(Self::span_metrics_summary_handler))
            .route("/querier/api/metrics/query_range",
                axum::routing::get(Self::query_range_handler)
                    .post(Self::query_range_handler))
            .with_state(self)
    }

    async fn trace_by_id_handler(
        State(handler): State<Arc<Self>>,
        Path(trace_id): Path<String>,
        Query(params): Query<TraceByIdParams>,
        headers: HeaderMap,
    ) -> Result<Response, QuerierError> {
        // 1. Extract tenant ID from headers
        let tenant_id = extract_tenant_id(&headers)?;

        // 2. Create request context with timeout
        let timeout = handler.config.trace_by_id.query_timeout;
        let ctx = RequestContext::new(tenant_id, timeout);

        // 3. Parse and validate trace ID
        let trace_id_bytes = parse_trace_id(&trace_id)?;

        // 4. Validate request parameters
        let (block_start, block_end, query_mode, time_start, time_end, rf1_after) =
            validate_and_sanitize_request(&params)?;

        // 5. Execute query
        let request = TraceByIdRequest {
            trace_id: trace_id_bytes,
            block_start,
            block_end,
            query_mode,
            allow_partial_trace: params.allow_partial,
            rf1_after,
        };

        let response = handler.querier
            .find_trace_by_id(ctx, request)
            .await?;

        // 6. Format response based on Accept header
        format_response(&headers, response)
    }

    // Similar implementations for other handlers...
}
```

### 4.3 Request/Response Types

```rust
// Query parameters for trace lookup
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

// Query parameters for search
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

// Query parameters for metrics
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
```

---

## 5. Core Data Structures

### 5.1 Querier Implementation

```rust
pub struct QuerierImpl {
    // Configuration
    config: QuerierConfig,

    // Ingester ring clients (multiple rings supported)
    ingester_pools: Vec<Arc<RingClientPool>>,
    ingester_rings: Vec<Arc<dyn ReadRing>>,

    // Generator ring client (for metrics)
    generator_pool: Option<Arc<RingClientPool>>,
    generator_ring: Option<Arc<dyn ReadRing>>,

    // Live store (post-rhythm architecture)
    query_partition_ring: bool,
    live_store_pool: Option<Arc<RingClientPool>>,
    live_store_ring: Option<Arc<dyn ReadRing>>,
    partition_ring: Option<Arc<PartitionInstanceRing>>,

    // Storage backend
    store: Arc<dyn StorageBackend>,

    // TraceQL engine
    engine: Arc<TraceQlEngine>,

    // Tenant limits and overrides
    limits: Arc<dyn LimitsProvider>,

    // Metrics and tracing
    metrics: QuerierMetrics,
}

impl QuerierImpl {
    pub fn new(
        config: QuerierConfig,
        ingester_pools: Vec<Arc<RingClientPool>>,
        ingester_rings: Vec<Arc<dyn ReadRing>>,
        store: Arc<dyn StorageBackend>,
        limits: Arc<dyn LimitsProvider>,
    ) -> Result<Self, QuerierError> {
        Ok(Self {
            config,
            ingester_pools,
            ingester_rings,
            generator_pool: None,
            generator_ring: None,
            query_partition_ring: false,
            live_store_pool: None,
            live_store_ring: None,
            partition_ring: None,
            store,
            engine: Arc::new(TraceQlEngine::new()),
            limits,
            metrics: QuerierMetrics::new(),
        })
    }
}
```

### 5.2 Configuration

```rust
use std::time::Duration;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct QuerierConfig {
    /// Search configuration
    pub search: SearchConfig,

    /// Trace lookup configuration
    pub trace_by_id: TraceByIdConfig,

    /// Metrics configuration
    pub metrics: MetricsConfig,

    /// Partition ring configuration
    pub partition_ring: PartitionRingConfig,

    /// Extra delay before querying ingesters (for eventual consistency)
    #[serde(default = "default_extra_query_delay")]
    pub extra_query_delay: Duration,

    /// Maximum concurrent queries
    #[serde(default = "default_max_concurrent_queries")]
    pub max_concurrent_queries: usize,

    /// Worker configuration for query frontend
    pub worker: WorkerConfig,

    /// Enable shuffle sharding for ingesters
    #[serde(default)]
    pub shuffle_sharding_ingesters_enabled: bool,

    /// Lookback period for shuffle sharding
    #[serde(default = "default_lookback_period")]
    pub shuffle_sharding_ingesters_lookback_period: Duration,

    /// Query only relevant ingesters based on trace ID hash
    #[serde(default)]
    pub query_relevant_ingesters: bool,

    /// Secondary ingester ring name
    pub secondary_ingester_ring: Option<String>,

    /// Query live store instead of ingesters (post-rhythm)
    #[serde(default)]
    pub query_live_store: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct SearchConfig {
    /// Query timeout
    #[serde(default = "default_search_timeout")]
    pub query_timeout: Duration,

    /// Maximum results to return
    #[serde(default = "default_search_max_results")]
    pub max_results: usize,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TraceByIdConfig {
    /// Query timeout
    #[serde(default = "default_trace_timeout")]
    pub query_timeout: Duration,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct MetricsConfig {
    /// Query timeout
    #[serde(default = "default_metrics_timeout")]
    pub query_timeout: Duration,

    /// Maximum samples per query
    #[serde(default = "default_max_samples")]
    pub max_samples: usize,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PartitionRingConfig {
    /// Enable partition ring queries
    #[serde(default)]
    pub enabled: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct WorkerConfig {
    /// Query frontend address
    pub frontend_address: String,

    /// Number of parallel workers
    #[serde(default = "default_parallelism")]
    pub parallelism: usize,

    /// Match max concurrent queries
    #[serde(default)]
    pub match_max_concurrent: bool,

    /// gRPC client configuration
    pub grpc_client_config: GrpcClientConfig,
}

// Default values
fn default_extra_query_delay() -> Duration { Duration::from_secs(0) }
fn default_max_concurrent_queries() -> usize { 20 }
fn default_lookback_period() -> Duration { Duration::from_secs(600) }
fn default_search_timeout() -> Duration { Duration::from_secs(30) }
fn default_search_max_results() -> usize { 1000 }
fn default_trace_timeout() -> Duration { Duration::from_secs(10) }
fn default_metrics_timeout() -> Duration { Duration::from_secs(30) }
fn default_max_samples() -> usize { 50_000_000 }
fn default_parallelism() -> usize { 10 }
```

---

## 6. Authentication and Authorization

### 6.1 Tenant Extraction

```rust
use tonic::metadata::MetadataMap;
use axum::http::HeaderMap;

pub const TENANT_HEADER_KEY: &str = "x-scope-orgid";
pub const DEFAULT_TENANT_ID: &str = "single-tenant";

/// Extract tenant ID from gRPC metadata
pub fn extract_tenant_from_metadata(
    metadata: &MetadataMap,
) -> Result<String, AuthError> {
    metadata
        .get(TENANT_HEADER_KEY)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .or_else(|| Some(DEFAULT_TENANT_ID.to_string()))
        .ok_or(AuthError::MissingTenantId)
}

/// Extract tenant ID from HTTP headers
pub fn extract_tenant_from_headers(
    headers: &HeaderMap,
) -> Result<String, AuthError> {
    headers
        .get(TENANT_HEADER_KEY)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .or_else(|| Some(DEFAULT_TENANT_ID.to_string()))
        .ok_or(AuthError::MissingTenantId)
}

/// Validate tenant ID format
pub fn validate_tenant_id(tenant_id: &str) -> Result<(), AuthError> {
    if tenant_id.is_empty() {
        return Err(AuthError::EmptyTenantId);
    }

    // Validate alphanumeric with hyphens/underscores
    if !tenant_id.chars().all(|c| c.is_alphanumeric() || c == '-' || c == '_') {
        return Err(AuthError::InvalidTenantIdFormat);
    }

    Ok(())
}
```

### 6.2 Tenant Context

```rust
use std::time::Instant;

/// Request context carrying tenant information and request metadata
#[derive(Debug, Clone)]
pub struct RequestContext {
    /// Tenant/organization ID
    pub tenant_id: String,

    /// Request deadline
    pub deadline: Instant,

    /// Request ID for tracing
    pub request_id: String,

    /// OpenTelemetry span context
    pub trace_context: Option<opentelemetry::Context>,
}

impl RequestContext {
    pub fn new(tenant_id: String, timeout: Duration) -> Self {
        Self {
            tenant_id,
            deadline: Instant::now() + timeout,
            request_id: uuid::Uuid::new_v4().to_string(),
            trace_context: None,
        }
    }

    pub fn with_trace_context(mut self, ctx: opentelemetry::Context) -> Self {
        self.trace_context = Some(ctx);
        self
    }

    pub fn remaining_time(&self) -> Duration {
        self.deadline.saturating_duration_since(Instant::now())
    }

    pub fn is_expired(&self) -> bool {
        Instant::now() >= self.deadline
    }
}
```

### 6.3 gRPC Interceptor

```rust
use tonic::service::Interceptor;

/// gRPC interceptor for tenant extraction and validation
#[derive(Clone)]
pub struct TenantInterceptor;

impl Interceptor for TenantInterceptor {
    fn call(&mut self, mut request: tonic::Request<()>) -> Result<tonic::Request<()>, Status> {
        // Extract tenant ID from metadata
        let tenant_id = extract_tenant_from_metadata(request.metadata())
            .map_err(|e| Status::unauthenticated(format!("Failed to extract tenant: {}", e)))?;

        // Validate tenant ID
        validate_tenant_id(&tenant_id)
            .map_err(|e| Status::invalid_argument(format!("Invalid tenant ID: {}", e)))?;

        // Store tenant ID in request extensions
        request.extensions_mut().insert(TenantId(tenant_id));

        Ok(request)
    }
}

/// Wrapper type for tenant ID stored in request extensions
#[derive(Debug, Clone)]
pub struct TenantId(pub String);

/// Extract tenant ID from tonic Request
pub fn get_tenant_id<T>(request: &tonic::Request<T>) -> Result<String, Status> {
    request
        .extensions()
        .get::<TenantId>()
        .map(|t| t.0.clone())
        .ok_or_else(|| Status::internal("Tenant ID not found in request extensions"))
}
```

### 6.4 HTTP Middleware

```rust
use axum::middleware::Next;
use axum::http::Request;

/// HTTP middleware for tenant extraction and validation
pub async fn tenant_middleware<B>(
    mut request: Request<B>,
    next: Next<B>,
) -> Result<Response, StatusCode> {
    // Extract tenant ID from headers
    let tenant_id = extract_tenant_from_headers(request.headers())
        .map_err(|_| StatusCode::UNAUTHORIZED)?;

    // Validate tenant ID
    validate_tenant_id(&tenant_id)
        .map_err(|_| StatusCode::BAD_REQUEST)?;

    // Store tenant ID in request extensions
    request.extensions_mut().insert(TenantId(tenant_id));

    Ok(next.run(request).await)
}

/// Extract tenant ID from axum Request
pub fn get_tenant_from_request<B>(request: &Request<B>) -> Result<String, StatusCode> {
    request
        .extensions()
        .get::<TenantId>()
        .map(|t| t.0.clone())
        .ok_or(StatusCode::INTERNAL_SERVER_ERROR)
}
```

### 6.5 Tenant Limits and Overrides

```rust
use async_trait::async_trait;

/// Trait for providing tenant-specific limits
#[async_trait]
pub trait LimitsProvider: Send + Sync {
    /// Maximum bytes per trace for tenant
    async fn max_bytes_per_trace(&self, tenant_id: &str) -> u64;

    /// Maximum search duration for tenant
    async fn max_search_duration(&self, tenant_id: &str) -> Duration;

    /// Maximum concurrent queries for tenant
    async fn max_concurrent_queries(&self, tenant_id: &str) -> usize;

    /// Ingestion rate limit in bytes/sec
    async fn ingestion_rate_limit_bytes(&self, tenant_id: &str) -> u64;

    /// Ingestion burst size in bytes
    async fn ingestion_burst_size_bytes(&self, tenant_id: &str) -> u64;

    /// Max trace size per span
    async fn max_bytes_per_tag_values_query(&self, tenant_id: &str) -> u64;

    /// Max blocks to query
    async fn max_blocks_per_tag_values_query(&self, tenant_id: &str) -> u32;
}

/// Default limits implementation
pub struct DefaultLimitsProvider {
    defaults: LimitsConfig,
    overrides: Arc<RwLock<HashMap<String, LimitsConfig>>>,
}

#[async_trait]
impl LimitsProvider for DefaultLimitsProvider {
    async fn max_bytes_per_trace(&self, tenant_id: &str) -> u64 {
        self.overrides
            .read()
            .await
            .get(tenant_id)
            .map(|c| c.max_bytes_per_trace)
            .unwrap_or(self.defaults.max_bytes_per_trace)
    }

    // Implement other methods similarly...
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct LimitsConfig {
    pub max_bytes_per_trace: u64,
    pub max_search_duration: Duration,
    pub max_concurrent_queries: usize,
    pub ingestion_rate_limit_bytes: u64,
    pub ingestion_burst_size_bytes: u64,
    pub max_bytes_per_tag_values_query: u64,
    pub max_blocks_per_tag_values_query: u32,
}

impl Default for LimitsConfig {
    fn default() -> Self {
        Self {
            max_bytes_per_trace: 50_000_000, // 50MB
            max_search_duration: Duration::from_secs(30),
            max_concurrent_queries: 20,
            ingestion_rate_limit_bytes: 15_000_000, // 15MB/s
            ingestion_burst_size_bytes: 20_000_000, // 20MB
            max_bytes_per_tag_values_query: 5_000_000, // 5MB
            max_blocks_per_tag_values_query: 100,
        }
    }
}
```

---

## 7. Query Processing Implementation

### 7.1 FindTraceByID

```rust
impl QuerierImpl {
    /// Find trace by ID across ingesters and storage
    pub async fn find_trace_by_id(
        &self,
        ctx: RequestContext,
        request: TraceByIdRequest,
    ) -> Result<TraceByIdResponse, QuerierError> {
        // 1. Validate trace ID
        if !is_valid_trace_id(&request.trace_id) {
            return Err(QuerierError::InvalidTraceId);
        }

        // 2. Get tenant limits
        let max_bytes = self.limits
            .max_bytes_per_trace(&ctx.tenant_id)
            .await;

        // 3. Create trace combiner
        let mut combiner = TraceCombiner::new(
            max_bytes,
            request.allow_partial_trace,
        );

        // 4. Query ingesters if needed
        if should_query_ingesters(&request.query_mode) {
            let ingester_results = self
                .query_ingesters(&ctx, &request)
                .await?;

            for result in ingester_results {
                combiner.add_trace(result.trace)?;
            }
        }

        // 5. Query storage if needed
        if should_query_storage(&request.query_mode) {
            let storage_results = self
                .query_storage(&ctx, &request)
                .await?;

            for result in storage_results {
                combiner.add_trace(result)?;
            }
        }

        // 6. Combine and return results
        let (trace, metrics) = combiner.finalize();

        Ok(TraceByIdResponse {
            trace: Some(trace),
            metrics: Some(metrics),
            status: combiner.status(),
            message: combiner.status_message(),
        })
    }

    /// Query all ingester rings in parallel
    async fn query_ingesters(
        &self,
        ctx: &RequestContext,
        request: &TraceByIdRequest,
    ) -> Result<Vec<TraceByIdResponse>, QuerierError> {
        let mut tasks = Vec::new();

        // Launch parallel queries to each ingester ring
        for (pool, ring) in self.ingester_pools.iter().zip(&self.ingester_rings) {
            let ctx = ctx.clone();
            let request = request.clone();
            let pool = pool.clone();
            let ring = ring.clone();

            tasks.push(tokio::spawn(async move {
                query_one_ingester_ring(&ctx, &request, pool, ring).await
            }));
        }

        // Collect results
        let mut all_results = Vec::new();
        for task in tasks {
            let results = task.await??;
            all_results.extend(results);
        }

        Ok(all_results)
    }

    /// Query storage backend
    async fn query_storage(
        &self,
        ctx: &RequestContext,
        request: &TraceByIdRequest,
    ) -> Result<Vec<Trace>, QuerierError> {
        // Brief: Query object storage (S3/GCS/etc.) for blocks
        // within the specified time range. Returns partial traces
        // from each matching block.

        self.store.find_trace_by_id(
            &ctx.tenant_id,
            &request.trace_id,
            request.block_start.as_ref(),
            request.block_end.as_ref(),
            ctx.remaining_time(),
        ).await
    }
}

/// Query a single ingester ring
async fn query_one_ingester_ring(
    ctx: &RequestContext,
    request: &TraceByIdRequest,
    pool: Arc<RingClientPool>,
    ring: Arc<dyn ReadRing>,
) -> Result<Vec<TraceByIdResponse>, QuerierError> {
    // Brief: Get replication set from ring, connect to each ingester
    // via gRPC, send FindTraceByID request in parallel, collect responses.

    let replication_set = ring.get_replication_set_for_operation()?;
    let mut tasks = Vec::new();

    for instance in replication_set.instances {
        let client = pool.get_client(&instance.addr).await?;
        let request = request.clone();
        let ctx = ctx.clone();

        tasks.push(tokio::spawn(async move {
            let mut client = client.clone();
            client.find_trace_by_id(request).await
        }));
    }

    // Collect successful responses
    let mut results = Vec::new();
    for task in tasks {
        if let Ok(Ok(response)) = task.await {
            results.push(response);
        }
    }

    Ok(results)
}

fn should_query_ingesters(mode: &str) -> bool {
    matches!(mode, "ingesters" | "all" | "recent")
}

fn should_query_storage(mode: &str) -> bool {
    matches!(mode, "blocks" | "all")
}
```

### 7.2 SearchRecent

```rust
impl QuerierImpl {
    /// Search recent traces (ingesters only)
    pub async fn search_recent(
        &self,
        ctx: RequestContext,
        request: SearchRequest,
    ) -> Result<SearchResponse, QuerierError> {
        // 1. Validate request
        validate_search_request(&request)?;

        // 2. Query all ingester rings in parallel
        let results = self
            .query_ingesters_search(&ctx, &request)
            .await?;

        // 3. Post-process and combine results
        let combined = self.combine_search_results(request, results)?;

        Ok(combined)
    }

    async fn query_ingesters_search(
        &self,
        ctx: &RequestContext,
        request: &SearchRequest,
    ) -> Result<Vec<SearchResponse>, QuerierError> {
        // Brief: Similar to find_trace_by_id but calls SearchRecent
        // on each ingester. Returns list of SearchResponse.

        // Implementation similar to query_ingesters but for search
        todo!("Query ingesters for search")
    }

    fn combine_search_results(
        &self,
        request: SearchRequest,
        results: Vec<SearchResponse>,
    ) -> Result<SearchResponse, QuerierError> {
        // Brief: Merge search results, deduplicate traces by ID,
        // sort by start time, apply limit, aggregate metrics.

        todo!("Combine search results")
    }
}
```

### 7.3 Query Range (Metrics)

```rust
impl QuerierImpl {
    /// Execute metrics query over time range
    pub async fn metrics_query_range(
        &self,
        ctx: RequestContext,
        request: QueryRangeRequest,
    ) -> Result<QueryRangeResponse, QuerierError> {
        // 1. Parse and validate TraceQL metrics query
        let query = self.engine.parse_metrics_query(&request.query)?;

        // 2. Get tenant limits
        let max_samples = self.limits
            .max_bytes_per_tag_values_query(&ctx.tenant_id)
            .await as usize;

        // 3. Query generators (metrics-generator instances)
        let results = self
            .query_generators(&ctx, &request)
            .await?;

        // 4. Combine and downsample results
        let combined = combine_metrics_results(results, max_samples)?;

        Ok(QueryRangeResponse {
            series: combined.series,
            metrics: Some(combined.metrics),
        })
    }

    async fn query_generators(
        &self,
        ctx: &RequestContext,
        request: &QueryRangeRequest,
    ) -> Result<Vec<QueryRangeResponse>, QuerierError> {
        // Brief: Query generator ring for metrics time series.
        // Similar pattern to ingester queries but targets generators.

        todo!("Query generators for metrics")
    }
}
```

---

## 8. Result Combination

### 8.1 Trace Combiner

```rust
use std::collections::HashMap;

pub struct TraceCombiner {
    max_bytes: u64,
    allow_partial: bool,
    current_bytes: u64,
    spans_by_id: HashMap<Vec<u8>, Span>,
    is_partial: bool,
    inspected_bytes: u64,
}

impl TraceCombiner {
    pub fn new(max_bytes: u64, allow_partial: bool) -> Self {
        Self {
            max_bytes,
            allow_partial,
            current_bytes: 0,
            spans_by_id: HashMap::new(),
            is_partial: false,
            inspected_bytes: 0,
        }
    }

    /// Add a trace to the combiner
    pub fn add_trace(&mut self, trace: Trace) -> Result<(), QuerierError> {
        // Brief: Iterate through spans, deduplicate by span ID,
        // track size, mark as partial if exceeds limit.

        for batch in trace.batches {
            for span in batch.spans {
                let span_size = estimate_span_size(&span);
                self.inspected_bytes += span_size;

                // Check size limit
                if self.current_bytes + span_size > self.max_bytes {
                    self.is_partial = true;
                    if !self.allow_partial {
                        return Err(QuerierError::TraceTooLarge);
                    }
                    continue;
                }

                // Deduplicate by span ID
                self.spans_by_id.entry(span.span_id.clone())
                    .or_insert_with(|| {
                        self.current_bytes += span_size;
                        span
                    });
            }
        }

        Ok(())
    }

    /// Finalize and return combined trace
    pub fn finalize(self) -> (Trace, TraceByIdMetrics) {
        let spans: Vec<Span> = self.spans_by_id.into_values().collect();

        let trace = Trace {
            batches: vec![ResourceSpans {
                spans,
                ..Default::default()
            }],
        };

        let metrics = TraceByIdMetrics {
            inspected_bytes: self.inspected_bytes,
        };

        (trace, metrics)
    }

    pub fn status(&self) -> PartialStatus {
        if self.is_partial {
            PartialStatus::Partial
        } else {
            PartialStatus::Complete
        }
    }

    pub fn status_message(&self) -> String {
        if self.is_partial {
            format!("Trace exceeded size limit of {} bytes", self.max_bytes)
        } else {
            String::new()
        }
    }
}

fn estimate_span_size(span: &Span) -> u64 {
    // Brief: Estimate span size by summing field sizes
    // (span_id, trace_id, name, attributes, events, etc.)
    todo!("Estimate span size")
}
```

---

## 9. Ring Client and Service Discovery

### 9.1 Ring Abstraction

```rust
use async_trait::async_trait;

/// Trait for consistent hash ring operations
#[async_trait]
pub trait ReadRing: Send + Sync {
    /// Get replication set for a given operation
    fn get_replication_set_for_operation(&self) -> Result<ReplicationSet, RingError>;

    /// Get all healthy instances
    async fn get_all_healthy_instances(&self) -> Result<Vec<InstanceDesc>, RingError>;

    /// Get instances for a specific key (e.g., trace ID hash)
    fn get_replication_set_for_key(&self, key: u32) -> Result<ReplicationSet, RingError>;
}

#[derive(Debug, Clone)]
pub struct ReplicationSet {
    pub instances: Vec<InstanceDesc>,
    pub max_errors: usize,
}

#[derive(Debug, Clone)]
pub struct InstanceDesc {
    pub id: String,
    pub addr: String,
    pub zone: String,
    pub state: InstanceState,
    pub tokens: Vec<u32>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum InstanceState {
    Active,
    Leaving,
    Joining,
    Unhealthy,
}
```

### 9.2 Ring Client Pool

```rust
use dashmap::DashMap;

/// Pool of gRPC clients for ring instances
pub struct RingClientPool {
    clients: DashMap<String, Arc<QuerierGrpcClient>>,
    config: GrpcClientConfig,
}

impl RingClientPool {
    pub fn new(config: GrpcClientConfig) -> Self {
        Self {
            clients: DashMap::new(),
            config,
        }
    }

    /// Get or create client for an address
    pub async fn get_client(
        &self,
        addr: &str,
    ) -> Result<Arc<QuerierGrpcClient>, QuerierError> {
        // Check if client exists
        if let Some(client) = self.clients.get(addr) {
            return Ok(client.clone());
        }

        // Create new client
        let endpoint = tonic::transport::Channel::from_shared(format!("http://{}", addr))
            .map_err(|e| QuerierError::InvalidEndpoint(e.to_string()))?
            .connect_timeout(self.config.connect_timeout)
            .timeout(self.config.request_timeout);

        let channel = endpoint.connect().await?;
        let client = Arc::new(QuerierGrpcClient::new(channel));

        // Store and return
        self.clients.insert(addr.to_string(), client.clone());
        Ok(client)
    }

    /// Remove client from pool
    pub fn remove_client(&self, addr: &str) {
        self.clients.remove(addr);
    }
}

#[derive(Debug, Clone)]
pub struct GrpcClientConfig {
    pub connect_timeout: Duration,
    pub request_timeout: Duration,
    pub max_recv_msg_size: usize,
    pub max_send_msg_size: usize,
}

impl Default for GrpcClientConfig {
    fn default() -> Self {
        Self {
            connect_timeout: Duration::from_secs(5),
            request_timeout: Duration::from_secs(30),
            max_recv_msg_size: 100 * 1024 * 1024, // 100MB
            max_send_msg_size: 16 * 1024 * 1024,  // 16MB
        }
    }
}
```

---

## 10. Storage Backend Integration

### 10.1 Storage Trait

```rust
#[async_trait]
pub trait StorageBackend: Send + Sync {
    /// Find trace by ID in storage
    async fn find_trace_by_id(
        &self,
        tenant_id: &str,
        trace_id: &[u8],
        block_start: Option<&str>,
        block_end: Option<&str>,
        timeout: Duration,
    ) -> Result<Vec<Trace>, StorageError>;

    /// Search for traces matching criteria
    async fn search(
        &self,
        tenant_id: &str,
        request: &SearchRequest,
        timeout: Duration,
    ) -> Result<Vec<TraceSearchMetadata>, StorageError>;

    /// Search tags in storage
    async fn search_tags(
        &self,
        tenant_id: &str,
        request: &SearchTagsRequest,
        timeout: Duration,
    ) -> Result<Vec<String>, StorageError>;

    /// Search tag values in storage
    async fn search_tag_values(
        &self,
        tenant_id: &str,
        tag_name: &str,
        request: &SearchTagValuesRequest,
        timeout: Duration,
    ) -> Result<Vec<String>, StorageError>;

    /// Fetch blocks for a tenant within time range
    async fn fetch_blocks(
        &self,
        tenant_id: &str,
        start_time: i64,
        end_time: i64,
    ) -> Result<Vec<BlockMeta>, StorageError>;
}

#[derive(Debug, Clone)]
pub struct BlockMeta {
    pub block_id: String,
    pub tenant_id: String,
    pub start_time: i64,
    pub end_time: i64,
    pub size: u64,
    pub total_objects: u64,
}
```

---

## 11. Response Formatting

### 11.1 Format Detection

```rust
pub enum ResponseFormat {
    Json,
    Protobuf,
}

impl ResponseFormat {
    pub fn from_headers(headers: &HeaderMap) -> Self {
        headers
            .get("accept")
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
```

### 11.2 Response Builder

```rust
use prost::Message;

pub fn format_response<T: Message>(
    headers: &HeaderMap,
    message: T,
) -> Result<Response, QuerierError> {
    match ResponseFormat::from_headers(headers) {
        ResponseFormat::Protobuf => {
            let mut buf = Vec::new();
            message.encode(&mut buf)?;

            Ok(Response::builder()
                .status(StatusCode::OK)
                .header(header::CONTENT_TYPE, "application/protobuf")
                .body(Body::from(buf))?)
        }
        ResponseFormat::Json => {
            let json = serde_json::to_string(&message)?;

            Ok(Response::builder()
                .status(StatusCode::OK)
                .header(header::CONTENT_TYPE, "application/json")
                .body(Body::from(json))?)
        }
    }
}
```

---

## 12. Worker Client (Query Frontend Integration)

### 12.1 Worker Implementation

```rust
pub struct QuerierWorker {
    config: WorkerConfig,
    handler: Arc<QuerierHttpHandler>,
    client: Option<FrontendClient>,
}

impl QuerierWorker {
    pub fn new(
        config: WorkerConfig,
        handler: Arc<QuerierHttpHandler>,
    ) -> Self {
        Self {
            config,
            handler,
            client: None,
        }
    }

    /// Connect to query frontend and process queries
    pub async fn run(&mut self) -> Result<(), QuerierError> {
        // Connect to frontend
        let channel = tonic::transport::Channel::from_shared(
            self.config.frontend_address.clone()
        )?
        .connect()
        .await?;

        let mut client = FrontendClient::new(channel);

        // Start bidirectional stream
        let outbound = tokio_stream::pending();
        let response = client.process(outbound).await?;
        let mut inbound = response.into_inner();

        // Process queries
        while let Some(request) = inbound.message().await? {
            self.handle_frontend_request(request).await?;
        }

        Ok(())
    }

    async fn handle_frontend_request(
        &self,
        request: FrontendRequest,
    ) -> Result<(), QuerierError> {
        // Brief: Convert httpgrpc request to axum Request,
        // route to appropriate handler, convert response back
        // to httpgrpc format, send to frontend.

        todo!("Handle frontend request")
    }
}
```

---

## 13. Error Handling

### 13.1 Error Types

```rust
use thiserror::Error;

#[derive(Debug, Error)]
pub enum QuerierError {
    #[error("Invalid trace ID")]
    InvalidTraceId,

    #[error("Trace too large: {0} bytes")]
    TraceTooLarge(u64),

    #[error("Query timeout exceeded")]
    Timeout,

    #[error("Storage error: {0}")]
    Storage(#[from] StorageError),

    #[error("Ring error: {0}")]
    Ring(#[from] RingError),

    #[error("Authentication error: {0}")]
    Auth(#[from] AuthError),

    #[error("Invalid request: {0}")]
    InvalidRequest(String),

    #[error("gRPC error: {0}")]
    Grpc(#[from] tonic::Status),

    #[error("Internal error: {0}")]
    Internal(String),
}

#[derive(Debug, Error)]
pub enum AuthError {
    #[error("Missing tenant ID")]
    MissingTenantId,

    #[error("Empty tenant ID")]
    EmptyTenantId,

    #[error("Invalid tenant ID format")]
    InvalidTenantIdFormat,

    #[error("Tenant not authorized")]
    Unauthorized,
}

#[derive(Debug, Error)]
pub enum StorageError {
    #[error("Block not found: {0}")]
    BlockNotFound(String),

    #[error("Object store error: {0}")]
    ObjectStore(String),

    #[error("Decoding error: {0}")]
    Decode(String),
}

#[derive(Debug, Error)]
pub enum RingError {
    #[error("No healthy instances")]
    NoHealthyInstances,

    #[error("Replication set not found")]
    ReplicationSetNotFound,
}

impl From<QuerierError> for tonic::Status {
    fn from(err: QuerierError) -> Self {
        match err {
            QuerierError::InvalidTraceId => Status::invalid_argument(err.to_string()),
            QuerierError::InvalidRequest(_) => Status::invalid_argument(err.to_string()),
            QuerierError::Auth(_) => Status::unauthenticated(err.to_string()),
            QuerierError::Timeout => Status::deadline_exceeded(err.to_string()),
            _ => Status::internal(err.to_string()),
        }
    }
}

impl IntoResponse for QuerierError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            QuerierError::InvalidTraceId => (StatusCode::BAD_REQUEST, self.to_string()),
            QuerierError::InvalidRequest(_) => (StatusCode::BAD_REQUEST, self.to_string()),
            QuerierError::Auth(_) => (StatusCode::UNAUTHORIZED, self.to_string()),
            QuerierError::Timeout => (StatusCode::GATEWAY_TIMEOUT, self.to_string()),
            _ => (StatusCode::INTERNAL_SERVER_ERROR, self.to_string()),
        };

        (status, Json(serde_json::json!({
            "error": message
        }))).into_response()
    }
}
```

---

## 14. Observability

### 14.1 Metrics

```rust
use prometheus::{
    Registry, IntCounter, IntGauge, Histogram, HistogramOpts, IntCounterVec,
    register_int_counter_vec_with_registry,
    register_histogram_vec_with_registry,
};

pub struct QuerierMetrics {
    // Request counts
    pub requests_total: IntCounterVec,
    pub requests_failed: IntCounterVec,

    // Latency histograms
    pub request_duration: HistogramVec,
    pub ingester_query_duration: HistogramVec,
    pub storage_query_duration: HistogramVec,

    // Resource usage
    pub bytes_inspected: IntCounterVec,
    pub traces_found: IntCounterVec,
    pub spans_ingested: IntCounter,

    // Current state
    pub concurrent_queries: IntGauge,
}

impl QuerierMetrics {
    pub fn new(registry: &Registry) -> Result<Self, prometheus::Error> {
        let requests_total = register_int_counter_vec_with_registry!(
            "tempo_querier_requests_total",
            "Total number of querier requests",
            &["method", "status"],
            registry
        )?;

        let requests_failed = register_int_counter_vec_with_registry!(
            "tempo_querier_requests_failed_total",
            "Total number of failed querier requests",
            &["method", "reason"],
            registry
        )?;

        let request_duration = register_histogram_vec_with_registry!(
            "tempo_querier_request_duration_seconds",
            "Request duration in seconds",
            &["method"],
            registry
        )?;

        // ... register other metrics

        Ok(Self {
            requests_total,
            requests_failed,
            request_duration,
            // ... other fields
        })
    }
}
```

### 14.2 Distributed Tracing

```rust
use opentelemetry::trace::{Tracer, Span, Status};
use tracing::{info, error, instrument};
use tracing_opentelemetry::OpenTelemetrySpanExt;

#[instrument(skip(self, ctx, request), fields(
    tenant_id = %ctx.tenant_id,
    trace_id = ?request.trace_id,
))]
pub async fn find_trace_by_id(
    &self,
    ctx: RequestContext,
    request: TraceByIdRequest,
) -> Result<TraceByIdResponse, QuerierError> {
    let span = tracing::Span::current();

    // Add custom attributes
    span.set_attribute("query_mode", request.query_mode.clone());
    span.set_attribute("allow_partial", request.allow_partial_trace);

    // ... implementation

    info!(
        inspected_bytes = response.metrics.inspected_bytes,
        "trace query completed"
    );

    Ok(response)
}
```

---

## 15. Validation and Testing

### 15.1 Request Validation

```rust
pub fn validate_trace_id(trace_id: &[u8]) -> Result<(), QuerierError> {
    if trace_id.is_empty() {
        return Err(QuerierError::InvalidRequest("Empty trace ID".to_string()));
    }

    if trace_id.len() != 16 && trace_id.len() != 32 {
        return Err(QuerierError::InvalidRequest(
            format!("Invalid trace ID length: expected 16 or 32 bytes, got {}", trace_id.len())
        ));
    }

    Ok(())
}

pub fn validate_search_request(request: &SearchRequest) -> Result<(), QuerierError> {
    if let Some(limit) = request.limit {
        if limit == 0 || limit > 10000 {
            return Err(QuerierError::InvalidRequest(
                "Limit must be between 1 and 10000".to_string()
            ));
        }
    }

    if let Some(min_duration) = request.min_duration_ms {
        if let Some(max_duration) = request.max_duration_ms {
            if min_duration > max_duration {
                return Err(QuerierError::InvalidRequest(
                    "min_duration cannot be greater than max_duration".to_string()
                ));
            }
        }
    }

    Ok(())
}
```

### 15.2 Testing Strategy

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_find_trace_by_id_valid() {
        let querier = create_test_querier().await;
        let ctx = create_test_context("test-tenant");

        let request = TraceByIdRequest {
            trace_id: vec![1; 16],
            query_mode: "all".to_string(),
            allow_partial_trace: false,
            ..Default::default()
        };

        let response = querier
            .find_trace_by_id(ctx, request)
            .await
            .expect("Query should succeed");

        assert!(response.trace.is_some());
    }

    #[tokio::test]
    async fn test_find_trace_by_id_invalid_trace_id() {
        let querier = create_test_querier().await;
        let ctx = create_test_context("test-tenant");

        let request = TraceByIdRequest {
            trace_id: vec![],  // Invalid: empty
            query_mode: "all".to_string(),
            ..Default::default()
        };

        let result = querier.find_trace_by_id(ctx, request).await;
        assert!(matches!(result, Err(QuerierError::InvalidTraceId)));
    }

    #[tokio::test]
    async fn test_tenant_extraction_from_headers() {
        let mut headers = HeaderMap::new();
        headers.insert("x-scope-orgid", "test-tenant-123".parse().unwrap());

        let tenant_id = extract_tenant_from_headers(&headers)
            .expect("Should extract tenant");

        assert_eq!(tenant_id, "test-tenant-123");
    }

    // Mock implementations for testing
    async fn create_test_querier() -> QuerierImpl {
        // Create mock components
        todo!("Create test querier")
    }

    fn create_test_context(tenant_id: &str) -> RequestContext {
        RequestContext::new(
            tenant_id.to_string(),
            Duration::from_secs(30),
        )
    }
}
```

---

## 16. Dependencies (Cargo.toml)

```toml
[package]
name = "tempo-querier"
version = "0.1.0"
edition = "2021"

[dependencies]
# Async runtime
tokio = { version = "1.35", features = ["full"] }
tokio-stream = "0.1"

# gRPC/Protobuf
tonic = "0.11"
prost = "0.12"
prost-types = "0.12"

# HTTP server
axum = "0.7"
tower = "0.4"
tower-http = { version = "0.5", features = ["trace", "cors"] }
hyper = "1.0"

# Serialization
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"

# Error handling
thiserror = "1.0"
anyhow = "1.0"

# Observability
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter", "json"] }
tracing-opentelemetry = "0.22"
opentelemetry = { version = "0.21", features = ["trace"] }
opentelemetry-otlp = "0.14"
prometheus = "0.13"

# Concurrency
dashmap = "5.5"
arc-swap = "1.6"

# UUID generation
uuid = { version = "1.6", features = ["v4"] }

# Time handling
chrono = "0.4"

# Configuration
config = "0.14"
toml = "0.8"

# Hashing (for consistent hashing)
xxhash-rust = { version = "0.8", features = ["xxh3"] }

# Async trait
async-trait = "0.1"

[build-dependencies]
tonic-build = "0.11"

[dev-dependencies]
mockall = "0.12"
```

---

## 17. Build Configuration (build.rs)

```rust
fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .compile(
            &["proto/tempo.proto"],
            &["proto/"],
        )?;

    Ok(())
}
```

---

## 18. Main Service Entry Point

```rust
use std::sync::Arc;
use tokio::signal;
use tracing::info;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Load configuration
    let config = QuerierConfig::from_file("config.toml")?;

    // Initialize components
    let storage = create_storage_backend(&config.storage).await?;
    let limits = Arc::new(create_limits_provider(&config.limits).await?);
    let (ingester_pools, ingester_rings) = create_ingester_clients(&config.ingesters).await?;

    // Create querier
    let querier = Arc::new(QuerierImpl::new(
        config.clone(),
        ingester_pools,
        ingester_rings,
        storage,
        limits,
    )?);

    // Create HTTP handler
    let http_handler = Arc::new(QuerierHttpHandler::new(querier.clone(), config.clone()));

    // Start HTTP server
    let http_addr = config.http_listen_address.parse()?;
    let http_router = http_handler.router();
    let http_server = axum::Server::bind(&http_addr)
        .serve(http_router.into_make_service());

    info!("HTTP server listening on {}", http_addr);

    // Start gRPC server
    let grpc_addr = config.grpc_listen_address.parse()?;
    let grpc_server = tonic::transport::Server::builder()
        .add_service(QuerierServer::with_interceptor(
            querier.clone(),
            TenantInterceptor,
        ))
        .add_service(StreamingQuerierServer::with_interceptor(
            querier.clone(),
            TenantInterceptor,
        ))
        .serve(grpc_addr);

    info!("gRPC server listening on {}", grpc_addr);

    // Start worker (if configured)
    let worker_task = if let Some(worker_config) = config.worker {
        let mut worker = QuerierWorker::new(worker_config, http_handler.clone());
        Some(tokio::spawn(async move {
            worker.run().await
        }))
    } else {
        None
    };

    // Wait for shutdown signal
    tokio::select! {
        _ = http_server => {},
        _ = grpc_server => {},
        _ = signal::ctrl_c() => {
            info!("Shutdown signal received");
        }
    }

    Ok(())
}
```

---

## 19. Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] Protocol buffer definitions and code generation
- [ ] Configuration loading and validation
- [ ] Error types and handling
- [ ] Request context and tenant extraction
- [ ] Metrics and tracing infrastructure

### Phase 2: Authentication & Authorization
- [ ] Tenant ID extraction from headers/metadata
- [ ] gRPC interceptor for tenant validation
- [ ] HTTP middleware for tenant validation
- [ ] Limits provider interface and implementation
- [ ] Per-tenant limits enforcement

### Phase 3: Ring Integration
- [ ] Ring abstraction trait
- [ ] Ring client pool implementation
- [ ] Replication set handling
- [ ] Health checking

### Phase 4: Query Implementation
- [ ] FindTraceByID (unary)
- [ ] SearchRecent (unary)
- [ ] SearchTags/TagValues (unary)
- [ ] Streaming search implementation
- [ ] Metrics query range/instant

### Phase 5: Storage Integration
- [ ] Storage backend trait
- [ ] Object store client (S3/GCS)
- [ ] Block metadata handling
- [ ] Parallel block queries

### Phase 6: Result Processing
- [ ] Trace combiner implementation
- [ ] Search result aggregation
- [ ] Metrics result aggregation
- [ ] Size limit enforcement

### Phase 7: HTTP/gRPC Servers
- [ ] HTTP endpoint handlers
- [ ] gRPC service implementations
- [ ] Response formatting (JSON/Protobuf)
- [ ] Request validation

### Phase 8: Worker Integration
- [ ] Frontend client connection
- [ ] Bidirectional streaming
- [ ] HTTP-over-gRPC handling

### Phase 9: Testing
- [ ] Unit tests for all components
- [ ] Integration tests with mock services
- [ ] Load testing
- [ ] Compatibility testing with Go implementation

### Phase 10: Production Readiness
- [ ] Graceful shutdown
- [ ] Health check endpoints
- [ ] Configuration hot-reload
- [ ] Documentation
- [ ] Docker containerization

---

## 20. Notes and Considerations

### Performance Optimizations
- Use `Arc` for shared immutable data to avoid cloning
- Use `DashMap` for concurrent access to shared mutable data
- Leverage `tokio::spawn` for parallel queries
- Consider using connection pooling with `bb8` or `deadpool`
- Use zero-copy deserialization where possible

### Compatibility
- Ensure protobuf compatibility with Go implementation
- Match HTTP endpoint paths and query parameters exactly
- Support same query modes: "ingesters", "blocks", "all", "recent"
- Maintain wire format compatibility for cross-service communication

### Security
- Always validate tenant IDs before processing requests
- Enforce per-tenant rate limits
- Sanitize user inputs (trace IDs, TraceQL queries)
- Use TLS for gRPC connections in production
- Consider implementing request signing for inter-service auth

### Scalability
- Design for horizontal scaling (stateless service)
- Use consistent hashing for cache affinity
- Implement circuit breakers for failing backends
- Add request queuing and backpressure handling

### Observability
- Emit metrics for all operations (success, failure, duration)
- Add distributed tracing context propagation
- Log at appropriate levels (error, warn, info, debug)
- Include tenant ID in all logs and traces

---

## 21. Example Configuration File

```toml
# config.toml

[querier]
http_listen_address = "0.0.0.0:3200"
grpc_listen_address = "0.0.0.0:9095"
max_concurrent_queries = 20
extra_query_delay = "0s"
query_relevant_ingesters = true
query_live_store = false

[querier.search]
query_timeout = "30s"
max_results = 1000

[querier.trace_by_id]
query_timeout = "10s"

[querier.metrics]
query_timeout = "30s"
max_samples = 50000000

[querier.worker]
frontend_address = "query-frontend:9095"
parallelism = 10
match_max_concurrent = true

[querier.worker.grpc_client_config]
connect_timeout = "5s"
request_timeout = "30s"
max_recv_msg_size = 104857600  # 100MB
max_send_msg_size = 16777216   # 16MB

[storage]
backend = "s3"
bucket = "tempo-traces"
endpoint = "s3.amazonaws.com"

[limits]
max_bytes_per_trace = 50000000
max_search_duration = "30s"
max_concurrent_queries = 20
ingestion_rate_limit_bytes = 15000000
ingestion_burst_size_bytes = 20000000

[ingesters]
ring_addresses = ["consul:8500"]

[observability.tracing]
enabled = true
endpoint = "localhost:4317"
service_name = "tempo-querier"

[observability.metrics]
enabled = true
listen_address = "0.0.0.0:9090"
```

---

## End of Specification

This specification provides a complete blueprint for implementing a Tempo-compatible querier service in Rust. All interfaces maintain compatibility with the existing Go implementation while leveraging Rust's type safety, performance, and modern async ecosystem.
