// Import from lib
use std::path::PathBuf;
use tempo_querier::{http, QuerierWorker, WorkerConfig};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize structured logging with timestamps and log levels
    tracing_subscriber::fmt()
        .with_target(true)
        .with_thread_ids(true)
        .with_line_number(true)
        .init();

    tracing::info!("Starting Tempo Querier in worker mode");

    // Load configuration from environment
    let config = WorkerConfig::from_env()?;

    // Load data path from environment or use default
    let data_path = std::env::var("DATA_PATH")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from("./data.parquet"));

    tracing::info!(
        frontend_address = %config.frontend_address,
        querier_id = %config.querier_id,
        parallelism = config.parallelism,
        data_path = %data_path.display(),
        "Worker configuration loaded"
    );

    // Create worker
    let mut worker = QuerierWorker::new(config, data_path);

    // Create HTTP server for metrics endpoint
    let http_addr: std::net::SocketAddr = "0.0.0.0:3100".parse()?;
    tracing::info!(address = %http_addr, "Starting HTTP server for metrics");

    let router = http::create_router();
    let http_server = axum::serve(
        tokio::net::TcpListener::bind(&http_addr).await?,
        router.into_make_service(),
    );

    // Run both worker and HTTP server
    tokio::select! {
        result = worker.run() => {
            if let Err(e) = result {
                tracing::error!(error = %e, "Worker error");
                return Err(e.into());
            }
        }
        result = http_server => {
            if let Err(e) = result {
                tracing::error!(error = %e, "HTTP server error");
                return Err(e.into());
            }
        }
    }

    tracing::info!("Tempo Querier shutting down");
    Ok(())
}
