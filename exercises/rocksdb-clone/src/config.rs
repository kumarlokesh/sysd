use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Configuration for the RocksDB Clone
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    /// Path to the database directory
    pub path: PathBuf,

    /// Maximum size of the MemTable in bytes before it becomes immutable
    pub memtable_size: usize,

    /// Whether to sync writes to disk immediately
    pub sync: bool,

    /// Whether to create the database if it doesn't exist
    pub create_if_missing: bool,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            path: std::env::current_dir().unwrap().join("rocksdb_data"),
            memtable_size: 64 * 1024 * 1024, // 64MB
            sync: false,
            create_if_missing: true,
        }
    }
}

impl Config {
    /// Create a new configuration with default values
    pub fn new() -> Self {
        Self::default()
    }

    /// Set the database path
    pub fn path<P: Into<PathBuf>>(mut self, path: P) -> Self {
        self.path = path.into();
        self
    }

    /// Set the maximum MemTable size in bytes
    pub fn memtable_size(mut self, size: usize) -> Self {
        self.memtable_size = size;
        self
    }

    /// Enable or disable sync writes
    pub fn sync(mut self, sync: bool) -> Self {
        self.sync = sync;
        self
    }

    /// Enable or disable creating the database if it doesn't exist
    pub fn create_if_missing(mut self, create: bool) -> Self {
        self.create_if_missing = create;
        self
    }
}
