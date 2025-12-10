use std::env;

/// Configuration for the querier worker
#[derive(Debug, Clone)]
pub struct WorkerConfig {
    /// Address of the query-frontend to connect to (e.g., "query-frontend:9095")
    pub frontend_address: String,

    /// Number of concurrent processor tasks per connection
    pub parallelism: usize,

    /// Unique identifier for this querier instance
    /// Defaults to hostname if not specified
    pub querier_id: String,

    /// Maximum message size for receiving (in bytes)
    pub max_recv_msg_size: usize,

    /// Maximum message size for sending (in bytes)
    pub max_send_msg_size: usize,
}

impl Default for WorkerConfig {
    fn default() -> Self {
        let hostname = hostname::get()
            .ok()
            .and_then(|h| h.into_string().ok())
            .unwrap_or_else(|| "unknown-querier".to_string());

        Self {
            frontend_address: "localhost:9095".to_string(),
            parallelism: 2,
            querier_id: hostname,
            max_recv_msg_size: 100 * 1024 * 1024, // 100MB
            max_send_msg_size: 16 * 1024 * 1024,  // 16MB
        }
    }
}

impl WorkerConfig {
    /// Create a new WorkerConfig from environment variables
    pub fn from_env() -> Result<Self, crate::error::QuerierError> {
        let mut config = Self::default();

        if let Ok(addr) = env::var("FRONTEND_ADDRESS") {
            config.frontend_address = addr;
        }

        if let Ok(parallelism) = env::var("QUERIER_PARALLELISM") {
            config.parallelism = parallelism.parse().map_err(|e| {
                crate::error::QuerierError::Config(format!("Invalid parallelism: {}", e))
            })?;
        }

        if let Ok(id) = env::var("QUERIER_ID") {
            config.querier_id = id;
        }

        if let Ok(size) = env::var("MAX_RECV_MSG_SIZE") {
            config.max_recv_msg_size = size.parse().map_err(|e| {
                crate::error::QuerierError::Config(format!("Invalid max_recv_msg_size: {}", e))
            })?;
        }

        if let Ok(size) = env::var("MAX_SEND_MSG_SIZE") {
            config.max_send_msg_size = size.parse().map_err(|e| {
                crate::error::QuerierError::Config(format!("Invalid max_send_msg_size: {}", e))
            })?;
        }

        Ok(config)
    }

    /// Validate the configuration
    pub fn validate(&self) -> Result<(), crate::error::QuerierError> {
        if self.frontend_address.is_empty() {
            return Err(crate::error::QuerierError::Config(
                "frontend_address cannot be empty".to_string(),
            ));
        }

        if self.parallelism == 0 {
            return Err(crate::error::QuerierError::Config(
                "parallelism must be greater than 0".to_string(),
            ));
        }

        if self.querier_id.is_empty() {
            return Err(crate::error::QuerierError::Config(
                "querier_id cannot be empty".to_string(),
            ));
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = WorkerConfig::default();
        assert_eq!(config.parallelism, 2);
        assert_eq!(config.max_recv_msg_size, 100 * 1024 * 1024);
        assert_eq!(config.max_send_msg_size, 16 * 1024 * 1024);
        assert!(!config.querier_id.is_empty());
    }

    #[test]
    fn test_validate_empty_frontend_address() {
        let mut config = WorkerConfig::default();
        config.frontend_address = String::new();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_validate_zero_parallelism() {
        let mut config = WorkerConfig::default();
        config.parallelism = 0;
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_validate_valid_config() {
        let config = WorkerConfig::default();
        assert!(config.validate().is_ok());
    }
}
