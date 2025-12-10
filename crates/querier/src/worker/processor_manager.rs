use tokio::task::JoinHandle;
use tonic::transport::Channel;

use crate::error::{QuerierError, Result};
use crate::frontend::frontend_client::FrontendClient;
use crate::worker::frontend_processor::FrontendProcessor;

/// Manages N concurrent processor tasks per connection
pub struct ProcessorManager {
    processor: FrontendProcessor,
    channel: Channel,
    address: String,
    tasks: Vec<JoinHandle<()>>,
    shutdown_tx: tokio::sync::broadcast::Sender<()>,
    #[allow(dead_code)]
    shutdown_rx: tokio::sync::broadcast::Receiver<()>,
    querier_id: String,
}

impl ProcessorManager {
    /// Create a new ProcessorManager
    pub fn new(
        processor: FrontendProcessor,
        channel: Channel,
        address: String,
        querier_id: String,
    ) -> Self {
        let (shutdown_tx, shutdown_rx) = tokio::sync::broadcast::channel(1);

        Self {
            processor,
            channel,
            address,
            tasks: Vec::new(),
            shutdown_tx,
            shutdown_rx,
            querier_id,
        }
    }

    /// Set the concurrency level by spawning or stopping tasks
    pub async fn set_concurrency(&mut self, n: usize) -> Result<()> {
        let current = self.tasks.len();

        if n > current {
            // Spawn new tasks
            let to_spawn = n - current;
            tracing::info!(
                current = current,
                target = n,
                spawning = to_spawn,
                "Increasing concurrency"
            );

            for i in 0..to_spawn {
                self.spawn_processor_task(current + i)?;
            }
        } else if n < current {
            // Stop excess tasks
            let to_stop = current - n;
            tracing::info!(
                current = current,
                target = n,
                stopping = to_stop,
                "Decreasing concurrency"
            );

            // Cancel excess tasks
            for _ in 0..to_stop {
                if let Some(handle) = self.tasks.pop() {
                    handle.abort();
                }
            }
        }

        Ok(())
    }

    /// Spawn a single processor task
    fn spawn_processor_task(&mut self, task_id: usize) -> Result<()> {
        let processor = self.processor.clone();
        let channel = self.channel.clone();
        let address = self.address.clone();
        let shutdown_rx = self.shutdown_tx.subscribe();

        let handle = tokio::spawn(async move {
            tracing::info!(
                task_id = task_id,
                address = %address,
                "Starting processor task"
            );

            if let Err(e) = processor
                .process_queries_on_single_stream(channel, &address, shutdown_rx)
                .await
            {
                tracing::error!(
                    task_id = task_id,
                    error = %e,
                    "Processor task failed"
                );
            }

            tracing::info!(task_id = task_id, "Processor task stopped");
        });

        self.tasks.push(handle);
        Ok(())
    }

    /// Stop all processor tasks and notify the frontend of graceful shutdown
    pub async fn stop(&mut self) -> Result<()> {
        tracing::info!("Stopping processor manager");

        // Notify frontend of graceful shutdown
        if let Err(e) = self.notify_frontend_shutdown().await {
            tracing::warn!(error = %e, "Failed to notify frontend of shutdown");
        }

        // Send shutdown signal to all tasks
        if let Err(e) = self.shutdown_tx.send(()) {
            tracing::warn!(error = %e, "Failed to send shutdown signal");
        }

        // Wait for all tasks to complete
        let mut failed_tasks = 0;
        for handle in self.tasks.drain(..) {
            if let Err(e) = handle.await {
                if !e.is_cancelled() {
                    tracing::error!(error = %e, "Task failed during shutdown");
                    failed_tasks += 1;
                }
            }
        }

        if failed_tasks > 0 {
            tracing::warn!(
                failed_tasks = failed_tasks,
                "Some tasks failed during shutdown"
            );
        }

        tracing::info!("Processor manager stopped");
        Ok(())
    }

    /// Notify the frontend that this querier is shutting down gracefully
    async fn notify_frontend_shutdown(&self) -> Result<()> {
        let mut client = FrontendClient::new(self.channel.clone());

        let request = crate::frontend::NotifyClientShutdownRequest {
            client_id: self.querier_id.clone(),
        };

        match client.notify_client_shutdown(request).await {
            Ok(_) => {
                tracing::info!("Successfully notified frontend of shutdown");
                Ok(())
            }
            Err(e) => {
                tracing::warn!(error = %e, "Failed to notify frontend of shutdown");
                Err(QuerierError::Status(e))
            }
        }
    }

    /// Get the current number of running tasks
    pub fn task_count(&self) -> usize {
        self.tasks.len()
    }
}

impl Drop for ProcessorManager {
    fn drop(&mut self) {
        // Abort any remaining tasks
        for handle in &self.tasks {
            handle.abort();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_processor_manager_creation() {
        let processor = FrontendProcessor::new("test-querier".to_string());

        // Create a mock channel (this won't actually connect)
        let endpoint = tonic::transport::Endpoint::from_static("http://localhost:9095");
        let channel = endpoint.connect_lazy();

        let manager = ProcessorManager::new(
            processor,
            channel,
            "localhost:9095".to_string(),
            "test-querier".to_string(),
        );

        assert_eq!(manager.task_count(), 0);
    }

    #[tokio::test]
    async fn test_set_concurrency() {
        let processor = FrontendProcessor::new("test-querier".to_string());

        let endpoint = tonic::transport::Endpoint::from_static("http://localhost:9095");
        let channel = endpoint.connect_lazy();

        let mut manager = ProcessorManager::new(
            processor,
            channel,
            "localhost:9095".to_string(),
            "test-querier".to_string(),
        );

        // Spawn tasks (they will fail to connect but that's ok for this test)
        manager.set_concurrency(3).await.unwrap();
        assert_eq!(manager.task_count(), 3);

        // Reduce concurrency
        manager.set_concurrency(1).await.unwrap();
        assert_eq!(manager.task_count(), 1);

        // Clean up
        manager.stop().await.unwrap();
    }
}
