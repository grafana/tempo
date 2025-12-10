use anyhow::{Context, Result};
use config_rs::{Config as ConfigBuilder, Environment, File};
use serde::{Deserialize, Serialize};
use std::path::Path;

/// Main application configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
#[derive(Default)]
pub struct Config {
    /// S3 storage configuration
    #[serde(default)]
    pub s3: S3Config,

    /// DataFusion engine configuration
    #[serde(default)]
    pub datafusion: DataFusionConfig,
}

/// Configuration for S3-compatible object storage
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct S3Config {
    /// S3 endpoint URL (e.g., "http://localhost:9000")
    /// Optional - if not provided, uses AWS SDK default endpoint for the region
    #[serde(default)]
    pub endpoint: String,

    /// S3 bucket name (e.g., "tempo")
    #[serde(default = "default_bucket")]
    pub bucket: String,

    /// Prefix/path within the bucket (e.g., "single-tenant")
    #[serde(default = "default_prefix")]
    pub prefix: String,

    /// AWS region (required but ignored by Minio)
    #[serde(default = "default_region")]
    pub region: String,

    /// S3 access key ID
    #[serde(default)]
    pub access_key_id: String,

    /// S3 secret access key
    #[serde(default)]
    pub secret_access_key: String,

    /// S3 session token (optional, for temporary credentials)
    #[serde(default)]
    pub session_token: Option<String>,

    /// Allow HTTP connections (true for Minio, false for AWS S3)
    #[serde(default)]
    pub allow_http: bool,

    /// Use AWS environment credentials via AmazonS3Builder::from_env()
    /// When true, uses AWS credential chain (env vars, instance profiles, etc.)
    #[serde(default)]
    pub use_env_credentials: bool,

    /// Maximum number of idle connections per host (default: 30)
    #[serde(default = "default_pool_max_idle_per_host")]
    pub pool_max_idle_per_host: usize,

    /// Pool idle timeout in seconds (default: 120)
    #[serde(default = "default_pool_idle_timeout")]
    pub pool_idle_timeout_secs: u64,

    /// Time cutoff in hours for filtering blocks (default: 24)
    /// Only blocks with end_time within the last N hours will be included
    #[serde(default = "default_cutoff_hours")]
    pub cutoff_hours: i64,
}

/// Configuration for DataFusion query engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataFusionConfig {
    /// Enable parquet predicate pushdown/pruning (default: true)
    #[serde(default = "default_parquet_pruning")]
    pub parquet_pruning: bool,
}

// Default value functions
fn default_bucket() -> String {
    "tempo".to_string()
}

fn default_prefix() -> String {
    "single-tenant".to_string()
}

fn default_region() -> String {
    "us-east-1".to_string()
}

fn default_pool_max_idle_per_host() -> usize {
    30
}

fn default_pool_idle_timeout() -> u64 {
    120
}

fn default_cutoff_hours() -> i64 {
    24
}

fn default_parquet_pruning() -> bool {
    true
}

impl Default for S3Config {
    fn default() -> Self {
        Self {
            endpoint: "http://localhost:9000".to_string(),
            bucket: default_bucket(),
            prefix: default_prefix(),
            region: default_region(),
            access_key_id: "tempo".to_string(),
            secret_access_key: "supersecret".to_string(),
            session_token: None,
            allow_http: true,
            use_env_credentials: false,
            pool_max_idle_per_host: default_pool_max_idle_per_host(),
            pool_idle_timeout_secs: default_pool_idle_timeout(),
            cutoff_hours: default_cutoff_hours(),
        }
    }
}

impl Default for DataFusionConfig {
    fn default() -> Self {
        Self {
            parquet_pruning: default_parquet_pruning(),
        }
    }
}


