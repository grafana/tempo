use chrono::{DateTime, Duration, Utc};
use datafusion::error::{DataFusionError, Result};
use object_store::path::Path as ObjectPath;
use object_store::ObjectStore;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::{debug, info, warn};

#[derive(Debug, Deserialize, Serialize)]
struct BlockMeta {
    #[serde(rename = "blockID")]
    block_id: String,
    #[serde(rename = "startTime")]
    start_time: String,
    #[serde(rename = "endTime")]
    end_time: String,
    #[serde(flatten)]
    _other: serde_json::Value,
}

#[derive(Debug, Clone)]
pub struct DiscoveredBlock {
    pub path: ObjectPath,
    pub size: u64,
    pub start_time: String,
    pub end_time: String,
}

/// TempoStorage - Handles S3 block discovery and metadata extraction for Tempo traces
#[derive(Debug)]
pub struct TempoStorage {
    object_store: Arc<dyn ObjectStore>,
    prefix: String,
    cutoff_hours: i64,
}

impl TempoStorage {
    /// Create a new TempoStorage instance
    pub fn new(object_store: Arc<dyn ObjectStore>, prefix: String, cutoff_hours: i64) -> Self {
        Self {
            object_store,
            prefix,
            cutoff_hours,
        }
    }

    /// Discover blocks by directly searching for data.parquet files and matching meta files
    ///
    /// Returns:
    /// - Vector of DiscoveredBlock structs containing path, size, and time range information
    pub async fn discover_blocks(&self) -> Result<Vec<DiscoveredBlock>> {
        use futures_util::stream::{StreamExt, TryStreamExt};

        // List all objects under the prefix
        let list_result = self
            .object_store
            .list(Some(&ObjectPath::from(self.prefix.as_str())));

        // Collect all paths into a vector
        let all_objects: Vec<_> = list_result.try_collect().await.map_err(|e| {
            DataFusionError::External(format!("Failed to list objects in S3: {}", e).into())
        })?;

        debug!(
            "Found {} total objects under prefix '{}'",
            all_objects.len(),
            self.prefix
        );

        // Find all data.parquet files
        let parquet_files: Vec<_> = all_objects
            .iter()
            .filter(|obj| obj.location.as_ref().ends_with("/data.parquet"))
            .collect();

        info!("Found {} data.parquet files", parquet_files.len());

        // Process metadata files in parallel with concurrency limit of 20
        let results: Vec<_> = futures_util::stream::iter(parquet_files)
            .map(|parquet_obj| {
                let object_store = Arc::clone(&self.object_store);
                let all_objects_ref = &all_objects;
                let parquet_path = parquet_obj.location.clone();
                let parquet_size = parquet_obj.size;

                async move {
                    // Extract the block prefix (everything before /data.parquet)
                    let block_prefix = match parquet_path.as_ref().strip_suffix("/data.parquet") {
                        Some(prefix) => prefix,
                        None => {
                            return Err(format!(
                                "Failed to extract block prefix from path: {}",
                                parquet_path
                            ));
                        }
                    };

                    // Try to find meta.json or meta.compacted.json
                    let meta_path = match find_meta_file(all_objects_ref, block_prefix) {
                        Ok(path) => path,
                        Err(_) => {
                            return Err(format!(
                                "No meta file found for block prefix: {}",
                                block_prefix
                            ));
                        }
                    };

                    // Read and parse the meta file
                    let block_meta = match read_meta_file(&object_store, &meta_path).await {
                        Ok(meta) => meta,
                        Err(e) => {
                            return Err(format!("Failed to read meta file {}: {}", meta_path, e));
                        }
                    };

                    Ok(DiscoveredBlock {
                        path: parquet_path,
                        size: parquet_size,
                        start_time: block_meta.start_time,
                        end_time: block_meta.end_time,
                    })
                }
            })
            .buffer_unordered(20) // Process up to 20 metadata files concurrently
            .collect()
            .await;

        // Separate successful blocks from errors
        let mut blocks = Vec::new();
        let mut skipped_blocks = 0;

        for result in results {
            match result {
                Ok(block) => blocks.push(block),
                Err(e) => {
                    warn!("{}, skipping block", e);
                    skipped_blocks += 1;
                }
            }
        }

        info!("Successfully matched {} blocks with metadata", blocks.len());
        if skipped_blocks > 0 {
            warn!(
                "Skipped {} blocks due to missing or invalid meta files",
                skipped_blocks
            );
        }

        // Filter blocks to only include those with end_time within a configured window
        let cutoff_time = Utc::now() - Duration::hours(self.cutoff_hours);
        let blocks_before_filter = blocks.len();

        blocks.retain(|block| {
            match DateTime::parse_from_rfc3339(&block.end_time) {
                Ok(end_time) => {
                    let end_time_utc = end_time.with_timezone(&Utc);
                    if end_time_utc < cutoff_time {
                        debug!(
                            "Filtering out block {} with end_time {}",
                            block.path, block.end_time
                        );
                        false
                    } else {
                        true
                    }
                }
                Err(e) => {
                    warn!(
                        "Failed to parse end_time '{}' for block {}: {}, keeping block",
                        block.end_time, block.path, e
                    );
                    true // Keep blocks with unparseable timestamps to be safe
                }
            }
        });

        let filtered_count = blocks_before_filter - blocks.len();
        if filtered_count > 0 {
            info!(
                "Filtered out {} blocks older than cutoff: {}",
                filtered_count,
                cutoff_time.to_rfc3339()
            );
        }

        // Display time range information for discovered blocks
        if !blocks.is_empty() {
            // Find min and max times
            let mut earliest_start: Option<&str> = None;
            let mut latest_end: Option<&str> = None;

            for block in &blocks {
                if earliest_start.is_none() || block.start_time.as_str() < earliest_start.unwrap() {
                    earliest_start = Some(&block.start_time);
                }
                if latest_end.is_none() || block.end_time.as_str() > latest_end.unwrap() {
                    latest_end = Some(&block.end_time);
                }
            }

            if let (Some(start), Some(end)) = (earliest_start, latest_end) {
                info!("Block time range: {} to {}", start, end);
            }
        }

        Ok(blocks)
    }

