//! Storage module for the RocksDB Clone
//!
//! This module contains the storage-related types and implementations,
//! including the MemTable, WAL, and (eventually) disk-based storage.

mod memtable;
mod sstable;
mod tests;
mod wal;

pub use memtable::{MemTable, Value};
pub use sstable::SSTable;
pub use wal::{WalOp, WriteAheadLog};

use std::{fs, path::Path};

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

/// A persistent key-value store that combines MemTable, WAL, and SSTables
pub struct PersistentStore {
    memtable: MemTable,
    wal: WriteAheadLog,
    sstables: Vec<SSTable>,
    data_dir: std::path::PathBuf,
    next_sstable_id: u64,
}

impl PersistentStore {
    /// Opens or creates a new persistent store at the given path
    pub fn open(path: impl AsRef<Path>) -> Result<Self> {
        let path = path.as_ref();

        fs::create_dir_all(path)?;

        let wal_path = path.join("wal.log");
        let mut memtable = MemTable::new();
        let mut sstables = Vec::new();
        let mut next_sstable_id = 0;

        // Scan for existing SSTables
        for entry in fs::read_dir(path)? {
            let entry = entry?;
            let path = entry.path();

            if let Some(ext) = path.extension() {
                if ext == "sst" {
                    // Extract ID from filename (format: id.sst)
                    if let Some(stem) = path.file_stem().and_then(|s| s.to_str()) {
                        if let Ok(id) = stem.parse::<u64>() {
                            next_sstable_id = next_sstable_id.max(id + 1);
                            sstables.push(SSTable::open(&path)?);
                        }
                    }
                }
            }
        }

        // Sort SSTables by ID (older first)
        sstables.sort_by_key(|sst| {
            sst.path()
                .file_stem()
                .and_then(|s| s.to_str())
                .and_then(|s| s.parse::<u64>().ok())
                .unwrap_or(0)
        });

        // Replay WAL to rebuild MemTable if it exists
        if wal_path.exists() {
            WriteAheadLog::replay(&wal_path, |op| {
                match op {
                    WalOp::Put { key, value } => {
                        memtable.put(key, value);
                    }
                    WalOp::Delete { key } => {
                        memtable.delete(key);
                    }
                }
                Ok(())
            })?;
        }

        let wal = WriteAheadLog::new(wal_path)?;

        Ok(Self {
            memtable,
            wal,
            sstables,
            data_dir: path.to_path_buf(),
            next_sstable_id,
        })
    }

    /// Flushes the current MemTable to a new SSTable
    fn flush_memtable(&mut self) -> Result<()> {
        // Skip if MemTable is empty
        if self.memtable.is_empty() {
            return Ok(());
        }

        // Create a new SSTable file
        let sstable_path = self
            .data_dir
            .join(format!("{:020}.sst", self.next_sstable_id));
        let mut sstable = SSTable::create(&sstable_path)?;

        // Get all entries from MemTable and convert to Vec<(Vec<u8>, Option<Vec<u8>>)>
        // where None represents a tombstone
        let entries: Vec<(Vec<u8>, Option<Vec<u8>>)> = self
            .memtable
            .iter()
            .map(|(k, v)| match v {
                Value::Value(v) => (k.to_vec(), Some(v.clone())),
                Value::Tombstone => (k.to_vec(), None), // Preserve tombstones as None
            })
            .collect();

        // Always write to SSTable, even if all entries are tombstones
        // This ensures deletions are properly persisted
        sstable.write_batch(&entries)?;
        self.sstables.push(sstable);
        self.next_sstable_id += 1;

        self.memtable.clear();
        self.wal.clear()?;

        Ok(())
    }

    /// Checks if the MemTable should be flushed to disk
    fn should_flush(&self) -> bool {
        // For now, just check if we have any data
        // In a real implementation, we'd check size thresholds
        !self.memtable.is_empty()
    }
}

impl Store for PersistentStore {
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>> {
        // First check MemTable (most recent data)
        if let Some(value) = self.memtable.get(key) {
            return match value {
                Value::Value(v) => Ok(Some(v.to_vec())),
                Value::Tombstone => {
                    return Ok(None);
                }
            };
        }

        // Then check SSTables in reverse order (newest first)
        for sstable in self.sstables.iter().rev() {
            match sstable.get(key)? {
                Some(value) => {
                    return Ok(Some(value));
                }
                None => {
                    return Ok(None);
                }
            }
        }

        Ok(None)
    }

    fn put(&mut self, key: &[u8], value: Vec<u8>) -> Result<()> {
        // Write to WAL first for durability
        self.wal.append(&WalOp::Put {
            key: key.to_vec(),
            value: value.clone(),
        })?;

        // Update MemTable with the value
        self.memtable.put(key.to_vec(), value);

        if self.should_flush() {
            self.flush_memtable()?;
        }

        Ok(())
    }

    fn delete(&mut self, key: &[u8]) -> Result<()> {
        self.wal.append(&WalOp::Delete { key: key.to_vec() })?;

        // Update MemTable with a tombstone
        self.memtable.delete(key);

        if self.should_flush() {
            self.flush_memtable()?;
        }

        Ok(())
    }

    fn iter(&self) -> Box<dyn Iterator<Item = (Vec<u8>, Vec<u8>)> + '_> {
        // For now, just return the MemTable iterator
        // In a real implementation, we'd merge iterators from all SSTables too
        Box::new(self.memtable.iter().filter_map(|(k, v)| match v {
            Value::Value(v) => Some((k.to_vec(), v.clone())),
            Value::Tombstone => None,
        }))
    }

    fn flush(&mut self) -> Result<()> {
        self.wal.flush()?;

        // If we have data in MemTable, flush it to a new SSTable
        if self.should_flush() {
            self.flush_memtable()?;
        }

        Ok(())
    }
}
