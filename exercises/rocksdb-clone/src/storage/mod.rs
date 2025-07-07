//! Storage module for the RocksDB Clone
//! 
//! This module contains the storage-related types and implementations,
//! including the MemTable and (eventually) disk-based storage.

mod memtable;

pub use memtable::{MemTable, Value};

/// A result type for storage operations
pub type Result<T> = std::result::Result<T, crate::error::Error>;

/// Trait for key-value storage operations
pub trait Store {
    /// Retrieves a value by key
    fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>>;
    
    /// Inserts or updates a key-value pair
    fn put(&mut self, key: Vec<u8>, value: Vec<u8>) -> Result<()>;
    
    /// Deletes a key
    fn delete(&mut self, key: &[u8]) -> Result<()>;
    
    /// Returns an iterator over the key-value pairs
    fn iter(&self) -> Box<dyn Iterator<Item = (Vec<u8>, Vec<u8>)> + '_>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::Bytes;

    #[test]
    fn test_store_trait() {
        // This is just a compile-time test to ensure our trait is object-safe
        fn _assert_object_safe(_: &dyn Store) {}
    }
}
