use std::any::Any;
use std::fmt::{self, Debug, Formatter};
use std::path::PathBuf;
use std::pin::Pin;
use std::sync::Arc;
use std::task::{Context, Poll};

use datafusion::arrow::datatypes::SchemaRef;
use datafusion::arrow::record_batch::RecordBatch;
use datafusion::error::Result;
use datafusion::execution::context::TaskContext;
use datafusion::physical_expr::{EquivalenceProperties, Partitioning};
use datafusion::physical_plan::execution_plan::{Boundedness, EmissionType};
use datafusion::physical_plan::{
    DisplayAs, DisplayFormatType, ExecutionPlan, PlanProperties, RecordBatchStream,
    SendableRecordBatchStream,
};
use futures::{Stream, StreamExt};
use vparquet4::{ReadOptions, SpanFilter, VParquet4Reader};

use crate::spanset_converter::{flat_span_schema, spansets_to_record_batch};

/// Execution plan that reads from VParquet4Reader and converts to RecordBatch
#[derive(Debug)]
pub struct VParquet4Exec {
    /// Path to the parquet file
    file_path: PathBuf,
    /// Schema of the output (flat span schema)
    schema: SchemaRef,
    /// Optional filter to push down to VParquet4Reader
    filter: Option<SpanFilter>,
    /// Projected columns (indices into schema)
    projection: Option<Vec<usize>>,
    /// Row limit
    limit: Option<usize>,
    /// Plan properties (cached for performance)
    properties: PlanProperties,
}

impl VParquet4Exec {
    pub fn new(
        file_path: PathBuf,
        filter: Option<SpanFilter>,
        projection: Option<Vec<usize>>,
        limit: Option<usize>,
    ) -> Self {
        let schema = flat_span_schema();

        // Apply projection to schema if specified
        let output_schema = match &projection {
            Some(indices) => {
                let fields: Vec<_> = indices.iter().map(|i| schema.field(*i).clone()).collect();
                Arc::new(datafusion::arrow::datatypes::Schema::new(fields))
            }
            None => schema,
        };

        let properties = PlanProperties::new(
            EquivalenceProperties::new(output_schema.clone()),
            Partitioning::UnknownPartitioning(1), // Single partition for local file
            EmissionType::Final,                  // Records emitted only once all input processed
            Boundedness::Bounded,
        );

        Self {
            file_path,
            schema: output_schema,
            filter,
            projection,
            limit,
            properties,
        }
    }
}

impl DisplayAs for VParquet4Exec {
    fn fmt_as(&self, t: DisplayFormatType, f: &mut Formatter<'_>) -> fmt::Result {
        match t {
            DisplayFormatType::Default | DisplayFormatType::Verbose | DisplayFormatType::TreeRender => {
                write!(
                    f,
                    "VParquet4Exec: file={}, filter={:?}, projection={:?}, limit={:?}",
                    self.file_path.display(),
                    self.filter,
                    self.projection,
                    self.limit
                )
            }
        }
    }
}

impl fmt::Display for VParquet4Exec {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        self.fmt_as(DisplayFormatType::Default, f)
    }
}

impl ExecutionPlan for VParquet4Exec {
    fn name(&self) -> &str {
        "VParquet4Exec"
    }

    fn as_any(&self) -> &dyn Any {
        self
    }

    fn schema(&self) -> SchemaRef {
        self.schema.clone()
    }

    fn properties(&self) -> &PlanProperties {
        &self.properties
    }

    fn children(&self) -> Vec<&Arc<dyn ExecutionPlan>> {
        vec![] // Leaf node
    }

    fn with_new_children(
        self: Arc<Self>,
        _children: Vec<Arc<dyn ExecutionPlan>>,
    ) -> Result<Arc<dyn ExecutionPlan>> {
        Ok(self) // No children to replace
    }

    fn execute(
        &self,
        _partition: usize,
        _context: Arc<TaskContext>,
    ) -> Result<SendableRecordBatchStream> {
        let file_path = self.file_path.clone();
        let filter = self.filter.clone();
        let projection = self.projection.clone();
        let limit = self.limit;
        let schema = self.schema.clone();

        Ok(Box::pin(VParquet4Stream::new(
            file_path, schema, filter, projection, limit,
        )))
    }
}

