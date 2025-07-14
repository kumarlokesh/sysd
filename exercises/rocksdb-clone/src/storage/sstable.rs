//! SSTable (Sorted String Table) implementation for persistent storage of key-value pairs.
//!
//! # File Format
//!
//! SSTables are immutable, sorted files that store key-value pairs. They consist of:
//! 1. Data section: Sequence of key-value pairs
//! 2. Index section: Maps keys to their offsets in the data section
//! 3. Metadata: Information about the SSTable (number of entries, data size, index size)
//! 4. Magic number: For file format validation
//!
//! # Tombstone Handling
//!
//! Tombstones (deletions) are represented by a special value length (u64::MAX) with no data.
//! When reading, a tombstone is returned as `None`.
//!
//! # Lookup Process
//!
//! 1. Check the in-memory index to find the key's offset in the data section
//! 2. If found, seek to the offset and read the value
//! 3. If the value length is u64::MAX, it's a tombstone and we return None
//! 4. If not found in the index, the key doesn't exist in this SSTable
//!
//! # Write Process
//!
//! 1. Write all key-value pairs to the data section, keeping track of offsets
//! 2. Write the index section with key-offset mappings
//! 3. Write metadata (number of entries, data size, index size)
//! 4. Write magic number for validation
//!
//! # Implementation Notes
//!
//! - Uses fixed-size encoding for metadata to ensure reliable deserialization
//! - All numbers are stored in big-endian format for consistency
//! - The file is truncated and rewritten on each write to ensure consistency

use crate::error::{Error, Result};
use bincode::{Decode, Encode};
use serde::{Deserialize, Serialize};
use std::{
    fs::{File, OpenOptions},
    io::{self, BufReader, BufWriter, Read, Seek, SeekFrom, Write},
    path::{Path, PathBuf},
};

// Magic number for SSTable file format validation
#[allow(dead_code)]
/// Magic number used to identify SSTable files
const MAGIC_NUMBER: u64 = 0x1234567890ABCDEF;

/// Represents metadata about an SSTable file
///
/// The metadata is stored at the end of the SSTable file and includes:
/// - Number of key-value entries
/// - Size of the data section
/// - Size of the index section
///
/// The metadata is serialized using fixed-size encoding for reliability.
pub struct SSTable {
    file: File,
    path: PathBuf,
}

/// Size of the SSTable footer in bytes (magic number + metadata length + metadata offset)
const FOOTER_SIZE: u64 = 24;

/// Represents the metadata for an SSTable
#[derive(Debug, Serialize, Deserialize, Encode, Decode)]
struct SSTableMeta {
    /// Number of entries in the SSTable
    num_entries: u64,
    /// Size of the data section in bytes
    data_size: u64,
    /// Size of the index section in bytes
    index_size: u64,
}

impl SSTable {
    /// Creates a new SSTable at the given path
    ///
    /// # Arguments
    /// * `path` - Filesystem path where the SSTable file will be created
    ///
    /// # Returns
    /// A new `SSTable` instance if successful, or an error if the file cannot be created
    ///
    /// # Example
    /// ```no_run
    /// use rocksdb_clone::storage::SSTable;
    ///
    /// let sstable = SSTable::create("path/to/sstable").unwrap();
    /// ```
    pub fn create(path: impl AsRef<Path>) -> Result<Self> {
        let path = path.as_ref();
        log::debug!("Creating new SSTable at: {}", path.display());

        if path.exists() {
            return Err(Error::custom("File already exists"));
        }

        let file = OpenOptions::new()
            .create(true)
            .truncate(true)
            .read(true)
            .write(true)
            .open(&path)
            .map_err(|e| {
                let msg = format!("Failed to create SSTable file at {}: {}", path.display(), e);
                log::error!("{}", msg);
                Error::Io(std::io::Error::new(std::io::ErrorKind::Other, msg))
            })?;

        Ok(Self {
            file,
            path: path.to_path_buf(),
        })
    }

    /// Returns the path to this SSTable file
    ///
    /// # Returns
    /// A reference to the path where this SSTable is stored
    ///
    /// # Example
    /// ```no_run
    /// # use rocksdb_clone::storage::SSTable;
    /// # let sstable = SSTable::create("path/to/sstable").unwrap();
    /// let path = sstable.path();
    /// println!("SSTable path: {:?}", path);
    /// ```
    pub fn path(&self) -> &Path {
        &self.path
    }

