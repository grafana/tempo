use parquet::arrow::arrow_reader::ParquetRecordBatchReaderBuilder;
use std::env;
use std::fs::File;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let block_id = env::var("BENCH_BLOCKID")?;
    let tenant_id = env::var("BENCH_TENANTID").unwrap_or_else(|_| "1".to_string());
    let bench_path = env::var("BENCH_PATH")?;

    let mut path = PathBuf::from(bench_path);
    path.push(tenant_id);
    path.push(block_id);
    path.push("data.parquet");

    println!("Opening file: {:?}", path);

    let file = File::open(&path)?;
    let builder = ParquetRecordBatchReaderBuilder::try_new(file)?;

    let schema = builder.schema();

    println!("\nTop-level schema:");
    println!("{:#?}", schema);

    // Get the first batch to inspect the actual structure
    let mut reader = builder.build()?;

    if let Some(batch_result) = reader.next() {
        let batch = batch_result?;

        println!("\n\nFirst batch columns:");
        for (i, field) in batch.schema().fields().iter().enumerate() {
            println!("\nColumn {}: {} ({})", i, field.name(), field.data_type());

            if field.name() == "rs" {
                // Look at the rs column structure
                let column = batch.column(i);
                println!("  RS array type: {:?}", column.data_type());

                // Try to get the first record to inspect nested structure
                use arrow::array::{Array, ListArray};
                if let Some(list_array) = column.as_any().downcast_ref::<ListArray>() {
                    if list_array.len() > 0 {
                        let values = list_array.values();
                        println!("  RS values type: {:?}", values.data_type());

                        // Look at the struct fields
                        use arrow::array::StructArray;
                        if let Some(struct_array) = values.as_any().downcast_ref::<StructArray>() {
                            println!("  RS struct fields:");
                            for (j, field) in struct_array.fields().iter().enumerate() {
                                println!(
                                    "    Field {}: {} ({})",
                                    j,
                                    field.name(),
                                    field.data_type()
                                );

                                if field.name() == "ss" {
                                    // Look at ScopeSpans structure
                                    let ss_column = struct_array.column(j);
                                    println!("      SS array type: {:?}", ss_column.data_type());

                                    if let Some(ss_list) =
                                        ss_column.as_any().downcast_ref::<ListArray>()
                                    {
                                        let ss_values = ss_list.values();
                                        println!(
                                            "      SS values type: {:?}",
                                            ss_values.data_type()
                                        );

                                        if let Some(ss_struct) =
                                            ss_values.as_any().downcast_ref::<StructArray>()
                                        {
                                            println!("      SS struct fields:");
                                            for (k, field) in ss_struct.fields().iter().enumerate()
                                            {
                                                println!(
                                                    "        Field {}: {} ({})",
                                                    k,
                                                    field.name(),
                                                    field.data_type()
                                                );

                                                if field.name() == "Spans" {
                                                    // Look at Spans structure
                                                    let spans_column = ss_struct.column(k);
                                                    println!(
                                                        "          Spans array type: {:?}",
                                                        spans_column.data_type()
                                                    );

                                                    if let Some(spans_list) = spans_column
                                                        .as_any()
                                                        .downcast_ref::<ListArray>(
                                                    ) {
                                                        let spans_values = spans_list.values();
                                                        println!(
                                                            "          Spans values type: {:?}",
                                                            spans_values.data_type()
                                                        );

                                                        if let Some(spans_struct) = spans_values
                                                            .as_any()
                                                            .downcast_ref::<StructArray>()
                                                        {
                                                            println!(
                                                                "          Span struct fields:"
                                                            );
                                                            for (l, field) in spans_struct
                                                                .fields()
                                                                .iter()
                                                                .enumerate()
                                                            {
                                                                println!(
                                                                    "            Field {}: {} ({})",
                                                                    l,
                                                                    field.name(),
                                                                    field.data_type()
                                                                );
                                                            }
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    Ok(())
}
