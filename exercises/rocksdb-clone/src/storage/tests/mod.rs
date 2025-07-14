//! Unit tests for the SSTable implementation
//!
//! These tests verify the core functionality of the SSTable, including:
//! - Creating new SSTables
//! - Writing and reading key-value pairs
//! - Handling tombstones (deletions)
//! - Error conditions and edge cases
//!
//! Tests use temporary directories that are automatically cleaned up.

use crate::storage::SSTable;
use std::error::Error;

/// Tests basic SSTable operations including creation, writing, and reading
#[test]
fn test_sstable_basic() -> Result<(), Box<dyn Error>> {
    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    let mut sstable = SSTable::create(&sstable_path)?;

    sstable.write_batch(&[
        (b"key1".to_vec(), Some(b"value1".to_vec())),
        (b"key2".to_vec(), Some(b"value2".to_vec())),
        (b"key3".to_vec(), Some(b"value3".to_vec())),
    ])?;

    assert_eq!(sstable.get(b"key1")?, Some(b"value1".to_vec()));
    assert_eq!(sstable.get(b"key2")?, Some(b"value2".to_vec()));
    assert_eq!(sstable.get(b"key3")?, Some(b"value3".to_vec()));

    // Test with a tombstone (deletion)
    sstable.write_batch(&[(b"key2".to_vec(), None)])?;
    assert_eq!(sstable.get(b"key2")?, None);

    assert_eq!(sstable.get(b"nonexistent")?, None);

    Ok(())
}

/// Tests handling of empty keys and values
#[test]
fn test_sstable_empty_keys_and_values() -> Result<(), Box<dyn Error>> {
    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    // Test empty key with non-empty value
    let mut sstable = SSTable::create(&sstable_path)?;
    sstable.write_batch(&[("".as_bytes().to_vec(), Some(b"value".to_vec()))])?;
    assert_eq!(sstable.get(b"")?, Some(b"value".to_vec()));

    // Test non-empty key with empty value
    sstable.write_batch(&[(b"key".to_vec(), Some(vec![]))])?;
    assert_eq!(sstable.get(b"key")?, Some(vec![]));

    // Test empty key with empty value
    sstable.write_batch(&[(vec![], Some(vec![]))])?;
    assert_eq!(sstable.get(b"")?, Some(vec![]));

    // Test tombstone with empty key
    sstable.write_batch(&[(vec![], None)])?;
    assert_eq!(sstable.get(b"")?, None);

    Ok(())
}

/// Tests handling of very large keys and values
#[test]
fn test_sstable_large_entries() -> Result<(), Box<dyn Error>> {
    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    let large_key = vec![b'x'; 1024 * 1024]; // 1MB key
    let large_value = vec![b'y'; 2 * 1024 * 1024]; // 2MB value

    let mut sstable = SSTable::create(&sstable_path)?;
    sstable.write_batch(&[(large_key.clone(), Some(large_value.clone()))])?;

    assert_eq!(sstable.get(&large_key)?, Some(large_value));

    // Test with tombstone
    sstable.write_batch(&[(large_key.clone(), None)])?;
    assert_eq!(sstable.get(&large_key)?, None);

    Ok(())
}

