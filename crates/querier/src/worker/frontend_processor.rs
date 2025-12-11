use backoff::{backoff::Backoff, ExponentialBackoffBuilder};
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
use tonic::transport::Channel;
use tonic::Streaming;

use crate::error::{QuerierError, Result};
use crate::frontend::{
    frontend_client::FrontendClient, ClientToFrontend, Feature, FrontendToClient, Type,
};
use crate::httpgrpc::{Header, HttpResponse};
use crate::worker::query_executor::QueryExecutor;

/// Processor that handles bidirectional streaming with the query-frontend
#[derive(Clone)]
pub struct FrontendProcessor {
    querier_id: String,
    query_executor: Arc<QueryExecutor>,
}

impl FrontendProcessor {
    /// Create a new FrontendProcessor
    pub fn new(querier_id: String, data_path: PathBuf) -> Self {
        Self {
            querier_id,
            query_executor: Arc::new(QueryExecutor::new(data_path)),
        }
    }

    /// Process queries on a single stream with automatic retry
    pub async fn process_queries_on_single_stream(
        &self,
        channel: Channel,
        address: &str,
        mut shutdown_rx: tokio::sync::broadcast::Receiver<()>,
    ) -> Result<()> {
        let mut backoff = ExponentialBackoffBuilder::new()
            .with_initial_interval(Duration::from_millis(100))
            .with_max_interval(Duration::from_secs(10))
            .with_max_elapsed_time(None) // Retry indefinitely
            .build();

        loop {
            // Check for shutdown signal
            if shutdown_rx.try_recv().is_ok() {
                tracing::info!("Shutdown signal received, stopping stream processor");
                return Ok(());
            }

            let mut client = FrontendClient::new(channel.clone());

            tracing::info!(
                address = %address,
                querier_id = %self.querier_id,
                "Connecting to query-frontend"
            );

            // Create channel for outbound messages
            let (outbound_tx, outbound_rx) = tokio::sync::mpsc::channel(100);
            let outbound_stream = tokio_stream::wrappers::ReceiverStream::new(outbound_rx);

            match client.process(outbound_stream).await {
                Ok(response) => {
                    tracing::info!(
                        "Successfully connected to query-frontend, starting stream processing"
                    );

                    // Reset backoff on successful connection
                    backoff.reset();

                    let inbound_stream = response.into_inner();

                    // Process the stream
                    match self
                        .process_stream(outbound_tx, inbound_stream, &mut shutdown_rx)
                        .await
                    {
                        Ok(()) => {
                            tracing::info!("Stream processing completed normally");
                            return Ok(());
                        }
                        Err(e) => {
                            tracing::warn!(
                                error = %e,
                                "Stream processing failed, will retry"
                            );
                        }
                    }
                }
                Err(e) => {
                    tracing::warn!(
                        error = %e,
                        "Failed to connect to query-frontend"
                    );
                }
            }

            // Wait before retrying
            if let Some(duration) = backoff.next_backoff() {
                tracing::info!(
                    wait_seconds = duration.as_secs_f64(),
                    "Waiting before retry"
                );

                tokio::select! {
                    _ = tokio::time::sleep(duration) => {},
                    _ = shutdown_rx.recv() => {
                        tracing::info!("Shutdown signal received during backoff");
                        return Ok(());
                    }
                }
            }
        }
    }

    /// Process a single bidirectional stream
    async fn process_stream(
        &self,
        outbound_tx: tokio::sync::mpsc::Sender<ClientToFrontend>,
        mut inbound_stream: Streaming<FrontendToClient>,
        shutdown_rx: &mut tokio::sync::broadcast::Receiver<()>,
    ) -> Result<()> {
        loop {
            tokio::select! {
                // Check for shutdown
                _ = shutdown_rx.recv() => {
                    tracing::info!("Shutdown signal received in stream processor");
                    return Ok(());
                }
                // Receive message from frontend
                msg = inbound_stream.message() => {
                    match msg {
                        Ok(Some(msg)) => {
                            if let Err(e) = self.handle_frontend_message(msg, &outbound_tx).await {
                                tracing::error!(error = %e, "Failed to handle frontend message");
                                return Err(e);
                            }
                        }
                        Ok(None) => {
                            tracing::info!("Stream closed by frontend");
                            return Ok(());
                        }
                        Err(e) => {
                            tracing::error!(error = %e, "Stream error");
                            return Err(QuerierError::StreamProcessing(e.to_string()));
                        }
                    }
                }
            }
        }
    }

