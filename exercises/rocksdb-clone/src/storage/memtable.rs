//! # MemTable Module
//!
//! This module implements an in-memory key-value store that maintains keys in sorted order.
//! It's a core component of the LSM tree storage engine, serving as the first layer of data storage.
//!
//! ## Key Concepts
//! - **MemTable**: An in-memory data structure that stores key-value pairs in sorted order.
//! - **Tombstone**: A special marker indicating that a key has been deleted.
//! - **Size Tracking**: The MemTable tracks its approximate size in bytes to determine when to flush to disk.

use std::collections::BTreeMap;

/// Represents an entry in the MemTable
/// Represents the value stored in the MemTable
///
/// In an LSM tree, values can be either actual data or tombstones.
/// Tombstones are used to mark keys as deleted during compaction.
#[derive(Debug, Clone, PartialEq)]
pub enum Value {
    /// A value that exists in the database
    Value(Vec<u8>),
    /// A tombstone indicating the key was deleted
    ///
    /// In LSM trees, deletes are implemented as special tombstone markers.
    /// During compaction, when a tombstone is encountered and there are no newer
    /// versions of the key, both the key and tombstone can be discarded.
    Tombstone,
}

/// An in-memory key-value store that maintains keys in sorted order
///
/// ## Implementation Details
/// - Uses a `BTreeMap` for in-memory storage, which keeps keys sorted and allows for efficient
///   range queries.
/// - Tracks the approximate size in bytes to determine when to flush to disk.
/// - Implements tombstone markers for deleted keys to support consistent reads during compaction.
///
/// ## Memory Management
/// The MemTable is designed to be flushed to disk (as an SSTable) when it reaches a certain size.
/// This helps control memory usage and provides durability.
#[derive(Debug, Default)]
pub struct MemTable {
    /// The actual key-value storage
    map: BTreeMap<Vec<u8>, Value>,
    /// Approximate size of the MemTable in bytes
    size: usize,
}

impl MemTable {
    /// Creates a new, empty MemTable
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::MemTable;
    ///
    /// let memtable = MemTable::new();
    /// assert!(memtable.is_empty());
    /// ```
    pub fn new() -> Self {
        Self {
            map: BTreeMap::new(),
            size: 0,
        }
    }

    /// Inserts a key-value pair into the MemTable
    ///
    /// If the key already exists, its value will be updated and the old value will be returned.
    /// The size tracking is automatically updated to reflect the change in storage requirements.
    ///
    /// # Arguments
    /// * `key` - The key to insert
    /// * `value` - The value to insert
    ///
    /// # Returns
    /// The previous value if the key existed, or `None` if it didn't
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::{MemTable, Value};
    ///
    /// let mut memtable = MemTable::new();
    /// assert!(memtable.put(b"key", b"value1").is_none());
    /// assert_eq!(memtable.put(b"key", b"value2"), Some(Value::Value(b"value1".to_vec())));
    /// ```
    pub fn put(&mut self, key: impl Into<Vec<u8>>, value: impl Into<Vec<u8>>) -> Option<Value> {
        let key = key.into();
        let value = value.into();
        let key_size = key.len();
        let value_size = value.len();

        let old_value = self.map.insert(key, Value::Value(value));

        match &old_value {
            Some(Value::Value(old_val)) => {
                // Update size: remove old value size, add new value size
                self.size = self.size - old_val.len() + value_size;
            }
            Some(Value::Tombstone) => {
                // Replace tombstone with new value
                self.size = self.size - 1 + value_size;
            }
            None => {
                // New entry: add both key and value sizes
                self.size += key_size + value_size;
            }
        }

        old_value
    }

    /// Retrieves a value by key
    ///
    /// # Arguments
    /// * `key` - The key to look up
    ///
    /// # Returns
    /// - `Some(Value::Value(_))` if the key exists
    /// - `Some(Value::Tombstone)` if the key was deleted
    /// - `None` if the key was never inserted
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::{MemTable, Value};
    ///
    /// let mut memtable = MemTable::new();
    /// memtable.put(b"key", b"value");
    /// assert_eq!(memtable.get(b"key"), Some(&Value::Value(b"value".to_vec())));
    ///
    /// memtable.delete(b"key");
    /// assert_eq!(memtable.get(b"key"), Some(&Value::Tombstone));
    ///
    /// assert_eq!(memtable.get(b"nonexistent"), None);
    /// ```
    pub fn get(&self, key: &[u8]) -> Option<&Value> {
        self.map.get(key)
    }

    /// Deletes a key from the MemTable by inserting a tombstone
    ///
    /// In LSM trees, deletes are implemented as special tombstone markers.
    /// The actual removal of the key happens during compaction.
    ///
    /// # Arguments
    /// * `key` - The key to delete
    ///
    /// # Returns
    /// The previous value if the key existed, or `None` if it didn't
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::{MemTable, Value};
    ///
    /// let mut memtable = MemTable::new();
    /// memtable.put(b"key", b"value");
    /// assert_eq!(memtable.delete(b"key"), Some(Value::Value(b"value".to_vec())));
    /// assert_eq!(memtable.get(b"key"), Some(&Value::Tombstone));
    ///
    /// // Deleting a non-existent key
    /// assert_eq!(memtable.delete(b"nonexistent"), None);
    /// ```
    pub fn delete<K: Into<Vec<u8>>>(&mut self, key: K) -> Option<Value> {
        let key = key.into();
        let old_value = self.map.insert(key.clone(), Value::Tombstone);

        match &old_value {
            Some(Value::Value(val)) => {
                // Replace value with tombstone: remove value size, add 1 byte for tombstone
                self.size = self.size - val.len() + 1;
            }
            Some(Value::Tombstone) => {
                // Already a tombstone, no size change
            }
            None => {
                // New tombstone: add key size + 1 byte for tombstone
                self.size += key.len() + 1;
            }
        }

        old_value
    }

