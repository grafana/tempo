/// Async Parquet reader for vparquet4 files
use std::collections::{HashMap, HashSet};
use std::path::{Path, PathBuf};
use std::pin::Pin;
use std::sync::Arc;

use arrow::array::{
    Array, BinaryArray, Int32Array, Int64Array, ListArray, RecordBatch, StringArray, StructArray,
    UInt64Array,
};
use futures::{Stream, StreamExt};
use parquet::arrow::arrow_reader::{
    ArrowReaderMetadata, ParquetRecordBatchReaderBuilder, RowFilter,
};
use parquet::arrow::ProjectionMask;
use parquet::column::page::PageReader;
use parquet::data_type::ByteArray;
use parquet::file::metadata::{ParquetMetaData, RowGroupMetaData};
use parquet::file::reader::{FileReader, SerializedFileReader};
use parquet::schema::types::{ColumnDescriptor, SchemaDescriptor};

use crate::error::{Error, Result};
use crate::filter::{CachedDictionary, SpanFilter, SpanNamePredicate};
use crate::schema::{field_paths, Span, Spanset};

/// Pre-loaded row group metadata with cached dictionaries
#[derive(Debug)]
pub struct RowGroupCache {
    pub index: usize,
    pub metadata: RowGroupMetaData,
    pub dictionaries: HashMap<usize, CachedDictionary>,
}

/// File-level cache loaded in open()
pub struct FileCache {
    pub metadata: Arc<ParquetMetaData>,
    pub schema: Arc<SchemaDescriptor>,
    pub arrow_schema: Arc<arrow::datatypes::Schema>,
    pub row_groups: Vec<RowGroupCache>,
    pub span_name_column_index: Option<usize>,
}

/// Read options for the async reader
#[derive(Debug, Clone)]
pub struct ReadOptions {
    pub filter: Option<SpanFilter>,
    pub batch_size: usize,  // Row groups per batch
    pub parallelism: usize, // Concurrent tasks
}

impl Default for ReadOptions {
    fn default() -> Self {
        Self {
            filter: None,
            batch_size: 4,
            parallelism: num_cpus::get(),
        }
    }
}

/// Main async reader (replaces old sync VParquet4Reader)
pub struct VParquet4Reader {
    path: PathBuf,
    pub cache: Arc<FileCache>,
    options: ReadOptions,
}

pub type SpansetStream = Pin<Box<dyn Stream<Item = Result<Spanset>> + Send>>;

impl VParquet4Reader {
    /// Open a parquet file and pre-load metadata and dictionaries
    pub async fn open<P: AsRef<Path>>(path: P, options: ReadOptions) -> Result<Self> {
        let path = path.as_ref().to_path_buf();

        // Step 1: Load metadata (sync file I/O via spawn_blocking)
        let metadata = {
            let path = path.clone();
            tokio::task::spawn_blocking(move || {
                let file = std::fs::File::open(&path)?;
                ArrowReaderMetadata::load(&file, Default::default())
            })
            .await??
        };

        let parquet_metadata = metadata.metadata().clone();
        let schema = parquet_metadata.file_metadata().schema_descr_ptr();
        let arrow_schema = metadata.schema().clone();

        let span_name_column_index = schema
            .columns()
            .iter()
            .position(|c| c.path().string() == field_paths::SPAN_NAME);

        // Step 2: Pre-load dictionaries in parallel using buffer_unordered
        let row_groups = Self::load_dictionaries_parallel(
            &path,
            &parquet_metadata,
            schema.clone(),
            span_name_column_index,
            options.parallelism,
        )
        .await?;

        let cache = Arc::new(FileCache {
            metadata: parquet_metadata,
            schema,
            arrow_schema,
            row_groups,
            span_name_column_index,
        });

        Ok(Self {
            path,
            cache,
            options,
        })
    }