/// Stream that reads from VParquet4Reader and converts Spansets to RecordBatch
pub struct VParquet4Stream {
    schema: SchemaRef,
    inner: Pin<Box<dyn Stream<Item = Result<RecordBatch>> + Send>>,
}

impl VParquet4Stream {
    fn new(
        file_path: PathBuf,
        schema: SchemaRef,
        filter: Option<SpanFilter>,
        projection: Option<Vec<usize>>,
        limit: Option<usize>,
    ) -> Self {
        let stream = Self::create_stream(file_path, schema.clone(), filter, projection, limit);
        Self {
            schema,
            inner: Box::pin(stream),
        }
    }

    fn create_stream(
        file_path: PathBuf,
        _schema: SchemaRef,
        filter: Option<SpanFilter>,
        projection: Option<Vec<usize>>,
        limit: Option<usize>,
    ) -> impl Stream<Item = Result<RecordBatch>> + Send {
        async_stream::try_stream! {
            let options = ReadOptions {
                filter,
                ..Default::default()
            };

            let reader = VParquet4Reader::open(&file_path, options)
                .await
                .map_err(|e| datafusion::error::DataFusionError::External(Box::new(e)))?;

            let mut stream = reader.read();
            let mut batch_buffer = Vec::with_capacity(1000);
            let mut total_rows = 0usize;
            let batch_size = 1000; // Number of spansets to buffer before converting to RecordBatch

            while let Some(result) = stream.next().await {
                let spanset = result
                    .map_err(|e| datafusion::error::DataFusionError::External(Box::new(e)))?;

                let span_count = spanset.spans.len();
                batch_buffer.push(spanset);
                total_rows += span_count;

                // Check limit
                if let Some(lim) = limit {
                    if total_rows >= lim {
                        // Flush and break
                        let batch = spansets_to_record_batch(std::mem::take(&mut batch_buffer))?;
                        let batch = apply_projection(batch, &projection)?;
                        let batch = apply_limit(batch, lim - (total_rows - span_count))?;
                        yield batch;
                        break;
                    }
                }

                // Flush batch when buffer is large enough
                if batch_buffer.len() >= batch_size {
                    let batch = spansets_to_record_batch(std::mem::take(&mut batch_buffer))?;
                    let batch = apply_projection(batch, &projection)?;
                    yield batch;
                }
            }

            // Flush remaining
            if !batch_buffer.is_empty() {
                let batch = spansets_to_record_batch(batch_buffer)?;
                let batch = apply_projection(batch, &projection)?;
                yield batch;
            }
        }
    }
}

impl Stream for VParquet4Stream {
    type Item = Result<RecordBatch>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        self.inner.as_mut().poll_next(cx)
    }
}

impl RecordBatchStream for VParquet4Stream {
    fn schema(&self) -> SchemaRef {
        self.schema.clone()
    }
}

/// Apply column projection to a RecordBatch
fn apply_projection(batch: RecordBatch, projection: &Option<Vec<usize>>) -> Result<RecordBatch> {
    match projection {
        Some(indices) => {
            let columns: Vec<_> = indices.iter().map(|i| batch.column(*i).clone()).collect();
            let fields: Vec<_> = indices
                .iter()
                .map(|i| batch.schema().field(*i).clone())
                .collect();
            let schema = Arc::new(datafusion::arrow::datatypes::Schema::new(fields));
            RecordBatch::try_new(schema, columns)
                .map_err(|e| datafusion::error::DataFusionError::ArrowError(Box::new(e), None))
        }
        None => Ok(batch),
    }
}

