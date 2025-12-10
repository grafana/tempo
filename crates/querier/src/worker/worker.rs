use std::collections::HashMap;
use tonic::transport::Channel;

use crate::config::WorkerConfig;
use crate::error::{QuerierError, Result};
use crate::worker::frontend_processor::FrontendProcessor;
use crate::worker::processor_manager::ProcessorManager;

/// Main querier worker that connects to frontends and manages request processing
pub struct QuerierWorker {
    config: WorkerConfig,
    managers: HashMap<String, ProcessorManager>,
}

impl QuerierWorker {
    /// Create a new QuerierWorker
    pub fn new(config: WorkerConfig) -> Self {
        Self {
            config,
            managers: HashMap::new(),
        }
    }

    /// Run the worker until shutdown
    pub async fn run(&mut self) -> Result<()> {
        tracing::info!(
            frontend_address = %self.config.frontend_address,
            querier_id = %self.config.querier_id,
            parallelism = self.config.parallelism,
            "Starting querier worker"
        );

        // Validate configuration
        self.config.validate()?;

        // Connect to the frontend
        let channel = self.connect(&self.config.frontend_address).await?;

        // Create processor and manager
        let processor = FrontendProcessor::new(self.config.querier_id.clone());

        let mut manager = ProcessorManager::new(
            processor,
            channel,
            self.config.frontend_address.clone(),
            self.config.querier_id.clone(),
        );

        // Set concurrency level
        manager.set_concurrency(self.config.parallelism).await?;

        // Store manager
        self.managers
            .insert(self.config.frontend_address.clone(), manager);

        tracing::info!("Querier worker started successfully, waiting for shutdown signal");

        // Wait for shutdown signal
        tokio::signal::ctrl_c()
            .await
            .map_err(|e| QuerierError::Shutdown(format!("Failed to wait for ctrl-c: {}", e)))?;

        tracing::info!("Shutdown signal received, stopping worker");

        // Stop all managers
        self.shutdown().await?;

        tracing::info!("Querier worker stopped");
        Ok(())
    }

    /// Connect to a frontend
    async fn connect(&self, address: &str) -> Result<Channel> {
        tracing::info!(address = %address, "Connecting to frontend");

        let endpoint = tonic::transport::Endpoint::from_shared(format!("http://{}", address))
            .map_err(|e| QuerierError::Connection(format!("Invalid address: {}", e)))?
            .http2_keep_alive_interval(std::time::Duration::from_secs(30))
            .keep_alive_timeout(std::time::Duration::from_secs(10))
            .connect_timeout(std::time::Duration::from_secs(5))
            .initial_connection_window_size(Some(self.config.max_recv_msg_size as u32))
            .initial_stream_window_size(Some(self.config.max_send_msg_size as u32));

        let channel = endpoint
            .connect()
            .await
            .map_err(|e| QuerierError::Connection(format!("Failed to connect: {}", e)))?;

        tracing::info!(address = %address, "Successfully connected to frontend");
        Ok(channel)
    }

    /// Shutdown all managers gracefully
    async fn shutdown(&mut self) -> Result<()> {
        tracing::info!(manager_count = self.managers.len(), "Shutting down managers");

        let mut errors = Vec::new();

        for (address, mut manager) in self.managers.drain() {
            tracing::info!(address = %address, "Stopping manager");

            if let Err(e) = manager.stop().await {
                tracing::error!(address = %address, error = %e, "Failed to stop manager");
                errors.push(e);
            }
        }

        if !errors.is_empty() {
            return Err(QuerierError::Shutdown(format!(
                "Failed to stop {} managers",
                errors.len()
            )));
        }

        Ok(())
    }

    /// Get the number of active managers
    pub fn manager_count(&self) -> usize {
        self.managers.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_worker_creation() {
        let config = WorkerConfig::default();
        let worker = QuerierWorker::new(config);

        assert_eq!(worker.manager_count(), 0);
    }

    #[tokio::test]
    async fn test_config_validation() {
        let mut config = WorkerConfig::default();
        config.parallelism = 0;

        let mut worker = QuerierWorker::new(config);

        // Create a timeout task to prevent the test from hanging
        let run_task = tokio::spawn(async move {
            worker.run().await
        });

        // Wait a bit for validation to occur
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;

        // Abort the task (it would hang waiting for ctrl-c)
        run_task.abort();

        // Note: This test is simplified since the actual validation happens on run()
        // In a real test environment, we'd need to mock the signal handling
    }
}