    /// Load dictionaries for all row groups in parallel
    async fn load_dictionaries_parallel(
        path: &Path,
        metadata: &ParquetMetaData,
        schema: Arc<SchemaDescriptor>,
        span_name_col_idx: Option<usize>,
        parallelism: usize,
    ) -> Result<Vec<RowGroupCache>> {
        let num_row_groups = metadata.num_row_groups();
        let columns_to_cache: Vec<usize> = span_name_col_idx.into_iter().collect();

        if columns_to_cache.is_empty() {
            // No columns to cache - return basic metadata only
            return Ok((0..num_row_groups)
                .map(|i| RowGroupCache {
                    index: i,
                    metadata: metadata.row_group(i).clone(),
                    dictionaries: HashMap::new(),
                })
                .collect());
        }

        // Use futures::stream for controlled concurrency
        let row_group_metas: Vec<_> = (0..num_row_groups)
            .map(|i| (i, metadata.row_group(i).clone()))
            .collect();

        let path = path.to_path_buf();

        let results: Vec<Result<RowGroupCache>> = futures::stream::iter(row_group_metas)
            .map(|(index, rg_meta)| {
                let path = path.clone();
                let schema = schema.clone();
                let columns = columns_to_cache.clone();

                // Each row group dictionary load runs as a blocking task
                tokio::task::spawn_blocking(move || {
                    let file = std::fs::File::open(&path)?;
                    let reader = SerializedFileReader::new(file)?;
                    let row_group = reader.get_row_group(index)?;

                    let mut dictionaries = HashMap::new();

                    for col_idx in columns {
                        let mut page_reader = row_group.get_column_page_reader(col_idx)?;
                        if let Some(dict) =
                            extract_dictionary(&mut page_reader, schema.column(col_idx))?
                        {
                            dictionaries.insert(col_idx, dict);
                        }
                    }

                    Ok(RowGroupCache {
                        index,
                        metadata: rg_meta,
                        dictionaries,
                    })
                })
            })
            .buffer_unordered(parallelism) // Concurrent task limit
            .map(|res| res?) // Flatten JoinError
            .collect()
            .await;

        // Collect results, preserving order by index
        let mut caches: Vec<_> = results.into_iter().collect::<Result<Vec<_>>>()?;
        caches.sort_by_key(|c| c.index);
        Ok(caches)
    }

    /// Read spansets as a stream with two-stage parallel pipeline
    pub fn read(&self) -> SpansetStream {
        let cache = Arc::clone(&self.cache);
        let options = self.options.clone();
        let path = self.path.clone();

        Box::pin(async_stream::try_stream! {
            // Stage 1: Filter row groups (statistics + dictionary, uses cached data)
            let filtered_indices = filter_row_groups(&cache, &options.filter);

            // Stage 2: Read filtered row groups with true parallel concurrency
            let mut stream = futures::stream::iter(filtered_indices)
                .map(move |rg_idx| {
                    let path = path.clone();
                    let cache = Arc::clone(&cache);
                    let filter = options.filter.clone();

                    tokio::task::spawn_blocking(move || {
                        read_single_row_group(&path, &cache, rg_idx, &filter)
                    })
                })
                .buffer_unordered(options.parallelism)
                .map(|result| {
                    // Flatten JoinError
                    match result {
                        Ok(inner) => inner,
                        Err(e) => Err(Error::TaskJoin(e)),
                    }
                })
                .flat_map(|result| {
                    // Flatten Vec<Spanset> into individual spansets
                    match result {
                        Ok(spansets) => {
                            futures::stream::iter(spansets.into_iter().map(Ok).collect::<Vec<_>>())
                        }
                        Err(e) => futures::stream::iter(vec![Err(e)]),
                    }
                });

            while let Some(result) = stream.next().await {
                yield result?;
            }
        })
    }
}

/// Extract dictionary from column pages
fn extract_dictionary(
    page_reader: &mut Box<dyn PageReader>,
    _column_desc: Arc<ColumnDescriptor>,
) -> Result<Option<CachedDictionary>> {
    use parquet::basic::Encoding;
    use parquet::column::page::Page;

    while let Some(page) = page_reader.get_next_page()? {
        match page {
            Page::DictionaryPage {
                buf,
                num_values,
                encoding,
                ..
            } => {
                // Only handle plain encoding for byte array dictionaries
                if encoding != Encoding::PLAIN && encoding != Encoding::PLAIN_DICTIONARY {
                    return Ok(None);
                }

                // For ByteArray type, manually decode the plain dictionary
                let mut values = Vec::new();
                let mut offset = 0;
                let data = buf.as_ref();

                for _ in 0..num_values {
                    if offset + 4 > data.len() {
                        break;
                    }

                    // Read 4-byte length prefix (little-endian)
                    let len = u32::from_le_bytes([
                        data[offset],
                        data[offset + 1],
                        data[offset + 2],
                        data[offset + 3],
                    ]) as usize;
                    offset += 4;

                    if offset + len > data.len() {
                        break;
                    }

                    let value_data = &data[offset..offset + len];
                    values.push(ByteArray::from(value_data.to_vec()));
                    offset += len;
                }

                let value_set: HashSet<Vec<u8>> =
                    values.iter().map(|v| v.data().to_vec()).collect();

                return Ok(Some(CachedDictionary { values, value_set }));
            }
            Page::DataPage { .. } | Page::DataPageV2 { .. } => {
                return Ok(None);
            }
        }
    }
    Ok(None)
}

