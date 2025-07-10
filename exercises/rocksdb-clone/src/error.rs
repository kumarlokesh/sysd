use bincode::error::{DecodeError, EncodeError};
use std::{fmt, io, result};
use thiserror::Error;

/// A type alias for `Result<T, Error>` where `Error` is this crate's error type.
pub type Result<T> = result::Result<T, Error>;

/// The error type for RocksDB Clone operations.
#[derive(Error, Debug)]
pub enum Error {
    /// I/O error occurred
    #[error("I/O error: {0}")]
    Io(#[from] io::Error),

    /// Serialization error
    #[error("Serialization error: {0}")]
    Serialization(String),

    /// Deserialization error
    #[error("Deserialization error: {0}")]
    Deserialization(String),

    /// Key not found in storage
    #[error("Key not found")]
    KeyNotFound,

    /// Database not found at the specified path
    #[error("Database not found at: {0}")]
    DatabaseNotFound(String),

    /// Invalid argument provided
    #[error("Invalid argument: {0}")]
    InvalidArgument(String),

    /// Operation not supported
    #[error("Operation not supported: {0}")]
    NotSupported(String),

    /// Custom error
    #[error("Error: {0}")]
    Custom(String),
}

impl Error {
    /// Creates a new custom error
    pub fn custom<T: Into<String>>(msg: T) -> Self {
        Error::Custom(msg.into())
    }

    /// Creates a new serialization error
    pub fn serialization<T: fmt::Display>(msg: T) -> Self {
        Error::Serialization(msg.to_string())
    }
}

impl From<EncodeError> for Error {
    fn from(err: EncodeError) -> Self {
        Error::Serialization(err.to_string())
    }
}

impl From<DecodeError> for Error {
    fn from(err: DecodeError) -> Self {
        Error::Deserialization(err.to_string())
    }
}

impl From<serde_json::Error> for Error {
    fn from(err: serde_json::Error) -> Self {
        Error::Serialization(err.to_string())
    }
}