    /// Get a reference to the object store
    pub fn object_store(&self) -> &Arc<dyn ObjectStore> {
        &self.object_store
    }
}

/// Find the meta.json or meta.compacted.json file for a given block prefix
fn find_meta_file(
    all_objects: &[object_store::ObjectMeta],
    block_prefix: &str,
) -> Result<ObjectPath> {
    // Look for meta.json first, then meta.compacted.json
    for meta_suffix in &["/meta.json", "/meta.compacted.json"] {
        let expected_path = format!("{}{}", block_prefix, meta_suffix);
        if all_objects
            .iter()
            .any(|obj| obj.location.as_ref() == expected_path)
        {
            return Ok(ObjectPath::from(expected_path));
        }
    }

    Err(DataFusionError::Execution(format!(
        "No meta file found for block prefix: {}",
        block_prefix
    )))
}

/// Read and parse a meta.json or meta.compacted.json file
async fn read_meta_file(
    s3_store: &Arc<dyn ObjectStore>,
    meta_path: &ObjectPath,
) -> Result<BlockMeta> {
    let get_result = s3_store.get(meta_path).await.map_err(|e| {
        DataFusionError::External(format!("Failed to fetch meta file from S3: {}", e).into())
    })?;

    let meta_bytes = get_result.bytes().await.map_err(|e| {
        DataFusionError::External(format!("Failed to read bytes from meta file: {}", e).into())
    })?;

    let block_meta: BlockMeta = serde_json::from_slice(&meta_bytes).map_err(|e| {
        DataFusionError::External(format!("Failed to parse meta file: {}", e).into())
    })?;

    Ok(block_meta)
}