/// Stage 1: Filter row groups using statistics AND dictionaries (combined)
fn filter_row_groups(cache: &FileCache, filter: &Option<SpanFilter>) -> Vec<usize> {
    let Some(filter) = filter else {
        return (0..cache.row_groups.len()).collect();
    };

    cache
        .row_groups
        .iter()
        .filter(|rg| {
            // Check 1: Statistics filter (min/max)
            if !filter.keep_row_group(&rg.metadata, &cache.schema) {
                return false;
            }

            // Check 2: Dictionary filter (if available)
            if let Some(col_idx) = cache.span_name_column_index {
                if let Some(dict) = rg.dictionaries.get(&col_idx) {
                    if !filter.matches_dictionary(dict) {
                        return false;
                    }
                }
            }

            true
        })
        .map(|rg| rg.index)
        .collect()
}

/// Read a single row group and extract Spansets
fn read_single_row_group(
    path: &Path,
    cache: &FileCache,
    rg_idx: usize,
    filter: &Option<SpanFilter>,
) -> Result<Vec<Spanset>> {
    let file = std::fs::File::open(path)?;
    let metadata = ArrowReaderMetadata::load(&file, Default::default())?;

    let mut builder = ParquetRecordBatchReaderBuilder::new_with_metadata(file, metadata);

    // Projection for span fields only
    let projection = get_spanset_projection(&cache.schema);
    builder = builder
        .with_projection(projection)
        .with_row_groups(vec![rg_idx]);

    // Arrow row filter
    if let Some(SpanFilter::NameEquals(ref name)) = filter {
        let predicate = SpanNamePredicate::new(name.clone(), &cache.schema);
        builder = builder.with_row_filter(RowFilter::new(vec![Box::new(predicate)]));
    }

    let reader = builder.build()?;

    let mut spansets = Vec::new();
    for batch_result in reader {
        let batch = batch_result?;
        // Extract spansets from batch
        let batch_spansets = extract_spansets_from_batch(&batch, filter)?;
        spansets.extend(batch_spansets);
    }

    Ok(spansets)
}

/// Get column indices for all span-level fields (no Resource/Trace fields)
fn get_spanset_projection(schema: &SchemaDescriptor) -> ProjectionMask {
    let span_field_paths = [
        "TraceID", // Need this for the spanset
        field_paths::SPAN_ID,
        field_paths::SPAN_PARENT_SPAN_ID,
        field_paths::SPAN_PARENT_ID,
        field_paths::SPAN_NESTED_SET_LEFT,
        field_paths::SPAN_NESTED_SET_RIGHT,
        field_paths::SPAN_NAME,
        field_paths::SPAN_KIND,
        field_paths::SPAN_START_TIME_UNIX_NANO,
        field_paths::SPAN_DURATION_NANO,
        field_paths::SPAN_STATUS_CODE,
    ];

    let indices: Vec<usize> = schema
        .columns()
        .iter()
        .enumerate()
        .filter(|(_, c)| span_field_paths.contains(&c.path().string().as_str()))
        .map(|(i, _)| i)
        .collect();

    ProjectionMask::leaves(schema, indices)
}

