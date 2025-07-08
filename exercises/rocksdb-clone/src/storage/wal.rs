use crate::error::{Error, Result};
use bincode::{Decode, Encode};
use serde::{Deserialize, Serialize};
use std::{
    fs::{File, OpenOptions},
    io::{self, BufReader, BufWriter, Read, Write},
    path::Path,
};

/// Represents an operation in the WAL
#[derive(Debug, Serialize, Deserialize, Encode, Decode)]
pub enum WalOp {
    Put { key: Vec<u8>, value: Vec<u8> },
    Delete { key: Vec<u8> },
}

/// Write-Ahead Log for persistence
pub struct WriteAheadLog {
    /// The buffered writer for the WAL file
    pub writer: BufWriter<File>,
    path: std::path::PathBuf,
}

impl WriteAheadLog {
    /// Creates or opens a WAL file at the given path
    pub fn new(path: impl AsRef<Path>) -> Result<Self> {
        let file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&path)
            .map_err(Error::Io)?;

        Ok(Self {
            writer: BufWriter::new(file),
            path: path.as_ref().to_path_buf(),
        })
    }

    /// Appends an operation to the WAL
    pub fn append(&mut self, op: &WalOp) -> Result<()> {
        // Serialize the operation to a buffer using bincode
        let mut serialized = Vec::new();
        bincode::encode_into_std_write(op, &mut serialized, bincode::config::standard())?;
        let len = serialized.len() as u64;

        // Write length prefix
        self.writer.write_all(&len.to_le_bytes())?;
        // Write the serialized operation
        self.writer.write_all(&serialized)?;

        // Flush to ensure it's written to disk
        self.writer.flush()?;
        Ok(())
    }

    /// Replays all operations in the WAL to rebuild the MemTable
    pub fn replay<F>(path: impl AsRef<Path>, mut apply: F) -> Result<()>
    where
        F: FnMut(WalOp) -> Result<()>,
    {
        let file = match File::open(path) {
            Ok(file) => file,
            Err(e) if e.kind() == io::ErrorKind::NotFound => return Ok(()),
            Err(e) => return Err(Error::Io(e)),
        };

        let mut reader = BufReader::new(file);
        let mut len_buf = [0u8; 8];

        loop {
            // Read the length prefix
            match reader.read_exact(&mut len_buf) {
                Ok(_) => {
                    let len = u64::from_le_bytes(len_buf) as usize;
                    let mut op_buf = vec![0u8; len];
                    reader.read_exact(&mut op_buf)?;

                    // Deserialize the operation
                    let op = bincode::decode_from_slice::<WalOp, _>(
                        &op_buf,
                        bincode::config::standard(),
                    )?
                    .0;

                    // Apply the operation to the MemTable
                    apply(op)?;
                }
                Err(ref e) if e.kind() == io::ErrorKind::UnexpectedEof => break,
                Err(e) => return Err(Error::Io(e)),
            }
        }

        Ok(())
    }

    /// Clears the WAL (after successful MemTable flush)
    pub fn clear(&mut self) -> Result<()> {
        self.writer = BufWriter::new(File::create(&self.path).map_err(Error::Io)?);
        Ok(())
    }

    /// Flushes any buffered data to disk
    pub fn flush(&mut self) -> Result<()> {
        self.writer.flush()?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_wal_append_and_replay() -> Result<()> {
        let dir = tempdir()?;
        let path = dir.path().join("test.wal");

        // Write some operations
        {
            let mut wal = WriteAheadLog::new(&path)?;
            wal.append(&WalOp::Put {
                key: b"key1".to_vec(),
                value: b"value1".to_vec(),
            })?;
            wal.append(&WalOp::Put {
                key: b"key2".to_vec(),
                value: b"value2".to_vec(),
            })?;
            wal.append(&WalOp::Delete {
                key: b"key1".to_vec(),
            })?;
        }

        // Replay and verify
        let mut ops = Vec::new();
        WriteAheadLog::replay(&path, |op| {
            ops.push(op);
            Ok(())
        })?;

        assert_eq!(ops.len(), 3);

        if let WalOp::Put { key, value } = &ops[0] {
            assert_eq!(key, b"key1");
            assert_eq!(value, b"value1");
        } else {
            panic!("Expected Put operation");
        }

        if let WalOp::Put { key, value } = &ops[1] {
            assert_eq!(key, b"key2");
            assert_eq!(value, b"value2");
        } else {
            panic!("Expected Put operation");
        }

        if let WalOp::Delete { key } = &ops[2] {
            assert_eq!(key, b"key1");
        } else {
            panic!("Expected Delete operation");
        }

        Ok(())
    }
}
