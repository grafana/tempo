use thiserror::Error;

/// Unified error type for the querier worker
#[derive(Debug, Error)]
pub enum QuerierError {
    /// Configuration error
    #[error("Configuration error: {0}")]
    Config(String),

    /// gRPC transport error
    #[error("gRPC transport error: {0}")]
    Transport(#[from] tonic::transport::Error),

    /// gRPC status error
    #[error("gRPC status error: {0}")]
    Status(#[from] tonic::Status),

    /// HTTP request/response conversion error
    #[error("HTTP conversion error: {0}")]
    HttpConversion(String),

    /// Worker connection error
    #[error("Worker connection error: {0}")]
    Connection(String),

    /// Stream processing error
    #[error("Stream processing error: {0}")]
    StreamProcessing(String),

    /// Query execution error
    #[error("Query execution error: {0}")]
    QueryExecution(String),

    /// Task management error
    #[error("Task management error: {0}")]
    TaskManagement(String),

    /// Shutdown error
    #[error("Shutdown error: {0}")]
    Shutdown(String),

    /// IO error
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    /// HTTP error
    #[error("HTTP error: {0}")]
    Http(#[from] http::Error),

    /// Hyper error
    #[error("Hyper error: {0}")]
    Hyper(#[from] hyper::Error),

    /// Axum error (boxed to avoid large enum variants)
    #[error("Axum error: {0}")]
    Axum(String),

    /// Generic error
    #[error("{0}")]
    Other(String),
}

/// Result type alias for QuerierError
pub type Result<T> = std::result::Result<T, QuerierError>;

impl From<Box<dyn std::error::Error + Send + Sync>> for QuerierError {
    fn from(err: Box<dyn std::error::Error + Send + Sync>) -> Self {
        QuerierError::Other(err.to_string())
    }
}

impl From<String> for QuerierError {
    fn from(err: String) -> Self {
        QuerierError::Other(err)
    }
}

impl From<&str> for QuerierError {
    fn from(err: &str) -> Self {
        QuerierError::Other(err.to_string())
    }
}
