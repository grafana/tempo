use config::S3Config;
use datafusion::error::{DataFusionError, Result};
use object_store::aws::AmazonS3Builder;
use object_store::ClientOptions;
use object_store::ObjectStore;
use std::sync::Arc;
use std::time::Duration;

/// Create an S3-compatible object store from configuration
///
/// Parameters:
/// - config: S3 configuration containing endpoint, bucket, credentials, etc.
///
/// Returns:
/// - Arc-wrapped ObjectStore configured according to the S3Config
pub fn create_object_store(config: &S3Config) -> Result<Arc<dyn ObjectStore>> {
    // Configure HTTP client options with connection pool settings
    let client_options = ClientOptions::new()
        .with_pool_max_idle_per_host(config.pool_max_idle_per_host)
        .with_pool_idle_timeout(Duration::from_secs(config.pool_idle_timeout_secs));

    let s3_builder = if config.use_env_credentials {
        // Use AWS environment credential chain (env vars, instance profiles, etc.)
        println!("Using AWS environment credential chain");
        let mut builder = AmazonS3Builder::from_env()
            .with_bucket_name(&config.bucket)
            .with_region(&config.region)
            .with_client_options(client_options);

        // Only set endpoint if explicitly provided (AWS uses default regional endpoints)
        if !config.endpoint.is_empty() {
            builder = builder.with_endpoint(&config.endpoint);
        }

        if config.allow_http {
            builder = builder.with_allow_http(true);
        }

        // Add session token if provided
        if let Some(token) = &config.session_token {
            builder = builder.with_token(token);
        }

        builder
    } else {
        // Use explicitly provided credentials
        let mut builder = AmazonS3Builder::new()
            .with_endpoint(&config.endpoint)
            .with_bucket_name(&config.bucket)
            .with_region(&config.region)
            .with_access_key_id(&config.access_key_id)
            .with_secret_access_key(&config.secret_access_key)
            .with_client_options(client_options);

        if config.allow_http {
            builder = builder.with_allow_http(true);
        }

        // Add session token if provided
        if let Some(token) = &config.session_token {
            builder = builder.with_token(token);
        }

        builder
    };

    let s3_store: Arc<dyn ObjectStore> = Arc::new(s3_builder.build().map_err(|e| {
        DataFusionError::External(format!("Failed to build S3 object store: {}", e).into())
    })?);

    Ok(s3_store)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_object_store_with_explicit_credentials() {
        let config = S3Config {
            endpoint: "http://localhost:9000".to_string(),
            bucket: "test-bucket".to_string(),
            prefix: "test-prefix".to_string(),
            region: "us-east-1".to_string(),
            access_key_id: "test-key".to_string(),
            secret_access_key: "test-secret".to_string(),
            session_token: None,
            allow_http: true,
            use_env_credentials: false,
            pool_max_idle_per_host: 30,
            pool_idle_timeout_secs: 120,
            cutoff_hours: 24,
        };

        // This should not panic during builder construction
        let result = create_object_store(&config);
        assert!(result.is_ok());
    }

    #[test]
    fn test_create_object_store_with_env_credentials() {
        let config = S3Config {
            endpoint: String::new(),
            bucket: "test-bucket".to_string(),
            prefix: "test-prefix".to_string(),
            region: "us-east-1".to_string(),
            access_key_id: String::new(),
            secret_access_key: String::new(),
            session_token: None,
            allow_http: false,
            use_env_credentials: true,
            pool_max_idle_per_host: 30,
            pool_idle_timeout_secs: 120,
            cutoff_hours: 24,
        };

        // This should not panic during builder construction
        let result = create_object_store(&config);
        assert!(result.is_ok());
    }
}
