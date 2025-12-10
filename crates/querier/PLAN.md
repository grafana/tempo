# Rust Querier Frontend Worker Implementation Plan

## Summary

Implement a Rust querier service using the **Frontend Worker architecture** (pull model). The querier connects to query-frontend via bidirectional gRPC streaming, pulls HTTP requests, executes them locally via Axum handlers, and returns responses.

## Key Files to Modify/Create

| File | Action | Purpose |
|------|--------|---------|
| `crates/querier/src/main.rs` | **Rewrite** | Remove gRPC server, add worker startup |
| `crates/querier/src/http.rs` | **Keep & Enhance** | HTTP handlers (already scaffolded) |
| `crates/querier/build.rs` | **Extend** | Add httpgrpc + frontend proto compilation |
| `crates/querier/Cargo.toml` | **Extend** | Add new dependencies |
| `crates/querier/src/config.rs` | **Create** | Worker configuration |
| `crates/querier/src/worker/mod.rs` | **Create** | Worker module |
| `crates/querier/src/worker/worker.rs` | **Create** | Main worker service |
| `crates/querier/src/worker/processor_manager.rs` | **Create** | Manages N processor tasks per connection |
| `crates/querier/src/worker/frontend_processor.rs` | **Create** | Handles bidirectional streaming |
| `crates/querier/src/request_handler.rs` | **Create** | HTTPRequest/Response conversion |
| `crates/querier/src/error.rs` | **Create** | Unified error types |
| `crates/querier/src/lib.rs` | **Create** | Library exports |

## Code to Remove

