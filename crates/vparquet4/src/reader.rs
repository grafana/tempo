/// Parquet reader for vparquet4 files

use std::fs::File;
use std::path::Path;

use arrow::array::{
    Array, BinaryArray, Int32Array, Int64Array, ListArray, RecordBatch, StringArray, StructArray,
    UInt64Array,
};
use parquet::arrow::arrow_reader::{
    ParquetRecordBatchReader, ParquetRecordBatchReaderBuilder, RowFilter,
};
use parquet::arrow::ProjectionMask;

use crate::error::{Error, Result};
use crate::filter::{SpanFilter, SpanNamePredicate};
use crate::schema::{field_paths, Span, Spanset};

/// Configuration for reading a parquet file
#[derive(Debug, Clone)]
pub struct ReadOptions {
    /// Starting row group index (0-based)
    pub start_row_group: usize,
    /// Number of row groups to read (0 = all remaining)
    pub total_row_groups: usize,
    /// Filter to apply to spans
    pub filter: Option<SpanFilter>,
}

impl Default for ReadOptions {
    fn default() -> Self {
        Self {
            start_row_group: 0,
            total_row_groups: 0,
            filter: None,
        }
    }
}

/// Get column indices for all span-level fields (no Resource/Trace fields)
fn get_spanset_projection(schema: &parquet::schema::types::SchemaDescriptor) -> ProjectionMask {
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

/// Reader for vparquet4 files that returns an iterator of spansets
pub struct VParquet4Reader {
    reader: ParquetRecordBatchReader,
    filter: Option<SpanFilter>,
    current_batch: Option<RecordBatch>,
    current_row: usize,
}

impl VParquet4Reader {
    /// Open a parquet file with the given options
    pub fn open<P: AsRef<Path>>(path: P, options: ReadOptions) -> Result<Self> {
        let file = File::open(path)?;

        // Build the reader
        let builder = ParquetRecordBatchReaderBuilder::try_new(file)?;

        let metadata = builder.metadata().clone();
        let schema = builder.parquet_schema();
        let num_row_groups = metadata.num_row_groups();

        // Determine row group range
        let start = options.start_row_group;
        let total = if options.total_row_groups == 0 {
            num_row_groups - start
        } else {
            options.total_row_groups.min(num_row_groups - start)
        };

        if start >= num_row_groups {
            return Err(Error::InvalidRowGroup(format!(
                "start_row_group {} >= total row groups {}",
                start, num_row_groups
            )));
        }

        // Step 1: Filter row groups based on statistics
        let row_groups: Vec<usize> = (start..start + total)
            .filter(|&idx| {
                options
                    .filter
                    .as_ref()
                    .map(|f| f.keep_row_group(metadata.row_group(idx), &schema))
                    .unwrap_or(true)
            })
            .collect();

        // Step 2: Calculate projection to only read spanset fields (skip Resource/Trace fields)
        let projection = get_spanset_projection(&schema);

        // Step 3: Create RowFilter for trace-level filtering
        let row_filter = match &options.filter {
            Some(SpanFilter::NameEquals(name)) => {
                let predicate = SpanNamePredicate::new(name.clone(), &schema);
                Some(RowFilter::new(vec![Box::new(predicate)]))
            }
            None => None,
        };

        // Now apply all configurations to the builder
        let builder = builder.with_row_groups(row_groups);
        let builder = builder.with_projection(projection);
        let builder = match row_filter {
            Some(filter) => builder.with_row_filter(filter),
            None => builder,
        };

        let reader = builder.build()?;

        Ok(Self {
            reader,
            filter: options.filter,
            current_batch: None,
            current_row: 0,
        })
    }

    /// Read the next spanset from the file
    fn next_spanset(&mut self) -> Result<Option<Spanset>> {
        // Loop until we find a spanset with matching spans or reach the end
        loop {
            // Get the next batch if we don't have one or exhausted the current one
            if self.current_batch.is_none() || self.current_row >= self.current_batch.as_ref().unwrap().num_rows() {
                self.current_batch = self.reader.next().transpose()?;
                self.current_row = 0;

                if self.current_batch.is_none() {
                    return Ok(None);
                }
            }

            let batch = self.current_batch.as_ref().unwrap();
            let row_idx = self.current_row;
            self.current_row += 1;

        // Extract trace-level data
        let trace_id = batch
            .column_by_name("TraceID")
            .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
            .ok_or_else(|| Error::SchemaError("TraceID column not found or wrong type".into()))?;

        let trace_id_bytes = trace_id.value(row_idx).to_vec();

        // Extract ResourceSpans
        let rs_array = batch
            .column_by_name("rs")
            .and_then(|col| col.as_any().downcast_ref::<ListArray>())
            .ok_or_else(|| Error::SchemaError("rs column not found or wrong type".into()))?;

        let mut all_spans = Vec::new();

        // Iterate through ResourceSpans
        let rs_offset = rs_array.value_offsets()[row_idx] as usize;
        let rs_length = (rs_array.value_offsets()[row_idx + 1] - rs_array.value_offsets()[row_idx]) as usize;

        if rs_length == 0 {
            return Ok(Some(Spanset {
                trace_id: trace_id_bytes.into(),
                spans: all_spans,
            }));
        }

        let rs_values = rs_array.values().as_any().downcast_ref::<StructArray>()
            .ok_or_else(|| Error::SchemaError("ResourceSpans values not a struct".into()))?;

        // Get ScopeSpans (ss) from ResourceSpans
        let ss_array = rs_values
            .column_by_name("ss")
            .and_then(|col| col.as_any().downcast_ref::<ListArray>())
            .ok_or_else(|| Error::SchemaError("ss column not found or wrong type".into()))?;

        // Iterate through each ResourceSpans
        for rs_idx in rs_offset..rs_offset + rs_length {
            let ss_offset = ss_array.value_offsets()[rs_idx] as usize;
            let ss_length = (ss_array.value_offsets()[rs_idx + 1] - ss_array.value_offsets()[rs_idx]) as usize;

            if ss_length == 0 {
                continue;
            }

            let ss_values = ss_array.values().as_any().downcast_ref::<StructArray>()
                .ok_or_else(|| Error::SchemaError("ScopeSpans values not a struct".into()))?;

            // Get Spans from ScopeSpans
            let spans_array = ss_values
                .column_by_name("Spans")
                .and_then(|col| col.as_any().downcast_ref::<ListArray>())
                .ok_or_else(|| Error::SchemaError("Spans column not found or wrong type".into()))?;

            // Iterate through each ScopeSpans
            for ss_idx in ss_offset..ss_offset + ss_length {
                let spans_offset = spans_array.value_offsets()[ss_idx] as usize;
                let spans_length = (spans_array.value_offsets()[ss_idx + 1] - spans_array.value_offsets()[ss_idx]) as usize;

                if spans_length == 0 {
                    continue;
                }

                let spans_values = spans_array.values().as_any().downcast_ref::<StructArray>()
                    .ok_or_else(|| Error::SchemaError("Spans values not a struct".into()))?;

                // Extract span fields
                let span_ids = spans_values
                    .column_by_name("SpanID")
                    .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
                    .ok_or_else(|| Error::SchemaError("SpanID not found or wrong type".into()))?;

                let parent_span_ids = spans_values
                    .column_by_name("ParentSpanID")
                    .and_then(|col| col.as_any().downcast_ref::<BinaryArray>())
                    .ok_or_else(|| Error::SchemaError("ParentSpanID not found or wrong type".into()))?;

                let parent_ids = spans_values
                    .column_by_name("ParentID")
                    .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                    .ok_or_else(|| Error::SchemaError("ParentID not found or wrong type".into()))?;

                let nested_set_lefts = spans_values
                    .column_by_name("NestedSetLeft")
                    .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                    .ok_or_else(|| Error::SchemaError("NestedSetLeft not found or wrong type".into()))?;

                let nested_set_rights = spans_values
                    .column_by_name("NestedSetRight")
                    .and_then(|col| col.as_any().downcast_ref::<Int32Array>())
                    .ok_or_else(|| Error::SchemaError("NestedSetRight not found or wrong type".into()))?;

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
                    .ok_or_else(|| Error::SchemaError("StartTimeUnixNano not found or wrong type".into()))?;

                let durations = spans_values
                    .column_by_name("DurationNano")
                    .and_then(|col| col.as_any().downcast_ref::<UInt64Array>())
                    .ok_or_else(|| Error::SchemaError("DurationNano not found or wrong type".into()))?;

                let status_codes = spans_values
                    .column_by_name("StatusCode")
                    .and_then(|col| col.as_any().downcast_ref::<Int64Array>())
                    .ok_or_else(|| Error::SchemaError("StatusCode not found or wrong type".into()))?;

                // Process each span
                for span_idx in spans_offset..spans_offset + spans_length {
                    let name = names.value(span_idx);

                    // Apply filter
                    if let Some(ref filter) = self.filter {
                        if !filter.matches(name) {
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

            // Only return spansets that have matching spans
            if all_spans.is_empty() && self.filter.is_some() {
                // Skip this trace and try the next one
                continue;
            }

            return Ok(Some(Spanset {
                trace_id: trace_id_bytes.into(),
                spans: all_spans,
            }));
        }
    }
}

impl Iterator for VParquet4Reader {
    type Item = Result<Spanset>;

    fn next(&mut self) -> Option<Self::Item> {
        match self.next_spanset() {
            Ok(Some(spanset)) => Some(Ok(spanset)),
            Ok(None) => None,
            Err(e) => Some(Err(e)),
        }
    }
}
