use backoff::{backoff::Backoff, ExponentialBackoffBuilder};
use std::time::Duration;
use tonic::transport::Channel;
use tonic::Streaming;

use crate::error::{QuerierError, Result};
use crate::frontend::{
    frontend_client::FrontendClient, ClientToFrontend, Feature, FrontendToClient, Type,
};
use crate::httpgrpc::{Header, HttpResponse};

/// Processor that handles bidirectional streaming with the query-frontend
#[derive(Clone)]
pub struct FrontendProcessor {
    querier_id: String,
}

impl FrontendProcessor {
    /// Create a new FrontendProcessor
    pub fn new(querier_id: String) -> Self {
        Self { querier_id }
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
                    tracing::info!("Successfully connected to query-frontend, starting stream processing");

                    // Reset backoff on successful connection
                    backoff.reset();

                    let inbound_stream = response.into_inner();

                    // Process the stream
                    match self.process_stream(outbound_tx, inbound_stream, &mut shutdown_rx).await {
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
        let msg_type = Type::try_from(msg.r#type)
            .map_err(|_| QuerierError::StreamProcessing(format!("Invalid message type: {}", msg.r#type)))?;

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

        response_tx
            .send(response)
            .await
            .map_err(|e| QuerierError::StreamProcessing(format!("Failed to send GET_ID response: {}", e)))?;

        Ok(())
    }

    /// Handle a single HTTP request
    /// Returns "Not Implemented" as query execution is stubbed
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

        // Stub implementation - return "Not Implemented"
        let response = create_not_implemented_response();

        let client_response = ClientToFrontend {
            client_id: String::new(), // Only needed for GET_ID
            features: 0,
            http_response: Some(response),
            http_response_batch: vec![],
        };

        response_tx
            .send(client_response)
            .await
            .map_err(|e| QuerierError::StreamProcessing(format!("Failed to send HTTP response: {}", e)))?;

        Ok(())
    }

    /// Handle a batch of HTTP requests
    /// Returns "Not Implemented" for all as query execution is stubbed
    async fn handle_http_request_batch(
        &self,
        requests: Vec<crate::httpgrpc::HttpRequest>,
        response_tx: &tokio::sync::mpsc::Sender<ClientToFrontend>,
    ) -> Result<()> {
        tracing::info!(count = requests.len(), "Processing HTTP request batch");

        // Stub implementation - return "Not Implemented" for all
        let responses: Vec<_> = requests
            .iter()
            .map(|_| create_not_implemented_response())
            .collect();

        let client_response = ClientToFrontend {
            client_id: String::new(),
            features: 0,
            http_response: None,
            http_response_batch: responses,
        };

        response_tx
            .send(client_response)
            .await
            .map_err(|e| QuerierError::StreamProcessing(format!("Failed to send batch response: {}", e)))?;

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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_processor_creation() {
        let processor = FrontendProcessor::new("test-querier".to_string());
        assert_eq!(processor.querier_id, "test-querier");
    }

    #[test]
    fn test_not_implemented_response() {
        let response = create_not_implemented_response();
        assert_eq!(response.code, 501);
        assert!(!response.body.is_empty());
    }
}