From `main.rs`:
- `QuerierService` struct and `impl Querier for QuerierService` (gRPC server trait - we're a client, not server)
- `QuerierServer::new()` and gRPC server setup
- Keep: proto module definitions, tracing init, HTTP server

## Protocol Overview

```
┌─────────────────┐                      ┌─────────────────┐
│  Query-Frontend │◄──── Process() ────►│  Querier Worker │
│     (Server)    │     bidi stream      │    (Client)     │
└─────────────────┘                      └─────────────────┘
        │                                        │
        │  FrontendToClient                      │
        │  - Type: GET_ID / HTTP_REQUEST /       │
        │          HTTP_REQUEST_BATCH            │
        │  - httpRequest / httpRequestBatch      │
        ├───────────────────────────────────────►│
        │                                        │
        │  ClientToFrontend                      │
        │  - clientID / features                 │
        │  - httpResponse / httpResponseBatch    │
        │◄───────────────────────────────────────┤
```

## Implementation Phases

### Phase 1: Proto Compilation & Config
**Files:** `build.rs`, `src/config.rs`, `src/error.rs`, `Cargo.toml`

1. Create Rust-compatible proto files (strip gogoproto options):
   - `httpgrpc.proto` → HTTPRequest, HTTPResponse, Header
   - `frontend.proto` → Frontend service, FrontendToClient, ClientToFrontend, Type enum

2. Extend `build.rs`:
```rust
// Compile httpgrpc and frontend protos with client generation
tonic_build::configure()
    .build_server(false)
    .build_client(true)
    .compile_protos(&["httpgrpc.proto", "frontend.proto"], &[...])?;
```

3. Create `src/config.rs`:
```rust
pub struct WorkerConfig {
    pub frontend_address: String,     // e.g., "query-frontend:9095"
    pub parallelism: usize,           // default: 2
    pub querier_id: Option<String>,   // default: hostname
    pub max_recv_msg_size: usize,     // default: 100MB
    pub max_send_msg_size: usize,     // default: 16MB
}
```

4. Add dependencies to `Cargo.toml`:
```toml
backoff = "0.4"
futures = { workspace = true }
```

**Tests:** Config parsing, proto compilation verification

### Phase 2: Request Handler
**Files:** `src/request_handler.rs`, `src/http.rs`

1. Create `RequestHandler` that converts between httpgrpc and axum:
```rust
pub struct RequestHandler {
    router: Router,
}

impl RequestHandler {
    pub async fn handle(&self, request: HttpRequest) -> HttpResponse {
        // 1. Convert HttpRequest -> axum::http::Request
        // 2. Call router as tower::Service
        // 3. Convert axum::http::Response -> HttpResponse
    }
}
```

2. Key conversions:
   - `HttpRequest.method` → `http::Method`
   - `HttpRequest.url` → URI path + query string
   - `HttpRequest.headers` → `http::HeaderMap`
   - `HttpRequest.body` → `axum::body::Body`

**Tests:** Request/response conversion, header handling

### Phase 3: Frontend Processor
**Files:** `src/worker/frontend_processor.rs`

1. Implement the core streaming loop:
```rust
impl FrontendProcessor {
    pub async fn process_queries_on_single_stream(&self, channel: Channel, address: &str) {
        let mut client = FrontendClient::new(channel);
        let backoff = ExponentialBackoff::default();

        loop {
            match client.process().await {
                Ok(stream) => {
                    if let Err(e) = self.process_stream(stream).await {
                        // Log and retry with backoff
                    }
                    backoff.reset();
                }
                Err(e) => {
                    // Wait and retry
                    tokio::time::sleep(backoff.next_backoff()).await;
                }
            }
        }
    }

    async fn process_stream(&self, mut stream: Streaming<FrontendToClient>) -> Result<()> {
        while let Some(msg) = stream.message().await? {
            match msg.r#type() {
                Type::GetId => {
                    // Send ClientToFrontend with client_id and features
                }
                Type::HttpRequest => {
                    // Execute request, send response
                    let resp = self.request_handler.handle(msg.http_request).await;
                    stream.send(ClientToFrontend { http_response: resp, .. }).await?;
                }
                Type::HttpRequestBatch => {
                    // Execute all in parallel, send batch response
                    let responses = futures::future::join_all(
                        msg.http_request_batch.into_iter()
                            .map(|req| self.request_handler.handle(req))
                    ).await;
                    stream.send(ClientToFrontend { http_response_batch: responses, .. }).await?;
                }
            }
        }
        Ok(())
    }
}
```

**Tests:** Mock gRPC stream, message type handling

### Phase 4: Processor Manager
**Files:** `src/worker/processor_manager.rs`

1. Manages N concurrent processor tasks per connection:
```rust
pub struct ProcessorManager {
    processor: Arc<FrontendProcessor>,
    channel: Channel,
    tasks: Vec<JoinHandle<()>>,
    shutdown_tx: broadcast::Sender<()>,
}

impl ProcessorManager {
    pub async fn set_concurrency(&mut self, n: usize) {
        // Spawn new tasks if n > current
        // Cancel excess tasks if n < current
    }

    pub async fn stop(&mut self) {
        // Send shutdown signal
        // Notify frontend of graceful shutdown
        // Wait for all tasks
    }
}
```

**Tests:** Concurrency adjustment, graceful shutdown

### Phase 5: Worker Service
**Files:** `src/worker/worker.rs`, `src/worker/mod.rs`

1. Main worker that discovers frontends and manages connections:
```rust
pub struct QuerierWorker {
    config: WorkerConfig,
    request_handler: Arc<RequestHandler>,
    managers: HashMap<String, ProcessorManager>,
}

impl QuerierWorker {
    pub async fn run(&mut self) -> Result<()> {
        // For now: single address from config
        // Future: DNS-based discovery
        let channel = self.connect(&self.config.frontend_address).await?;
        let mut manager = ProcessorManager::new(...);
        manager.set_concurrency(self.config.parallelism).await;

        // Wait for shutdown signal
        tokio::signal::ctrl_c().await?;
        manager.stop().await;
        Ok(())
    }
}
```

**Tests:** Connection lifecycle, shutdown handling

### Phase 6: Main Integration
**Files:** `src/main.rs`, `src/lib.rs`

1. Rewrite `main.rs`:
```rust
#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt().init();

    // Load config (from file, env, or defaults)
    let config = WorkerConfig::from_env()?;

    // Create HTTP router for local request execution
    let router = http::create_router();
    let request_handler = Arc::new(RequestHandler::new(router));

    // Start worker
    let mut worker = QuerierWorker::new(config, request_handler);

    // Optionally keep HTTP server for health/metrics
    let http_server = axum::serve(...);

    tokio::select! {
        result = worker.run() => result?,
        result = http_server => result?,
    }

    Ok(())
}
```

2. Remove:
   - `QuerierService` struct
   - `impl Querier for QuerierService`
   - gRPC server setup (`Server::builder()...`)

**Tests:** End-to-end with mock frontend

## Module Structure

```
crates/querier/src/
├── main.rs              # Entry point (rewritten)
├── lib.rs               # Library exports (new)
├── config.rs            # Configuration (new)
├── error.rs             # Error types (new)
├── http.rs              # HTTP handlers (existing, keep)
├── request_handler.rs   # HTTP<->gRPC conversion (new)
└── worker/
    ├── mod.rs
    ├── worker.rs              # Main worker service
    ├── processor_manager.rs   # Task management per connection
    └── frontend_processor.rs  # Streaming handler
```

## Dependencies to Add

```toml
[dependencies]
# Async/Futures
futures = { workspace = true }

# Backoff for retries
backoff = "0.4"

# HTTP types for conversion
http = "1.0"
http-body-util = "0.1"
bytes = { workspace = true }

[build-dependencies]
tonic-build = { workspace = true }
```

## Test Strategy

1. **Unit tests** in each module for isolated functionality
2. **Integration tests** with mock Frontend gRPC server
3. Tests run alongside implementation (not as separate phase)

## Important Constraints

1. **Stub all query execution** - All HTTP handlers will return "not implemented" errors. This implementation focuses on the worker infrastructure shape only, not actual query execution.

2. **No DataFusion/storage integration** - Do not integrate with `context`, `storage`, `traceql` crates. Keep the existing stub handlers in `http.rs`.

## Notes

- Proto files need gogoproto options stripped (create local copies in `crates/querier/proto/`)
- HTTP handlers in `http.rs` already return `NotImplemented` - keep them as-is
- DNS-based frontend discovery is future enhancement (start with single address)
