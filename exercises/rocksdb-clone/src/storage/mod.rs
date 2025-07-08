//! Storage module for the RocksDB Clone
//!
//! This module contains the storage-related types and implementations,
//! including the MemTable, WAL, and (eventually) disk-based storage.

mod memtable;
mod wal;

pub use memtable::{MemTable, Value};
pub use wal::{WalOp, WriteAheadLog};

use std::path::Path;

use crate::error::Result;

/// Trait for key-value storage operations
pub trait Store {
    /// Retrieves a value by key
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>>;

    /// Inserts or updates a key-value pair
    fn put(&mut self, key: &[u8], value: Vec<u8>) -> Result<()>;

    /// Deletes a key
    fn delete(&mut self, key: &[u8]) -> Result<()>;

    /// Returns an iterator over the key-value pairs
    fn iter(&self) -> Box<dyn Iterator<Item = (Vec<u8>, Vec<u8>)> + '_>;

    /// Flushes any pending writes to disk
    fn flush(&mut self) -> Result<()>;
}

/// A persistent key-value store that combines MemTable and WAL
pub struct PersistentStore {
    memtable: MemTable,
    wal: WriteAheadLog,
}

impl PersistentStore {
    /// Opens or creates a new persistent store at the given path
    pub fn open(path: impl AsRef<Path>) -> Result<Self> {
        let wal_path = path.as_ref().join("wal.log");
        let memtable = MemTable::new();

        // Replay WAL to rebuild MemTable if it exists
        if wal_path.exists() {
            let mut memtable = MemTable::new();
            WriteAheadLog::replay(&wal_path, |op| {
                match op {
                    WalOp::Put { key, value } => {
                        // Convert key and value to owned Vec<u8> if they're not already
                        memtable.put(key, value);
                    }
                    WalOp::Delete { key } => {
                        // Convert key to owned Vec<u8> if it's not already
                        memtable.delete(key);
                    }
                }
                Ok(())
            })?;

            let wal = WriteAheadLog::new(wal_path)?;
            Ok(Self { memtable, wal })
        } else {
            let wal = WriteAheadLog::new(wal_path)?;
            Ok(Self { memtable, wal })
        }
    }
}

impl Store for PersistentStore {
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>> {
        match self.memtable.get(key) {
            Some(Value::Value(value)) => Ok(Some(value.to_vec())),
            Some(Value::Tombstone) | None => Ok(None),
        }
    }

    fn put(&mut self, key: &[u8], value: Vec<u8>) -> Result<()> {
        self.wal.append(&WalOp::Put {
            key: key.to_vec(),
            value: value.clone(),
        })?;
        self.memtable.put(key.to_vec(), value);
        Ok(())
    }

    fn delete(&mut self, key: &[u8]) -> Result<()> {
        self.wal.append(&WalOp::Delete { key: key.to_vec() })?;
        self.memtable.delete(key);
        Ok(())
    }

    fn iter(&self) -> Box<dyn Iterator<Item = (Vec<u8>, Vec<u8>)> + '_> {
        Box::new(self.memtable.iter().filter_map(|(k, v)| {
            if let Value::Value(v) = v {
                Some((k.to_vec(), v.clone()))
            } else {
                None
            }
        }))
    }

    fn flush(&mut self) -> Result<()> {
        // In a real implementation, we would flush the MemTable to disk as an SSTable
        // and then clear the WAL. For now, we'll just flush the WAL.
        self.wal.flush()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_store_trait() {
        // This is just a compile-time test to ensure our trait is object-safe
        fn _assert_object_safe(_: &dyn Store) {}
    }
}