    /// Opens an existing SSTable for reading
    ///
    /// # Arguments
    /// * `path` - Filesystem path to the existing SSTable file
    ///
    /// # Returns
    /// An `SSTable` instance if the file exists and is valid, or an error
    ///
    /// # Errors
    /// Returns an error if the file doesn't exist, is not a valid SSTable, or cannot be opened
    ///
    /// # Example
    /// ```no_run
    /// # use rocksdb_clone::storage::SSTable;
    /// let sstable = SSTable::open("path/to/existing/sstable").unwrap();
    /// ```
    pub fn open(path: impl AsRef<Path>) -> Result<Self> {
        let path = path.as_ref();
        let file = OpenOptions::new().read(true).open(path)?;
        let sstable = Self {
            file,
            path: path.to_path_buf(),
        };

        sstable.verify_metadata()?;

        Ok(sstable)
    }

    /// Verifies that the SSTable file has valid metadata
    fn verify_metadata(&self) -> Result<()> {
        // The footer is the last 24 bytes of the file
        // It contains: [magic_number (8)][meta_len (8)][meta_offset (8)]
        const FOOTER_SIZE: u64 = 24; // 8 (magic) + 8 (meta_len) + 8 (meta_offset)

        let file_size = self.file.metadata()?.len();
        if file_size < FOOTER_SIZE {
            return Err(crate::error::Error::custom(
                "SSTable file is too small to contain a valid footer",
            ));
        }

        // Read the footer
        let footer_start = file_size - FOOTER_SIZE;

        let mut footer = vec![0u8; FOOTER_SIZE as usize];
        {
            let mut file = &self.file;
            file.seek(SeekFrom::Start(footer_start))?;
            file.read_exact(&mut footer)?;
        }

        // The first 8 bytes of the footer should be the magic number
        let magic_bytes = &footer[0..8];

        if magic_bytes != MAGIC_NUMBER.to_be_bytes() {
            return Err(crate::error::Error::custom(format!(
                "Invalid SSTable magic number: {:?}",
                magic_bytes
            )));
        }

        // The next 8 bytes are the metadata length in little-endian
        let meta_len_bytes = &footer[8..16];
        let meta_len = u64::from_le_bytes(meta_len_bytes.try_into().map_err(|_| {
            crate::error::Error::custom("Failed to parse metadata length from footer")
        })?);

        // Verify the metadata length is reasonable
        if meta_len == 0 || meta_len > 1024 {
            return Err(crate::error::Error::custom(format!(
                "Invalid metadata length in SSTable: {} (0x{:x})",
                meta_len, meta_len
            )));
        }

        // The next 8 bytes are the metadata offset in little-endian
        let meta_offset_bytes = &footer[16..24];
        let meta_offset = u64::from_le_bytes(meta_offset_bytes.try_into().map_err(|_| {
            crate::error::Error::custom("Failed to parse metadata offset from footer")
        })?);

        // Verify the metadata offset is within the file
        if meta_offset >= file_size - FOOTER_SIZE {
            return Err(crate::error::Error::custom(format!(
                "Invalid metadata offset in SSTable: {} (file size: {})",
                meta_offset, file_size
            )));
        }

        // Verify the metadata ends before the footer
        if meta_offset + meta_len > file_size - FOOTER_SIZE {
            return Err(crate::error::Error::custom(format!(
                "Metadata extends beyond footer in SSTable: offset={}, len={}, footer_start={}",
                meta_offset,
                meta_len,
                file_size - FOOTER_SIZE
            )));
        }

        let mut meta_buf = vec![0u8; meta_len as usize];
        {
            let mut file = &self.file;
            file.seek(SeekFrom::Start(meta_offset))?;
            file.read_exact(&mut meta_buf)?;
        }

        Ok(())
    }

