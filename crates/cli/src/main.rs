use anyhow::{Context, Result};
use clap::Parser;
use context::{create_context, execute_query};
use rustyline::error::ReadlineError;
use rustyline::DefaultEditor;
use std::path::PathBuf;
use tracing::error;

/// DataFusion SQL query interface
#[derive(Parser, Debug)]
#[command(name = "tempo-datafusion")]
#[command(about = "DataFusion SQL query interface", long_about = None)]
struct Args {
    /// Execute a single query and exit
    #[arg(short, long)]
    exec: Option<String>,

    /// Path to TOML configuration file
    #[arg(short, long)]
    config: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing subscriber with environment filter
    // Set RUST_LOG environment variable to control log level
    // Example: RUST_LOG=info or RUST_LOG=debug
    // hyper_util, rustyline, and reqwest are always set to ERROR level to reduce noise
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info"))
                .add_directive("hyper_util=error".parse().unwrap())
                .add_directive("rustyline=error".parse().unwrap()), //.add_directive("reqwest=error".parse().unwrap())
        )
        .init();

    let args = Args::parse();

    // Create DataFusion context with optional config file
    let ctx = create_context(args.config.as_deref()).await?;

    // If --exec flag is provided, execute the query and exit
    if let Some(query) = args.exec {
        let result = execute_query(&ctx, &query).await?;
        println!("{}", result);
        return Ok(());
    }

    println!("DataFusion REPL");
    println!("Type 'exit' or 'quit' to exit, '\\h' for help\n");

    // Setup history file path
    let history_file = get_history_file_path();

    // Create rustyline editor
    let mut rl = DefaultEditor::new().context("Failed to create readline editor")?;

    // Load history from file if it exists
    if history_file.exists() {
        rl.load_history(&history_file)
            .context("Failed to load history file")?;
    }

    // REPL loop
    loop {
        let readline = rl.readline("datafusion> ");
        match readline {
            Ok(line) => {
                let line = line.trim();

                // Skip empty lines
                if line.is_empty() {
                    continue;
                }

                // Add to history
                let _ = rl.add_history_entry(line);

                // Check for exit commands
                if line.eq_ignore_ascii_case("exit") || line.eq_ignore_ascii_case("quit") {
                    save_history(&mut rl, &history_file);
                    println!("Goodbye!");
                    break;
                }

                // Check for help command
                if line == "\\h" {
                    print_help();
                    continue;
                }

                // Execute SQL query
                match execute_query(&ctx, line).await {
                    Ok(result) => {
                        println!("{}", result);
                    }
                    Err(e) => {
                        error!("Query execution failed: {}", e);
                        eprintln!("Error: {}", e);
                    }
                }
            }
            Err(ReadlineError::Interrupted) => {
                save_history(&mut rl, &history_file);
                println!("CTRL-C");
                break;
            }
            Err(ReadlineError::Eof) => {
                save_history(&mut rl, &history_file);
                println!("CTRL-D");
                break;
            }
            Err(err) => {
                save_history(&mut rl, &history_file);
                error!("Readline error: {:?}", err);
                eprintln!("Error: {:?}", err);
                break;
            }
        }
    }

    Ok(())
}

fn get_history_file_path() -> PathBuf {
    // Try to use home directory, otherwise use current directory
    if let Some(home) = dirs::home_dir() {
        home.join(".datafusion_history")
    } else {
        PathBuf::from(".datafusion_history")
    }
}

fn save_history(rl: &mut DefaultEditor, history_file: &PathBuf) {
    use tracing::warn;
    rl.save_history(history_file)
        .context("Failed to save history file")
        .unwrap_or_else(|e| {
            warn!("Could not save history: {}", e);
            eprintln!("Warning: Could not save history: {}", e);
        });
}

fn print_help() {
    println!("DataFusion REPL Help:");
    println!("  exit, quit    - Exit the REPL");
    println!("  \\h            - Show this help message");
    println!("  <SQL>         - Execute a SQL statement");
    println!("  |<TraceQL>    - Execute a TraceQL query (prefix with |)");
    println!("\nExample SQL queries:");
    println!("  SELECT * FROM spans LIMIT 10;");
    println!("  SELECT COUNT(*) FROM traces;");
    println!("\nExample TraceQL queries:");
    println!(r#"  |{{ span.http.method = "GET" }}"#);
    println!(r#"  |{{ duration > 100ms }}"#);
    println!(r#"  |{{ span.http.method = "POST" && span.http.status_code = 500 }}"#);
    println!("  |{{ }} | rate()");
}