/// Apply row limit to a RecordBatch
fn apply_limit(batch: RecordBatch, limit: usize) -> Result<RecordBatch> {
    if batch.num_rows() <= limit {
        Ok(batch)
    } else {
        Ok(batch.slice(0, limit))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_vparquet4_exec_creation() {
        let exec = VParquet4Exec::new(
            PathBuf::from("/test/file.parquet"),
            None,
            None,
            Some(10),
        );

        assert_eq!(exec.name(), "VParquet4Exec");
        assert_eq!(exec.schema().fields().len(), 11);
        assert!(exec.children().is_empty());
    }

    #[test]
    fn test_vparquet4_exec_with_filter() {
        let exec = VParquet4Exec::new(
            PathBuf::from("/test/file.parquet"),
            Some(SpanFilter::NameEquals("test".to_string())),
            None,
            None,
        );

        assert!(exec.filter.is_some());
    }

    #[test]
    fn test_vparquet4_exec_with_projection() {
        let exec = VParquet4Exec::new(
            PathBuf::from("/test/file.parquet"),
            None,
            Some(vec![0, 3]), // trace_id and name columns
            None,
        );

        // Projected schema should have only 2 fields
        assert_eq!(exec.schema().fields().len(), 2);
    }

    #[test]
    fn test_vparquet4_exec_display() {
        let exec = VParquet4Exec::new(
            PathBuf::from("/test/file.parquet"),
            Some(SpanFilter::NameEquals("test".to_string())),
            Some(vec![0, 3]),
            Some(100),
        );

        let display = format!("{}", exec);
        assert!(display.contains("VParquet4Exec"));
        assert!(display.contains("file=/test/file.parquet"));
        assert!(display.contains("filter=Some(NameEquals"));
    }

    #[test]
    fn test_with_new_children_returns_self() {
        let exec = Arc::new(VParquet4Exec::new(
            PathBuf::from("/test/file.parquet"),
            None,
            None,
            None,
        ));

        let result = exec.clone().with_new_children(vec![]).unwrap();
        let result_vparquet4 = result.as_any().downcast_ref::<VParquet4Exec>().unwrap();
        assert!(std::ptr::eq(
            exec.as_ref() as *const VParquet4Exec,
            result_vparquet4 as *const VParquet4Exec
        ));
    }

    #[test]
    fn test_apply_projection_none() {
        use datafusion::arrow::array::{Int32Array, StringArray};

        let schema = Arc::new(datafusion::arrow::datatypes::Schema::new(vec![
            datafusion::arrow::datatypes::Field::new("a", datafusion::arrow::datatypes::DataType::Int32, false),
            datafusion::arrow::datatypes::Field::new("b", datafusion::arrow::datatypes::DataType::Utf8, false),
        ]));

        let batch = RecordBatch::try_new(
            schema,
            vec![
                Arc::new(Int32Array::from(vec![1, 2, 3])),
                Arc::new(StringArray::from(vec!["a", "b", "c"])),
            ],
        )
        .unwrap();

        let result = apply_projection(batch.clone(), &None).unwrap();
        assert_eq!(result.num_columns(), 2);
    }

    #[test]
    fn test_apply_projection_some() {
        use datafusion::arrow::array::{Int32Array, StringArray};

        let schema = Arc::new(datafusion::arrow::datatypes::Schema::new(vec![
            datafusion::arrow::datatypes::Field::new("a", datafusion::arrow::datatypes::DataType::Int32, false),
            datafusion::arrow::datatypes::Field::new("b", datafusion::arrow::datatypes::DataType::Utf8, false),
        ]));

        let batch = RecordBatch::try_new(
            schema,
            vec![
                Arc::new(Int32Array::from(vec![1, 2, 3])),
                Arc::new(StringArray::from(vec!["a", "b", "c"])),
            ],
        )
        .unwrap();

        let result = apply_projection(batch, &Some(vec![1])).unwrap();
        assert_eq!(result.num_columns(), 1);
        assert_eq!(result.schema().field(0).name(), "b");
    }

    #[test]
    fn test_apply_limit() {
        use datafusion::arrow::array::Int32Array;

        let schema = Arc::new(datafusion::arrow::datatypes::Schema::new(vec![
            datafusion::arrow::datatypes::Field::new("a", datafusion::arrow::datatypes::DataType::Int32, false),
        ]));

        let batch = RecordBatch::try_new(
            schema,
            vec![Arc::new(Int32Array::from(vec![1, 2, 3, 4, 5]))],
        )
        .unwrap();

        let result = apply_limit(batch, 3).unwrap();
        assert_eq!(result.num_rows(), 3);
    }

    #[test]
    fn test_apply_limit_no_change() {
        use datafusion::arrow::array::Int32Array;

        let schema = Arc::new(datafusion::arrow::datatypes::Schema::new(vec![
            datafusion::arrow::datatypes::Field::new("a", datafusion::arrow::datatypes::DataType::Int32, false),
        ]));

        let batch = RecordBatch::try_new(
            schema,
            vec![Arc::new(Int32Array::from(vec![1, 2, 3]))],
        )
        .unwrap();

        let result = apply_limit(batch.clone(), 5).unwrap();
        assert_eq!(result.num_rows(), 3);
    }
}