    /// Handle a message from the frontend
    async fn handle_frontend_message(
        &self,
        msg: FrontendToClient,
        response_tx: &tokio::sync::mpsc::Sender<ClientToFrontend>,
    ) -> Result<()> {
        let msg_type = Type::try_from(msg.r#type).map_err(|_| {
            QuerierError::StreamProcessing(format!("Invalid message type: {}", msg.r#type))
        })?;

        match msg_type {
            Type::GetId => {
                tracing::debug!("Received GET_ID request");
                self.handle_get_id(response_tx).await?;
            }
            Type::HttpRequest => {
                tracing::debug!("Received HTTP_REQUEST");
                if let Some(request) = msg.http_request {
                    self.handle_http_request(request, response_tx).await?;
                } else {
                    tracing::warn!("Received HTTP_REQUEST without request body");
                }
            }
            Type::HttpRequestBatch => {
                tracing::debug!(
                    batch_size = msg.http_request_batch.len(),
                    "Received HTTP_REQUEST_BATCH"
                );
                self.handle_http_request_batch(msg.http_request_batch, response_tx)
                    .await?;
            }
        }

        Ok(())
    }

    /// Handle GET_ID request by sending querier ID and features
    async fn handle_get_id(
        &self,
        response_tx: &tokio::sync::mpsc::Sender<ClientToFrontend>,
    ) -> Result<()> {
        let response = ClientToFrontend {
            client_id: self.querier_id.clone(),
            features: Feature::RequestBatching as i32,
            http_response: None,
            http_response_batch: vec![],
        };

        response_tx.send(response).await.map_err(|e| {
            QuerierError::StreamProcessing(format!("Failed to send GET_ID response: {}", e))
        })?;

        Ok(())
    }

    /// Handle a single HTTP request
    async fn handle_http_request(
        &self,
        request: crate::httpgrpc::HttpRequest,
        response_tx: &tokio::sync::mpsc::Sender<ClientToFrontend>,
    ) -> Result<()> {
        tracing::info!(
            method = %request.method,
            url = %request.url,
            "Processing HTTP request"
        );

        // Parse URL to determine endpoint
        let full_url = if request.url.starts_with("http://") || request.url.starts_with("https://") {
            request.url.clone()
        } else {
            format!("http://localhost{}", request.url)
        };

        let url = url::Url::parse(&full_url).map_err(|e| {
            QuerierError::StreamProcessing(format!("Failed to parse URL: {}", e))
        })?;

        // Route to appropriate handler
        let http_response = match url.path() {
            "/api/search" => self.handle_search_request(&url).await,
            _ => {
                tracing::warn!(path = url.path(), "Unknown endpoint");
                Ok(create_not_found_response())
            }
        }?;

        let client_response = ClientToFrontend {
            client_id: String::new(), // Only needed for GET_ID
            features: 0,
            http_response: Some(http_response),
            http_response_batch: vec![],
        };

        response_tx.send(client_response).await.map_err(|e| {
            QuerierError::StreamProcessing(format!("Failed to send HTTP response: {}", e))
        })?;

        Ok(())
    }

    /// Handle /api/search endpoint
    async fn handle_search_request(&self, url: &url::Url) -> Result<HttpResponse> {
        use crate::worker::query_executor::SearchParams;

        // Parse query parameters
        let params = SearchParams::from_url(url);

        // Execute query
        let search_response = self.query_executor.search(&params).await?;

        // Convert to JSON manually
        let traces_json: Vec<_> = search_response
            .traces
            .iter()
            .map(|t| {
                serde_json::json!({
                    "traceID": t.trace_id,
                    "rootServiceName": t.root_service_name,
                    "rootTraceName": t.root_trace_name,
                    "startTimeUnixNano": t.start_time_unix_nano.to_string(),
                    "durationMs": t.duration_ms,
                })
            })
            .collect();

        let metrics_json = if let Some(ref metrics) = search_response.metrics {
            serde_json::json!({
                "inspectedTraces": metrics.inspected_traces,
                "inspectedBytes": metrics.inspected_bytes,
                "totalBlocks": metrics.total_blocks,
                "completedJobs": metrics.completed_jobs,
                "totalJobs": metrics.total_jobs,
            })
        } else {
            serde_json::json!({})
        };

        let response_json = serde_json::json!({
            "traces": traces_json,
            "metrics": metrics_json,
        });

        let body = serde_json::to_vec(&response_json).map_err(|e| {
            QuerierError::StreamProcessing(format!("Failed to serialize response: {}", e))
        })?;

        Ok(HttpResponse {
            code: 200,
            headers: vec![Header {
                key: "content-type".to_string(),
                values: vec!["application/json".to_string()],
            }],
            body,
        })
    }

    /// Handle a batch of HTTP requests
    async fn handle_http_request_batch(
        &self,
        requests: Vec<crate::httpgrpc::HttpRequest>,
        response_tx: &tokio::sync::mpsc::Sender<ClientToFrontend>,
    ) -> Result<()> {
        tracing::info!(count = requests.len(), "Processing HTTP request batch");

        // Process all requests in parallel
        let mut handles = Vec::new();
        for request in requests {
            let processor = self.clone();
            let handle = tokio::spawn(async move {
                // Parse URL
                let full_url = if request.url.starts_with("http://") || request.url.starts_with("https://") {
                    request.url.clone()
                } else {
                    format!("http://localhost{}", request.url)
                };

                let url = match url::Url::parse(&full_url) {
                    Ok(u) => u,
                    Err(e) => {
                        tracing::error!(error = %e, "Failed to parse URL");
                        return create_error_response(400, "Invalid URL");
                    }
                };

                // Route to appropriate handler
                match url.path() {
                    "/api/search" => processor.handle_search_request(&url).await,
                    _ => Ok(create_not_found_response()),
                }
                .unwrap_or_else(|e| {
                    tracing::error!(error = %e, "Request processing failed");
                    create_error_response(500, &format!("Internal error: {}", e))
                })
            });
            handles.push(handle);
        }

        // Wait for all requests to complete
        let mut responses = Vec::new();
        for handle in handles {
            match handle.await {
                Ok(response) => responses.push(response),
                Err(e) => {
                    tracing::error!(error = %e, "Task join error");
                    responses.push(create_error_response(500, "Internal error"));
                }
            }
        }

        let client_response = ClientToFrontend {
            client_id: String::new(),
            features: 0,
            http_response: None,
            http_response_batch: responses,
        };

        response_tx.send(client_response).await.map_err(|e| {
            QuerierError::StreamProcessing(format!("Failed to send batch response: {}", e))
        })?;

        Ok(())
    }
}

/// Create a "Not Implemented" HTTP response
fn create_not_implemented_response() -> HttpResponse {
    let error_body = serde_json::json!({
        "error": "Query execution not yet implemented"
    });

    HttpResponse {
        code: 501, // Not Implemented
        headers: vec![Header {
            key: "content-type".to_string(),
            values: vec!["application/json".to_string()],
        }],
        body: serde_json::to_vec(&error_body).unwrap_or_default(),
    }
}

/// Create a "Not Found" HTTP response
fn create_not_found_response() -> HttpResponse {
    let error_body = serde_json::json!({
        "error": "Endpoint not found"
    });

    HttpResponse {
        code: 404,
        headers: vec![Header {
            key: "content-type".to_string(),
            values: vec!["application/json".to_string()],
        }],
        body: serde_json::to_vec(&error_body).unwrap_or_default(),
    }
}

/// Create a generic error HTTP response
fn create_error_response(code: i32, message: &str) -> HttpResponse {
    let error_body = serde_json::json!({
        "error": message
    });

    HttpResponse {
        code,
        headers: vec![Header {
            key: "content-type".to_string(),
            values: vec!["application/json".to_string()],
        }],
        body: serde_json::to_vec(&error_body).unwrap_or_default(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_processor_creation() {
        let data_path = PathBuf::from("test.parquet");
        let processor = FrontendProcessor::new("test-querier".to_string(), data_path);
        assert_eq!(processor.querier_id, "test-querier");
    }

    #[test]
    fn test_not_implemented_response() {
        let response = create_not_implemented_response();
        assert_eq!(response.code, 501);
        assert!(!response.body.is_empty());
    }

    #[test]
    fn test_not_found_response() {
        let response = create_not_found_response();
        assert_eq!(response.code, 404);
        assert!(!response.body.is_empty());

        // Parse body as JSON
        let body: serde_json::Value = serde_json::from_slice(&response.body).unwrap();
        assert_eq!(body["error"], "Endpoint not found");
    }

    #[test]
    fn test_error_response() {
        let response = create_error_response(500, "Test error message");
        assert_eq!(response.code, 500);

        let body: serde_json::Value = serde_json::from_slice(&response.body).unwrap();
        assert_eq!(body["error"], "Test error message");
    }
}