    /// Returns an iterator over the entries in the MemTable
    ///
    /// The iterator yields key-value pairs in sorted order by key.
    /// Both regular values and tombstones are included in the iteration.
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::{MemTable, Value};
    ///
    /// let mut memtable = MemTable::new();
    /// memtable.put(b"b", b"value_b");
    /// memtable.put(b"a", b"value_a");
    ///
    /// let mut iter = memtable.iter();
    /// assert_eq!(iter.next(), Some((&b"a"[..], &Value::Value(b"value_a".to_vec()))));
    /// assert_eq!(iter.next(), Some((&b"b"[..], &Value::Value(b"value_b".to_vec()))));
    /// assert_eq!(iter.next(), None);
    /// ```
    pub fn iter(&self) -> impl Iterator<Item = (&[u8], &Value)> + '_ {
        self.map.iter().map(|(k, v)| (k.as_slice(), v))
    }

    /// Returns the approximate size of the MemTable in bytes
    ///
    /// The size includes:
    /// - The size of all keys
    /// - The size of all values (for Value::Value variants)
    /// - 1 byte per tombstone (for Value::Tombstone variants)
    ///
    /// This is an approximation used to determine when to flush the MemTable to disk.
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::MemTable;
    ///
    /// let mut memtable = MemTable::new();
    /// assert_eq!(memtable.size(), 0);
    ///
    /// // Key "a" (1 byte) + Value "value" (5 bytes) = 6 bytes
    /// memtable.put(b"a", b"value");
    /// assert_eq!(memtable.size(), 6);
    ///
    /// // Deleting replaces the value with a 1-byte tombstone
    /// memtable.delete(b"a");
    /// assert_eq!(memtable.size(), 2); // 1 byte key + 1 byte tombstone
    /// ```
    pub fn size(&self) -> usize {
        self.size
    }

    /// Returns `true` if the MemTable is empty
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::MemTable;
    ///
    /// let mut memtable = MemTable::new();
    /// assert!(memtable.is_empty());
    ///
    /// memtable.put(b"key", b"value");
    /// assert!(!memtable.is_empty());
    ///
    /// memtable.delete(b"key");
    /// assert!(!memtable.is_empty()); // Still contains a tombstone
    ///
    /// let mut empty_memtable = MemTable::new();
    /// assert!(empty_memtable.is_empty());
    /// ```
    pub fn is_empty(&self) -> bool {
        self.map.is_empty()
    }

    /// Clears the MemTable, removing all key-value pairs
    ///
    /// This resets the MemTable to its initial empty state.
    ///
    /// # Examples
    /// ```
    /// use rocksdb_clone::storage::MemTable;
    ///
    /// let mut memtable = MemTable::new();
    /// memtable.put(b"key", b"value");
    /// assert!(!memtable.is_empty());
    ///
    /// memtable.clear();
    /// assert!(memtable.is_empty());
    /// assert_eq!(memtable.size(), 0);
    /// ```
    pub fn clear(&mut self) {
        self.map.clear();
        self.size = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_memtable_put_get() {
        let mut memtable = MemTable::new();

        let key1 = b"key1".to_vec();
        let value1 = b"value1".to_vec();

        assert!(memtable.put(key1.clone(), value1.clone()).is_none());
        assert_eq!(memtable.get(&key1), Some(&Value::Value(value1.clone())));

        let value2 = b"value2".to_vec();
        assert_eq!(
            memtable.put(key1.clone(), value2.clone()),
            Some(Value::Value(value1))
        );
        assert_eq!(memtable.get(&key1), Some(&Value::Value(value2)));

        assert_eq!(memtable.get(b"nonexistent"), None);
    }

    #[test]
    fn test_memtable_delete() {
        let mut memtable = MemTable::new();

        let key = b"key1".to_vec();
        let value = b"value1".to_vec();

        memtable.put(key.clone(), value);
        assert!(memtable.get(&key).is_some());

        memtable.delete(key.clone());
        assert_eq!(memtable.get(&key), Some(&Value::Tombstone));

        assert!(memtable.delete(b"nonexistent".to_vec()).is_none());
    }

    #[test]
    fn test_memtable_size() {
        let mut memtable = MemTable::new();

        assert_eq!(memtable.size(), 0);

        let key1 = b"key1".to_vec();
        let value1 = b"value1".to_vec();
        let value2 = b"value2".to_vec();
        let value3 = b"new_value1".to_vec();

        // Size should be key length + value length
        assert_eq!(memtable.size(), 0);
        memtable.put(key1.clone(), value1);
        assert_eq!(memtable.size(), 4 + 6); // "key1" (4) + "value1" (6)

        // Update with a different value of same length
        memtable.put(key1.clone(), value2);
        assert_eq!(memtable.size(), 4 + 6); // "key1" (4) + "value2" (6)

        // Update with a longer value
        memtable.put(key1.clone(), value3);
        assert_eq!(memtable.size(), 4 + 10); // "key1" (4) + "new_value1" (10)

        // Delete the key (replaces with tombstone)
        memtable.delete(key1.clone());
        // Tombstone adds 1 byte to the size (4 bytes for key + 1 byte for tombstone)
        assert_eq!(memtable.size(), 5); // Key size (4) + tombstone (1)

        memtable.clear();
        assert_eq!(memtable.size(), 0);
    }
}
