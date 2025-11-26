use std::fs;
use std::path::Path;
use traceql::traceql_to_sql_string;

/// Strip comments starting with # from the query
fn strip_comments(content: &str) -> String {
    content
        .lines()
        .filter(|line| !line.trim_start().starts_with('#'))
        .collect::<Vec<_>>()
        .join("\n")
        .trim()
        .to_string()
}

fn main() {
    let queries_dir = Path::new("crates/traceql/queries");

    // Read all .tql files
    let entries = fs::read_dir(queries_dir).unwrap();

    for entry in entries {
        let entry = entry.unwrap();
        let path = entry.path();

        if path.extension().and_then(|s| s.to_str()) == Some("tql") {
            let tql_content = fs::read_to_string(&path).unwrap();
            let query = strip_comments(&tql_content);

            if query.is_empty() {
                continue;
            }

            // Convert to SQL
            match traceql_to_sql_string(&query) {
                Ok(sql) => {
                    // Write to corresponding .sql file
                    let sql_path = path.with_extension("sql");
                    fs::write(&sql_path, format!("{}\n", sql)).unwrap();
                    println!("Generated: {}", sql_path.display());
                }
                Err(e) => {
                    eprintln!("Error converting {}: {}", path.display(), e);
                }
            }
        }
    }
}