    /// Writes a batch of key-value pairs to the SSTable
    ///
    /// This method will completely overwrite any existing data in the SSTable.
    /// Tombstones (deletions) are represented by `None` values in the input.
    ///
    /// # Arguments
    /// * `entries` - Slice of key-value pairs where the value is an Option:
    ///   - `Some(Vec<u8>)`: A regular key-value pair
    ///   - `None`: A tombstone (deletion marker)
    ///
    /// # Returns
    /// `Ok(())` on success, or an error if the write fails
    ///
    /// # Example
    /// ```no_run
    /// # use rocksdb_clone::storage::SSTable;
    /// # let mut sstable = SSTable::create("path/to/sstable").unwrap();
    /// // Write some data
    /// sstable.write_batch(&[
    ///     (b"key1".to_vec(), Some(b"value1".to_vec())),
    ///     (b"key2".to_vec(), None),  // Tombstone
    /// ]).unwrap();
    /// ```
    pub fn write_batch(&mut self, entries: &[(Vec<u8>, Option<Vec<u8>>)]) -> Result<()> {
        // Create a new BufWriter for the file
        let file = &mut self.file;
        file.seek(SeekFrom::Start(0))?; // Start from beginning of file
        file.set_len(0)?; // Truncate the file

        let mut writer = BufWriter::new(file);

        // Write data section
        let data_start = 0; // Start at beginning of file
        let mut current_offset = data_start;

        // First, collect all entries with their offsets
        let mut index_entries = Vec::new();

        // Write all key-value pairs and record their offsets
        for (key, maybe_value) in entries {
            // Record the current position before writing the key
            let entry_offset = current_offset;

            // Calculate the size of the key header (8 bytes for key length)
            let key_header_size = 8;

            // Write key length (8 bytes) + key
            writer.write_all(&(key.len() as u64).to_le_bytes())?;
            writer.write_all(key)?;

            // For None (tombstone), write u64::MAX as the length
            // For Some(value), write the actual value length
            match maybe_value {
                Some(value) => {
                    // Write value length + value
                    writer.write_all(&(value.len() as u64).to_le_bytes())?;
                    writer.write_all(value)?;

                    // Update current offset: key header + key + value header + value
                    current_offset +=
                        key_header_size as u64 + key.len() as u64 + 8 + value.len() as u64;
                }
                None => {
                    // Tombstone: write u64::MAX as length and no value
                    let tombstone_marker = u64::MAX;
                    writer.write_all(&tombstone_marker.to_le_bytes())?;

                    // Update current offset: key header + key + tombstone marker (8 bytes)
                    current_offset += key_header_size as u64 + key.len() as u64 + 8;
                }
            }

            // Add the entry to the index with the correct offset
            // For tombstones, we still need to add them to the index so they can override previous values
            index_entries.push((key.clone(), entry_offset));
        }

        // Make sure all data is written to the underlying file
        writer.flush()?;

        // Get the current position for the start of the index section
        let index_start = current_offset;

        // Write index entries
        for (key, offset) in &index_entries {
            // Write key length (8 bytes) + key + offset (8 bytes)
            writer.write_all(&(key.len() as u64).to_le_bytes())?;
            writer.write_all(key)?;
            writer.write_all(&offset.to_le_bytes())?;
        }

        // Make sure all index entries are written
        writer.flush()?;

        // Get the current position for the end of the index section
        let index_end = writer.stream_position()?;
        let index_size = index_end - index_start;

        // Calculate data size (from start of file to start of index)
        let data_size = index_start;

        // Write footer with metadata
        let meta = SSTableMeta {
            num_entries: entries.len() as u64,
            data_size,
            index_size,
        };

        // Serialize metadata to a buffer with fixed-size encoding
        let config = bincode::config::standard()
            .with_fixed_int_encoding()
            .with_big_endian();

        let meta_bytes = bincode::encode_to_vec(&meta, config).map_err(|e| {
            io::Error::new(
                io::ErrorKind::InvalidData,
                format!("Failed to encode SSTable metadata: {}", e),
            )
        })?;

        // Calculate the footer position and write metadata length (8 bytes)
        let meta_len = meta_bytes.len() as u64;
        let meta_len_bytes = meta_len.to_le_bytes();

        // First, write the data and index blocks
        // The footer will be written after the metadata

        // Get the current position where we'll write the metadata
        let meta_start = writer.stream_position()?;

        // Write metadata first
        writer.write_all(&meta_bytes)?;
        log::debug!(
            "Wrote metadata bytes ({}): {:?}",
            meta_bytes.len(),
            &meta_bytes
        );

        // Then write the footer at the end of the file
        // Footer structure: [magic_number (8)][meta_len (8)][meta_offset (8)]
        let magic_bytes = MAGIC_NUMBER.to_be_bytes();

        // Write magic number (8 bytes)
        writer.write_all(&magic_bytes)?;

        // Write metadata length (8 bytes)
        writer.write_all(&meta_len_bytes)?;

        // Write metadata start offset (8 bytes)
        let meta_start_bytes = meta_start.to_le_bytes();
        writer.write_all(&meta_start_bytes)?;

        // Log the exact bytes being written to the footer
        let mut footer = Vec::new();
        footer.extend_from_slice(&magic_bytes);
        footer.extend_from_slice(&meta_len_bytes);
        footer.extend_from_slice(&meta_start_bytes);

        // Verify the footer can be read back correctly
        let read_magic = &footer[0..8];
        let read_meta_len = u64::from_le_bytes(footer[8..16].try_into().unwrap());
        let read_meta_offset = u64::from_le_bytes(footer[16..24].try_into().unwrap());

        if read_magic != MAGIC_NUMBER.to_be_bytes() {
            return Err(crate::error::Error::custom(format!(
                "Footer verification failed: invalid magic number: {:?}",
                read_magic
            )));
        }

        if read_meta_len != meta_len {
            return Err(crate::error::Error::custom(format!(
                "Footer verification failed: expected meta_len={}, got {}",
                meta_len, read_meta_len
            )));
        }

        if read_meta_offset != meta_start {
            return Err(crate::error::Error::custom(format!(
                "Footer verification failed: expected meta_offset={}, got {}",
                meta_start, read_meta_offset
            )));
        }

        // Ensure everything is written to disk
        writer.flush()?;

        Ok(())
    }

