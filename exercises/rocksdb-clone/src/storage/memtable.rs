use crate::error::{Error, Result};
use bytes::Bytes;
use std::collections::BTreeMap;

/// Represents an entry in the MemTable
#[derive(Debug, Clone, PartialEq)]
pub enum Value {
    /// A value that exists in the database
    Value(Bytes),
    /// A tombstone indicating the key was deleted
    Tombstone,
}

/// An in-memory key-value store that maintains keys in sorted order
#[derive(Debug, Default)]
pub struct MemTable {
    /// The actual key-value storage
    map: BTreeMap<Bytes, Value>,
    /// Approximate size of the MemTable in bytes
    size: usize,
}

impl MemTable {
    /// Creates a new, empty MemTable
    pub fn new() -> Self {
        Self {
            map: BTreeMap::new(),
            size: 0,
        }
    }

    /// Inserts a key-value pair into the MemTable
    /// 
    /// # Arguments
    /// * `key` - The key to insert
    /// * `value` - The value to insert
    /// 
    /// # Returns
    /// The previous value if the key existed, or `None` if it didn't
    pub fn put(&mut self, key: Bytes, value: Bytes) -> Option<Value> {
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
    pub fn get(&self, key: &[u8]) -> Option<&Value> {
        self.map.get(key)
    }

    /// Deletes a key from the MemTable by inserting a tombstone
    /// 
    /// # Arguments
    /// * `key` - The key to delete
    /// 
    /// # Returns
    /// The previous value if the key existed, or `None` if it didn't
    pub fn delete(&mut self, key: Bytes) -> Option<Value> {
        let old_value = self.map.insert(key, Value::Tombstone);
        
        if let Some(Value::Value(val)) = &old_value {
            // Replace value with tombstone: remove value size, add 1 byte for tombstone
            self.size = self.size - val.len() + 1;
        } else if old_value.is_none() {
            // New tombstone for non-existent key
            self.size += 1;
        }
        
        old_value
    }

    /// Returns an iterator over the entries in the MemTable
    pub fn iter(&self) -> impl Iterator<Item = (&Bytes, &Value)> + '_ {
        self.map.iter()
    }

    /// Returns the approximate size of the MemTable in bytes
    pub fn size(&self) -> usize {
        self.size
    }

    /// Returns `true` if the MemTable is empty
    pub fn is_empty(&self) -> bool {
        self.map.is_empty()
    }

    /// Clears the MemTable, removing all key-value pairs
    pub fn clear(&mut self) {
        self.map.clear();
        self.size = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::Bytes;

    #[test]
    fn test_memtable_put_get() {
        let mut memtable = MemTable::new();
        
        let key1 = Bytes::from("key1");
        let value1 = Bytes::from("value1");
        
        // Test insert and get
        assert!(memtable.put(key1.clone(), value1.clone()).is_none());
        let retrieved = memtable.get(b"key1").unwrap();
        assert_eq!(retrieved, &Value::Value(value1));
        
        // Test update
        let new_value1 = Bytes::from("new_value1");
        let old_value = memtable.put(key1.clone(), new_value1.clone()).unwrap();
        assert_eq!(old_value, Value::Value(Bytes::from("value1")));
        
        // Test non-existent key
        assert!(memtable.get(b"non_existent").is_none());
    }
    
    #[test]
    fn test_memtable_delete() {
        let mut memtable = MemTable::new();
        let key = Bytes::from("key1");
        let value = Bytes::from("value1");
        
        // Insert and then delete
        memtable.put(key.clone(), value);
        memtable.delete(key.clone());
        
        // Should return tombstone
        assert_eq!(memtable.get(b"key1"), Some(&Value::Tombstone));
        
        // Delete non-existent key (should still create a tombstone)
        memtable.delete(Bytes::from("non_existent"));
        assert_eq!(memtable.get(b"non_existent"), Some(&Value::Tombstone));
    }
    
    #[test]
    fn test_memtable_size() {
        let mut memtable = MemTable::new();
        
        // Initial size should be 0
        assert_eq!(memtable.size(), 0);
        
        // Insert a key-value pair
        let key = Bytes::from("key1");
        let value = Bytes::from("value1");
        memtable.put(key, value);
        
        // Size should be key length + value length
        assert_eq!(memtable.size(), 4 + 6); // "key1" (4) + "value1" (6)
        
        // Update with a longer value
        memtable.put(Bytes::from("key1"), Bytes::from("new_value1"));
        assert_eq!(memtable.size(), 4 + 10); // "key1" (4) + "new_value1" (10)
        
        // Delete the key (replaces with tombstone)
        memtable.delete(Bytes::from("key1"));
        // Size should be just the key length + 1 (for tombstone)
        assert_eq!(memtable.size(), 4 + 1);
    }
}