/// Extract spansets from a RecordBatch
fn extract_spansets_from_batch(
    batch: &RecordBatch,
    filter: &Option<SpanFilter>,
) -> Result<Vec<Spanset>> {
    let mut spansets = Vec::new();

    // Extract trace-level data
    let trace_id = batch
        .column_by_name("TraceID")
        .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
        .ok_or_else(|| Error::SchemaError("TraceID column not found or wrong type".into()))?;

    // Extract ResourceSpans
    let rs_array = batch
        .column_by_name("rs")
        .and_then(|col| col.as_any().downcast_ref::<ListArray>())
        .ok_or_else(|| Error::SchemaError("rs column not found or wrong type".into()))?;

    // Process each row (trace)
    for row_idx in 0..batch.num_rows() {
        let trace_id_bytes = trace_id.value(row_idx).to_vec();
        let mut all_spans = Vec::new();

        // Iterate through ResourceSpans
        let rs_offset = rs_array.value_offsets()[row_idx] as usize;
        let rs_length =
            (rs_array.value_offsets()[row_idx + 1] - rs_array.value_offsets()[row_idx]) as usize;

        if rs_length > 0 {
            let rs_values = rs_array
                .values()
                .as_any()
                .downcast_ref::<StructArray>()
                .ok_or_else(|| Error::SchemaError("ResourceSpans values not a struct".into()))?;

            // Get ScopeSpans (ss) from ResourceSpans
            let ss_array = rs_values
                .column_by_name("ss")
                .and_then(|col| col.as_any().downcast_ref::<ListArray>())
                .ok_or_else(|| Error::SchemaError("ss column not found or wrong type".into()))?;

            // Iterate through each ResourceSpans
            for rs_idx in rs_offset..rs_offset + rs_length {
                let ss_offset = ss_array.value_offsets()[rs_idx] as usize;
                let ss_length = (ss_array.value_offsets()[rs_idx + 1]
                    - ss_array.value_offsets()[rs_idx]) as usize;

                if ss_length == 0 {
                    continue;
                }

                let ss_values = ss_array
                    .values()
                    .as_any()
                    .downcast_ref::<StructArray>()
                    .ok_or_else(|| Error::SchemaError("ScopeSpans values not a struct".into()))?;

                // Get Spans from ScopeSpans
                let spans_array = ss_values
                    .column_by_name("Spans")
                    .and_then(|col| col.as_any().downcast_ref::<ListArray>())
                    .ok_or_else(|| {
                        Error::SchemaError("Spans column not found or wrong type".into())
                    })?;

                // Iterate through each ScopeSpans
                for ss_idx in ss_offset..ss_offset + ss_length {
                    let spans_offset = spans_array.value_offsets()[ss_idx] as usize;
                    let spans_length = (spans_array.value_offsets()[ss_idx + 1]
                        - spans_array.value_offsets()[ss_idx])
                        as usize;

                    if spans_length == 0 {
                        continue;
                    }

                    let spans_values = spans_array
                        .values()
                        .as_any()
                        .downcast_ref::<StructArray>()
                        .ok_or_else(|| Error::SchemaError("Spans values not a struct".into()))?;

                    // Extract span fields
                    let span_ids = spans_values
                        .column_by_name("SpanID")
                        .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
                        .ok_or_else(|| {
                            Error::SchemaError("SpanID not found or wrong type".into())
                        })?;

                    let parent_span_ids = spans_values
                        .column_by_name("ParentSpanID")
                        .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
                        .ok_or_else(|| {
                            Error::SchemaError("ParentSpanID not found or wrong type".into())
                        })?;

                    let parent_ids = spans_values
                        .column_by_name("ParentID")
                        .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("ParentID not found or wrong type".into())
                        })?;

                    let nested_set_lefts = spans_values
                        .column_by_name("NestedSetLeft")
                        .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("NestedSetLeft not found or wrong type".into())
                        })?;

                    let nested_set_rights = spans_values
                        .column_by_name("NestedSetRight")
                        .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("NestedSetRight not found or wrong type".into())
                        })?;

                    let names = spans_values
                        .column_by_name("Name")
                        .and_then(|col| col.as_any().downcast_ref::<StringArray>())
                        .ok_or_else(|| Error::SchemaError("Name not found or wrong type".into()))?;

                    let kinds = spans_values
                        .column_by_name("Kind")
                        .and_then(|col| col.as_any().downcast_ref::<Int64Array>())
                        .ok_or_else(|| Error::SchemaError("Kind not found or wrong type".into()))?;

                    let start_times = spans_values
                        .column_by_name("StartTimeUnixNano")
                        .and_then(|col| col.as_any().downcast_ref::<UInt64Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("StartTimeUnixNano not found or wrong type".into())
                        })?;

                    let durations = spans_values
                        .column_by_name("DurationNano")
                        .and_then(|col| col.as_any().downcast_ref::<UInt64Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("DurationNano not found or wrong type".into())
                        })?;

                    let status_codes = spans_values
                        .column_by_name("StatusCode")
                        .and_then(|col| col.as_any().downcast_ref::<Int64Array>())
                        .ok_or_else(|| {
                            Error::SchemaError("StatusCode not found or wrong type".into())
                        })?;

                    // Process each span
                    for span_idx in spans_offset..spans_offset + spans_length {
                        let name = names.value(span_idx);

                        // Apply filter
                        if let Some(ref f) = filter {
                            if !f.matches(name) {
                                continue;
                            }
                        }

                        let span = Span {
                            span_id: span_ids.value(span_idx).to_vec().into(),
                            parent_span_id: parent_span_ids.value(span_idx).to_vec().into(),
                            parent_id: parent_ids.value(span_idx),
                            nested_set_left: nested_set_lefts.value(span_idx),
                            nested_set_right: nested_set_rights.value(span_idx),
                            name: name.to_string(),
                            kind: kinds.value(span_idx),
                            start_time_unix_nano: start_times.value(span_idx),
                            duration_nano: durations.value(span_idx),
                            status_code: status_codes.value(span_idx),
                        };

                        all_spans.push(span);
                    }
                }
            }
        }

        // Only include spansets with matching spans (or all if no filter)
        if !all_spans.is_empty() || filter.is_none() {
            spansets.push(Spanset {
                trace_id: trace_id_bytes.into(),
                spans: all_spans,
            });
        }
    }

    Ok(spansets)
}
