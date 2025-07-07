//! A learning-focused implementation of a RocksDB-like key-value store in Rust.
//!
//! This crate provides a simple, embeddable key-value store with an LSM tree storage
//! engine design, inspired by RocksDB.

#![warn(missing_docs)]
#![warn(rustdoc::missing_crate_level_docs)]

pub mod config;
pub mod error;
pub mod storage;

use bytes::Bytes;
use error::Result;

/// Main database type that provides the key-value store interface
pub struct DB {
    /// In-memory table for recent writes
    memtable: storage::MemTable,
    // TODO: Add other fields as we implement more features
}

impl DB {
    /// Opens a database with the given configuration
    pub fn open(config: config::Config) -> Result<Self> {
        // For now, just create a new in-memory database
        // In the future, this will load from disk
        Ok(Self {
            memtable: storage::MemTable::new(),
        })
    }

    /// Retrieves a value by key
    pub fn get(&self, key: &[u8]) -> Result<Option<Bytes>> {
        match self.memtable.get(key) {
            Some(storage::Value::Value(value)) => Ok(Some(value.clone())),
            Some(storage::Value::Tombstone) => Ok(None),
            None => Ok(None),
        }
    }

    /// Inserts or updates a key-value pair
    pub fn put(&mut self, key: Bytes, value: Bytes) -> Result<()> {
        self.memtable.put(key, value);
        Ok(())
    }

    /// Deletes a key from the database
    pub fn delete(&mut self, key: Bytes) -> Result<()> {
        self.memtable.delete(key);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::Bytes;

    #[test]
    fn test_db_basic_operations() -> Result<()> {
        let result = add(2, 2);
        assert_eq!(result, 4);
    }
}
