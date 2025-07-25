use clap::{Parser, Subcommand};
use rocksdb_clone::config::Config;
use rocksdb_clone::{error::Result, DB};
use std::path::PathBuf;

/// A simple key-value store CLI
#[derive(Debug, Parser)]
#[clap(name = "rocksdb-clone", version = "0.1.0")]
struct Cli {
    /// Path to the database directory
    #[clap(short, long, default_value = "rocksdb_data")]
    path: PathBuf,

    #[clap(subcommand)]
    command: Commands,
}

#[derive(Debug, Subcommand)]
enum Commands {
    /// Get a value by key
    Get { key: String },

    /// Set a key-value pair
    Set { key: String, value: String },

    /// Delete a key
    Delete { key: String },
}

fn main() -> Result<()> {
    env_logger::init();

    let cli = Cli::parse();

    let config = Config::new().path(cli.path);

    let mut db = DB::open(&config.path, config.create_if_missing)?;

    match cli.command {
        Commands::Get { key } => {
            if let Some(value) = db.get(key.as_bytes())?.map(|v| v.to_vec()) {
                println!("{}", String::from_utf8_lossy(&value));
            } else {
                println!("Key not found");
            }
        }
        Commands::Set { key, value } => {
            db.put(key.as_bytes(), value.as_bytes().to_vec())?;
            println!("OK");
        }
        Commands::Delete { key } => {
            db.delete(key.as_bytes())?;
            println!("OK");
        }
    }

    Ok(())
}