impl Config {
    /// Load Config with layered configuration priority:
    /// 1. Default values
    /// 2. TOML file (if provided)
    /// 3. Environment variables (S3_* prefix for S3 config, DATAFUSION_* for DataFusion config)
    /// 4. Explicit AWS credentials from environment (fallback for empty credentials)
    ///
    /// This uses the config-rs crate for robust configuration management.
    pub fn load(config_file: Option<&str>) -> Result<Self> {
        let mut builder = ConfigBuilder::builder()
            // S3 defaults
            .set_default("s3.endpoint", "http://localhost:9000")?
            .set_default("s3.bucket", "tempo")?
            .set_default("s3.prefix", "single-tenant")?
            .set_default("s3.region", "us-east-1")?
            .set_default("s3.access_key_id", "tempo")?
            .set_default("s3.secret_access_key", "supersecret")?
            .set_default("s3.allow_http", true)?
            .set_default("s3.use_env_credentials", false)?
            .set_default("s3.pool_max_idle_per_host", 30)?
            .set_default("s3.pool_idle_timeout_secs", 120)?
            .set_default("s3.cutoff_hours", 24)?
            // DataFusion defaults
            .set_default("datafusion.parquet_pruning", true)?;

        // Add TOML file if provided
        if let Some(file_path) = config_file {
            let path = Path::new(file_path);
            if !path.exists() {
                anyhow::bail!("Configuration file not found: {}", path.display());
            }
            builder = builder.add_source(File::from(path));
        }

        // Add environment variables with S3_ prefix for S3 config
        builder = builder.add_source(
            Environment::with_prefix("S3")
                .separator("_")
                .try_parsing(true),
        );

        // Add environment variables with DATAFUSION_ prefix for DataFusion config
        builder = builder.add_source(
            Environment::with_prefix("DATAFUSION")
                .separator("_")
                .try_parsing(true),
        );

        let config = builder.build().context("Failed to build configuration")?;

        let mut app_config: Config = config
            .try_deserialize()
            .context("Failed to deserialize configuration")?;

        // Fallback: If credentials are still empty, try AWS_* environment variables
        // This maintains backward compatibility with AWS credential conventions
        if app_config.s3.access_key_id.is_empty() {
            if let Ok(key) = std::env::var("AWS_ACCESS_KEY_ID") {
                app_config.s3.access_key_id = key;
            }
        }

        if app_config.s3.secret_access_key.is_empty() {
            if let Ok(secret) = std::env::var("AWS_SECRET_ACCESS_KEY") {
                app_config.s3.secret_access_key = secret;
            }
        }

        if app_config.s3.session_token.is_none() {
            if let Ok(token) = std::env::var("AWS_SESSION_TOKEN") {
                app_config.s3.session_token = Some(token);
            }
        }

        Ok(app_config)
    }

    /// Load Config from a TOML file
    ///
    /// This is a convenience method that calls load() with a file path.
    /// Environment variables can still override values from the file.
    pub fn from_file<P: AsRef<Path>>(path: P) -> Result<Self> {
        Self::load(Some(path.as_ref().to_str().unwrap()))
    }

    /// Create a new Config from environment variables with defaults
    ///
    /// This is a convenience method that calls load() without a file.
    /// Uses only environment variables and defaults.
    pub fn from_env() -> Result<Self> {
        Self::load(None)
    }

    /// Validate the configuration
    pub fn validate(&self) -> Result<()> {
        self.s3.validate()?;
        self.datafusion.validate()?;
        Ok(())
    }
}

impl S3Config {
    /// Get the base URL for the S3 bucket
    /// Returns: URL like "http://localhost:9000/tempo/single-tenant"
    pub fn base_url(&self) -> String {
        format!("{}/{}/{}", self.endpoint, self.bucket, self.prefix)
    }

    /// Validate the S3 configuration
    pub fn validate(&self) -> Result<()> {
        anyhow::ensure!(!self.bucket.is_empty(), "S3 bucket cannot be empty");
        anyhow::ensure!(!self.prefix.is_empty(), "S3 prefix cannot be empty");

        // If not using env credentials, validate that credentials are provided
        if !self.use_env_credentials {
            anyhow::ensure!(
                !self.endpoint.is_empty(),
                "S3 endpoint cannot be empty (unless use_env_credentials=true)"
            );
            anyhow::ensure!(
                !self.access_key_id.is_empty(),
                "S3 access key ID cannot be empty (unless use_env_credentials=true)"
            );
            anyhow::ensure!(
                !self.secret_access_key.is_empty(),
                "S3 secret access key cannot be empty (unless use_env_credentials=true)"
            );
        }

        Ok(())
    }
}

impl DataFusionConfig {
    /// Validate the DataFusion configuration
    pub fn validate(&self) -> Result<()> {
        // Currently no validation needed for DataFusion config
        // This method exists for consistency and future extensibility
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = Config::default();
        assert_eq!(config.s3.endpoint, "http://localhost:9000");
        assert_eq!(config.s3.bucket, "tempo");
        assert_eq!(config.s3.prefix, "single-tenant");
        assert!(config.s3.allow_http);
        assert!(config.datafusion.parquet_pruning);
    }

    #[test]
    fn test_load_with_defaults() {
        // Load without file should use defaults
        let config = Config::load(None).expect("Failed to load config");
        assert_eq!(config.s3.endpoint, "http://localhost:9000");
        assert_eq!(config.s3.bucket, "tempo");
        assert_eq!(config.s3.prefix, "single-tenant");
        assert!(config.s3.allow_http);
        assert!(config.datafusion.parquet_pruning);
    }

    #[test]
    fn test_base_url() {
        let config = Config::default();
        assert_eq!(
            config.s3.base_url(),
            "http://localhost:9000/tempo/single-tenant"
        );
    }

    #[test]
    fn test_validate_valid_config() {
        let config = Config::default();
        assert!(config.validate().is_ok());
    }

    #[test]
    fn test_validate_empty_bucket() {
        let mut config = Config::default();
        config.s3.bucket = String::new();
        assert!(config.validate().is_err());
    }
}