    /// Looks up a key in the SSTable
    ///
    /// # Arguments
    /// * `key` - The key to look up
    ///
    /// # Returns
    /// - `Ok(Some(Vec<u8>))` if the key exists and has a value
    /// - `Ok(None)` if the key has a tombstone or doesn't exist
    /// - `Err(_)` if there was an error reading the SSTable
    ///
    /// # Example
    /// ```no_run
    /// # use rocksdb_clone::storage::SSTable;
    /// # let mut sstable = SSTable::create("path/to/sstable").unwrap();
    /// # sstable.write_batch(&[(b"key1".to_vec(), Some(b"value1".to_vec()))]).unwrap();
    /// // Look up a key
    /// if let Some(value) = sstable.get(b"key1").unwrap() {
    ///     println!("Found value: {:?}", value);
    /// } else {
    ///     println!("Key not found or deleted");
    /// }
    /// ```
    pub fn get(&self, key: &[u8]) -> Result<Option<Vec<u8>>> {
        let mut file = BufReader::new(File::open(&self.path)?);

        // Get file size
        let file_size = file.seek(SeekFrom::End(0))?;

        if file_size < FOOTER_SIZE {
            return Ok(None);
        }

        // Read footer
        let footer_start = file_size - FOOTER_SIZE;
        file.seek(SeekFrom::Start(footer_start))?;

        let mut footer = [0u8; FOOTER_SIZE as usize];
        file.read_exact(&mut footer)?;

        // Parse footer
        let magic_bytes = &footer[0..8];
        let meta_len = u64::from_le_bytes(footer[8..16].try_into().map_err(|_| {
            io::Error::new(
                io::ErrorKind::InvalidData,
                "Failed to parse metadata length from footer",
            )
        })?);

        let meta_offset = u64::from_le_bytes(footer[16..24].try_into().map_err(|_| {
            io::Error::new(
                io::ErrorKind::InvalidData,
                "Failed to parse metadata offset from footer",
            )
        })?);

        // Verify magic number
        if magic_bytes != MAGIC_NUMBER.to_be_bytes() {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!(
                    "Invalid SSTable: magic number mismatch. Got: {:?}, expected: {:?}",
                    magic_bytes,
                    MAGIC_NUMBER.to_be_bytes()
                ),
            )
            .into());
        }

        // Verify metadata offset and length
        if meta_offset >= footer_start || meta_len == 0 || meta_len > 1024 {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!(
                    "Invalid metadata in SSTable: offset={}, len={}",
                    meta_offset, meta_len
                ),
            )
            .into());
        }

        file.seek(SeekFrom::Start(meta_offset))?;

        let mut meta_buf = vec![0u8; meta_len as usize];
        file.read_exact(&mut meta_buf)?;

        // Decode the metadata with the same config used for encoding
        let config = bincode::config::standard()
            .with_fixed_int_encoding()
            .with_big_endian();

        let meta = match bincode::decode_from_slice::<SSTableMeta, _>(&meta_buf, config) {
            Ok((meta, bytes_read)) => {
                log::debug!("Successfully decoded metadata, bytes read: {}", bytes_read);
                meta
            }
            Err(e) => {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    format!("Failed to decode SSTable metadata: {}", e),
                )
                .into());
            }
        };

        // Verify metadata values make sense
        if meta.num_entries == 0 || meta.data_size == 0 || meta.index_size == 0 {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!("Invalid metadata values in SSTable: {:?}", meta),
            )
            .into());
        }

        let data_end = meta.data_size;
        let index_start = data_end;
        let index_end = meta_offset;

        if index_end <= index_start || index_end > meta_offset {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!(
                    "Invalid index section in SSTable: start={}, end={}, meta_offset={}",
                    index_start, index_end, meta_offset
                ),
            )
            .into());
        }

        // Read index entries
        file.seek(SeekFrom::Start(index_start))?;

        let mut entry_count = 0;
        let mut _last_key = None;

        while file.stream_position()? < index_end {
            entry_count += 1;

            // Read key length (8 bytes)
            let mut key_len_buf = [0u8; 8];
            file.read_exact(&mut key_len_buf)?;
            let key_len = u64::from_le_bytes(key_len_buf) as usize;

            // Read key
            let mut key_buf = vec![0u8; key_len];
            file.read_exact(&mut key_buf)?;

            // Read value offset
            let mut offset_buf = [0u8; 8];
            file.read_exact(&mut offset_buf)?;
            let value_offset = u64::from_le_bytes(offset_buf);

            let current_key = &key_buf[..];
            let key_match = current_key == key;
            _last_key = Some(String::from_utf8_lossy(current_key).to_string());

            // If we found our key, process it immediately
            if key_match {
                // Verify the value offset is within the data section
                if value_offset >= data_end {
                    return Err(io::Error::new(
                        io::ErrorKind::InvalidData,
                        format!(
                            "Invalid value offset in SSTable: {} >= {}",
                            value_offset, data_end
                        ),
                    )
                    .into());
                }

                // Save current position to restore later
                let _current_pos = file.stream_position()?;

                // Seek to the value position
                file.seek(SeekFrom::Start(value_offset))?;

                // Read key length (8 bytes)
                let mut stored_key_len_buf = [0u8; 8];
                file.read_exact(&mut stored_key_len_buf)?;
                let stored_key_len = u64::from_le_bytes(stored_key_len_buf) as usize;

                // Read key
                let mut stored_key_buf = vec![0u8; stored_key_len];
                file.read_exact(&mut stored_key_buf)?;

                // Read value length (8 bytes)
                let mut value_len_buf = [0u8; 8];
                file.read_exact(&mut value_len_buf)?;
                let value_len = u64::from_le_bytes(value_len_buf);

                if value_len == u64::MAX {
                    return Ok(None);
                }

                // Read the value in chunks to avoid large allocations
                let mut value = Vec::with_capacity(value_len as usize);
                let mut remaining = value_len;
                let mut buf = [0u8; 8192]; // 8KB buffer

                while remaining > 0 {
                    let to_read = std::cmp::min(remaining, buf.len() as u64) as usize;
                    let buf = &mut buf[..to_read];
                    file.read_exact(buf)?;
                    value.extend_from_slice(buf);
                    remaining -= to_read as u64;
                }

                return Ok(Some(value));
            }
        }

        Ok(None)
    }
}