/// Tests handling of many key-value pairs and tombstones
#[test]
fn test_sstable_many_entries() -> Result<(), Box<dyn Error>> {
    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    let mut entries = Vec::new();
    for i in 0..1000 {
        let key = format!("key_{:04}", i).into_bytes();
        let value = format!("value_{}", i).into_bytes();
        entries.push((key, Some(value)));
    }

    let mut sstable = SSTable::create(&sstable_path)?;
    sstable.write_batch(&entries)?;

    // Verify all entries can be read back
    for (key, value) in entries.clone() {
        assert_eq!(sstable.get(&key)?, value);
    }

    // Create a new SSTable for tombstones
    let tombstone_path = temp_dir.path().join("test_sstable_tombstones");
    let mut tombstone_table = SSTable::create(&tombstone_path)?;

    // Write tombstones for even-numbered keys
    let mut tombstones = Vec::new();
    for i in 0..1000 {
        if i % 2 == 0 {
            // Delete even-numbered keys
            let key = format!("key_{:04}", i).into_bytes();
            tombstones.push((key, None));
        }
    }
    tombstone_table.write_batch(&tombstones)?;

    for i in 0..1000 {
        let key_str = format!("key_{:04}", i);
        let key = key_str.as_bytes();
        match tombstone_table.get(key)? {
            Some(_) => println!("Tombstone table has key: {}", key_str),
            None => {}
        }
    }

    let tombstone_file = std::fs::read(&tombstone_path)?;

    // Try to read the file as a string to see if it's text
    if let Ok(s) = String::from_utf8(tombstone_file.clone()) {
        println!("\nFile as UTF-8 string (first 200 chars):");
        println!("{}", &s[..std::cmp::min(200, s.len())]);
    } else {
        println!("\nFile is not valid UTF-8");
    }

    println!("=== End Raw Tombstone Table File Contents ===\n");

    // In a real LSM tree, we'd check newer SSTables first, then older ones
    // For this test, we'll check both tables manually
    for i in 0..1000 {
        let key_str = format!("key_{:04}", i);
        let key = key_str.as_bytes();
        println!("\n--- Checking key: {} ---", key_str);

        // Check if key exists in tombstone table
        let tombstone_result = tombstone_table.get(key)?;
        println!("Tombstone table lookup: {:?}", tombstone_result);

        // First check the tombstone table (newer)
        let result = if i % 2 == 0 {
            // Even-numbered keys should be tombstones
            match tombstone_result {
                None => {
                    println!("Found tombstone for even key {}, returning None", key_str);
                    None
                }
                Some(_) => {
                    println!(
                        "ERROR: Expected tombstone for even key {}, but got value!",
                        key_str
                    );
                    tombstone_result
                }
            }
        } else {
            // Odd-numbered keys should not be in tombstone table, check original table
            match tombstone_result {
                None => {
                    println!(
                        "Odd key {} not in tombstone table, checking original table",
                        key_str
                    );
                    sstable.get(key)?
                }
                Some(_) => {
                    println!("ERROR: Odd key {} found in tombstone table!", key_str);
                    tombstone_result
                }
            }
        };

        println!("Final result for {}: {:?}", key_str, result);

        if i % 2 == 0 {
            // Should be deleted (tombstone)
            assert_eq!(
                result, None,
                "Expected key {} to be deleted (tombstone)",
                key_str
            );
        } else {
            // Should still exist with original value
            let expected_value = format!("value_{}", i).into_bytes();
            println!("Expected value for {}: {:?}", key_str, expected_value);
            assert_eq!(
                result,
                Some(expected_value),
                "Key {} should have its original value",
                key_str
            );
        }
    }

    Ok(())
}

/// Tests that we can reopen an SSTable and read the data
#[test]
fn test_sstable_reopen() -> Result<(), Box<dyn Error>> {
    let _ = env_logger::builder().is_test(true).try_init();

    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    // Create and write some data
    {
        let mut sstable = SSTable::create(&sstable_path)?;

        sstable.write_batch(&[
            (b"key1".to_vec(), Some(b"value1".to_vec())),
            (b"key2".to_vec(), None), // Tombstone
            (b"key3".to_vec(), Some(b"value3".to_vec())),
        ])?;
    }

    // Read and log the raw file content for inspection
    let file_content = std::fs::read(&sstable_path)?;

    // Try to read the footer directly
    if file_content.len() >= 16 {
        let footer = &file_content[file_content.len() - 16..];

        // Try to parse the metadata length
        if let Ok(meta_len) = <[u8; 8]>::try_from(&footer[0..8]) {
            let meta_len = u64::from_le_bytes(meta_len);
            log::debug!("Parsed metadata length from footer: {}", meta_len);
        } else {
            log::warn!("Could not parse metadata length from footer");
        }
    }

    let sstable = SSTable::open(&sstable_path)?;

    assert_eq!(sstable.get(b"key1")?, Some(b"value1".to_vec()));
    assert_eq!(sstable.get(b"key2")?, None);
    assert_eq!(sstable.get(b"key3")?, Some(b"value3".to_vec()));

    Ok(())
}

/// Tests error conditions
#[test]
fn test_sstable_errors() -> Result<(), Box<dyn Error>> {
    let temp_dir = tempfile::tempdir()?;
    let sstable_path = temp_dir.path().join("test_sstable");

    let mut sstable = SSTable::create(&sstable_path)?;

    sstable.write_batch(&[
        (b"key1".to_vec(), Some(b"value1".to_vec())),
        (b"key2".to_vec(), None), // Tombstone
        (b"key3".to_vec(), Some(b"value3".to_vec())),
    ])?;

    assert_eq!(sstable.get(b"key1")?, Some(b"value1".to_vec()));
    assert_eq!(sstable.get(b"key2")?, None); // Tombstone should return None
    assert_eq!(sstable.get(b"key3")?, Some(b"value3".to_vec()));
    assert_eq!(sstable.get(b"nonexistent")?, None); // Non-existent key

    // Test opening non-existent file
    let non_existent = temp_dir.path().join("nonexistent");
    assert!(SSTable::open(&non_existent).is_err());

    // Test creating existing file
    assert!(SSTable::create(&sstable_path).is_err());

    // Test corrupted file
    std::fs::write(&sstable_path, b"corrupted data")?;
    assert!(SSTable::open(&sstable_path).is_err());

    Ok(())
}
