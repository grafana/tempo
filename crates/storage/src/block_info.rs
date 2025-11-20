/// Information about a Tempo block for context creation
#[derive(Debug, Clone)]
pub struct BlockInfo {
    /// The block ID (GUID)
    pub block_id: String,
    /// The tenant ID
    pub tenant_id: String,
}

impl BlockInfo {
    /// Create a new BlockInfo
    pub fn new(block_id: impl Into<String>, tenant_id: impl Into<String>) -> Self {
        Self {
            block_id: block_id.into(),
            tenant_id: tenant_id.into(),
        }
    }

    /// Get the object store path to the data.parquet file for this block
    /// Path format: <tenant_id>/<block_id>/data.parquet
    pub fn data_parquet_path(&self) -> String {
        format!("{}/{}/data.parquet", self.tenant_id, self.block_id)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_block_info_path() {
        let block_info = BlockInfo::new("test-block-id", "test-tenant");
        assert_eq!(
            block_info.data_parquet_path(),
            "test-tenant/test-block-id/data.parquet"
        );
    }
}
