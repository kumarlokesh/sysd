//! A learning-focused implementation of a RocksDB-like key-value store in Rust.
//!
//! This crate provides a simple, embeddable key-value store with an LSM tree storage
//! engine design, inspired by RocksDB.
//!
//! # Features
//! - In-memory MemTable with get/put/delete operations
//! - Write-ahead log (WAL) for durability
//! - SSTable storage format

#![warn(missing_docs)]
#![warn(rustdoc::missing_crate_level_docs)]

/// Configuration types and utilities for the key-value store
///
/// This module contains types and functions for managing the database configuration,
/// including storage paths, performance tuning parameters, and other settings.
pub mod config;

/// Error types and result aliases used throughout the crate
///
/// This module defines the error type used by the database and related utilities
/// for error handling and conversion.
pub mod error;

/// Storage engine implementation
///
/// This module contains the core storage engine implementation, including the MemTable,
/// Write-Ahead Log (WAL), and SSTable components that make up the LSM tree storage engine.
pub mod storage;

use error::Result;
use std::path::Path;
use storage::Store;

/// Main database type that provides the key-value store interface
pub struct DB {
    /// Persistent storage backend
    store: storage::PersistentStore,
}

impl DB {
    /// Opens a database with the given configuration
    pub fn open<P: AsRef<Path>>(path: P, create_if_missing: bool) -> Result<Self> {
        let path = path.as_ref();
        if !path.exists() {
            if create_if_missing {
                std::fs::create_dir_all(path)?;
            } else {
                return Err(error::Error::DatabaseNotFound(
                    path.to_string_lossy().to_string(),
                ));
            }
        }

        let store = storage::PersistentStore::open(path)?;
        Ok(Self { store })
    }

    /// Retrieves a value by key
    pub fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>> {
        self.store.get(key)
    }

    /// Inserts or updates a key-value pair
    pub fn put(&mut self, key: &[u8], value: Vec<u8>) -> Result<()> {
        self.store.put(key, value)
    }

    /// Deletes a key from the database
    pub fn delete(&mut self, key: &[u8]) -> Result<()> {
        self.store.delete(key)
    }

    /// Flushes any pending writes to disk
    pub fn flush(&mut self) -> Result<()> {
        self.store.flush()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_db_basic_operations() -> Result<()> {
        let _ = env_logger::builder().is_test(true).try_init();

        let temp_dir = tempdir()?;
        let mut db = DB::open(temp_dir.path(), true)?;

        let key = b"test_key";
        let value = b"test_value".to_vec();

        let initial_get = db.get(key)?;
        assert!(
            initial_get.is_none(),
            "Expected None for initial get, got {:?}",
            initial_get
        );

        db.put(key, value.clone())?;

        let after_put = db.get(key)?;
        assert_eq!(
            after_put.as_deref(),
            Some(value.as_slice()),
            "Expected value after put, got {:?}",
            after_put
        );

        db.delete(key)?;

        db.flush()?;

        let after_delete = db.get(key)?;
        assert!(
            after_delete.is_none(),
            "Expected None after delete, got {:?}",
            after_delete
        );

        Ok(())
    }
}
