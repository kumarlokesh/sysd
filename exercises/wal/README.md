# Write-Ahead Log (WAL) Implementation

A high-performance, durable Write-Ahead Log implementation in Go, designed for building reliable storage systems. This implementation provides atomicity and durability guarantees, making it suitable for building persistent storage engines, databases, and other systems requiring reliable write operations.

## Features

- **Durability**: Ensures data durability through write-ahead logging
- **High Performance**: Uses buffered I/O and batch writes for throughput
- **Crash Recovery**: Recovers unflushed data after crashes
- **Segment-based**: Automatically rotates log segments to manage file sizes
- **Concurrent Access**: Safe for concurrent reads and writes
- **Transactions**: Support for atomic multi-record operations
- **Non-blocking**: Background flushing for improved throughput
- **Configurable**: Tunable parameters for different workloads

## Architecture

### Core Components

1. **LogWriter**: Handles writing records to the WAL
   - Buffers writes for better performance
   - Manages segment rotation
   - Handles file I/O operations

2. **LogReader**: Reads records from the WAL
   - Supports sequential scanning of records
   - Handles segment transitions

3. **WAL Manager**: High-level interface for WAL operations
   - Coordinates between readers and writers
   - Manages WAL lifecycle

### On-Disk Format

Each WAL segment file contains a sequence of records:

```
[Record 1][Record 2]...[Record N]
```

Record format:

```
+----------------+----------------+----------------+----------------+----------------+----------------+
|    LSN (8B)    |   TxID (8B)    |   Type (1B)    |  KeyLen (4B)   |  ValueLen (4B) |                
+----------------+----------------+----------------+----------------+----------------+
|                                                                 |                
|                            Key (KeyLen B)                       |                
|                                                                 |                
+----------------+----------------+----------------+----------------+
|                                                                 |                
|                           Value (ValueLen B)                    |                
|                                                                 |                
+----------------+----------------+----------------+----------------+
|                      Checksum (8B)                              |
+----------------+----------------+
```

## Getting Started

### Installation

```bash
go get github.com/kumarlokesh/sysd/exercises/wal
```

### CLI Tool

The project includes a command-line interface (CLI) tool called `wald` that demonstrates the WAL's functionality:

```bash
# Build the CLI
go build -o wald ./cmd/wald

# Show help
./wald -h

# List available commands
./wald help
```

#### Available Commands

- `write`: Write a key-value pair to the WAL (non-transactional)
- `read`: Read and display all records from the WAL
- `begin-tx`: Start a new transaction
- `tx-write`: Write a key-value pair within a transaction
- `commit`: Commit the current transaction
- `abort`: Abort the current transaction

#### Test Scripts

The repository includes test scripts that demonstrate common usage patterns:

1. **test_tx_flow.sh**: Demonstrates a complete transaction commit flow

   ```bash
   ./test_tx_flow.sh
   ```

   This script performs the following steps:
   1. Begins a new transaction
   2. Writes a record within the transaction
   3. Commits the transaction
   4. Verifies the record was persisted

2. **test_abort_flow.sh**: Demonstrates a transaction abort flow

   ```bash
   ./test_abort_flow.sh
   ```

   This script performs the following steps:
   1. Begins a new transaction
   2. Writes a record within the transaction
   3. Aborts the transaction
   4. Verifies the record was not persisted

These scripts are useful for:

- Understanding the transaction API usage
- Verifying the WAL's behavior with different operations
- Testing recovery scenarios
- Demonstrating the WAL's durability guarantees

### Library Usage

```go
package main

import (
 "log"
 "github.com/kumarlokesh/sysd/exercises/wal/internal/wal"
)

### Basic Usage

```go
package main

import (
 "log"
 "github.com/kumarlokesh/sysd/exercises/wal/internal/wal"
)

func main() {
 // Initialize WAL with default configuration
 config := &wal.Config{
  Dir:         "./data/wal",
  Sync:        true,
  SegmentSize: 64 * 1024 * 1024, // 64MB segments
 }

 // Open or create a new WAL
 wal, err := wal.Open(config)
 if err != nil {
  log.Fatalf("Failed to open WAL: %v", err)
 }
 defer func() {
  if err := wal.Close(); err != nil {
   log.Printf("Error closing WAL: %v", err)
  }
 }()

 // Write a non-transactional record (txID = 0)
 lsn, err := wal.Write(0, []byte("key1"), []byte("value1"))
 if err != nil {
  log.Fatalf("Write failed: %v", err)
 }
 log.Printf("Wrote record with LSN: %d", lsn)

 // Read all records
 records, err := wal.ReadAll()
 if err != nil {
  log.Fatalf("ReadAll failed: %v", err)
 }

 for _, rec := range records {
  log.Printf("Record: LSN=%d, TxID=%d, Key=%s, Value=%s", 
   rec.LSN, rec.TxID, rec.Key, rec.Value)
 }
}
```

### Transactional Usage

```go
// Start a new transaction
txID := wal.Begin()

defer func() {
    if r := recover(); r != nil {
        // Handle panic and rollback
        if err := wal.Abort(txID); err != nil {
            log.Printf("Error aborting transaction: %v", err)
        }
        panic(r) // re-throw panic after cleanup
    }
}()

// Perform transactional writes
_, err1 := wal.Write(txID, []byte("key1"), []byte("value1"))
_, err2 := wal.Write(txID, []byte("key2"), []byte("value2"))

if err1 != nil || err2 != nil {
    if err := wal.Abort(txID); err != nil {
        log.Printf("Error aborting transaction: %v", err)
    }
    return fmt.Errorf("transaction failed: %v, %v", err1, err2)
}

// Commit the transaction
if err := wal.Commit(txID); err != nil {
    return fmt.Errorf("commit failed: %v", err)
}
```

## Configuration Options

The `Config` struct provides several options to tune WAL behavior:

```go
type Config struct {
    // Directory to store WAL segments
    Dir         string
    
    // Whether to sync writes to disk (default: false)
    Sync        bool
    
    // Maximum size of each segment file in bytes (default: 1GB)
    SegmentSize int64
    
    // Buffer size for writes (default: 64KB)
    BufferSize  int
    
    // Flush interval for background flusher (default: 1s)
    FlushInterval time.Duration
}
```

## Performance Tuning

### Write Performance

- **Batch Writes**: Group multiple writes into transactions
- **Sync Policy**: Set `Sync: false` for better throughput (but less durability)
- **Buffer Size**: Increase `BufferSize` for write-heavy workloads
- **Segment Size**: Larger segments reduce file rotation overhead

### Read Performance

- Use `ReadAll` for recovery scenarios only
- For production use, implement a cursor-based iterator
- Consider adding caching for frequently accessed records

## Recovery Process

The WAL implements a robust recovery mechanism:

1. **Startup Recovery**:
   - Scans all segments in order
   - Rebuilds transaction state
   - Identifies the last consistent point

2. **Crash Recovery**:
   - Detects partially written records
   - Recovers completed transactions
   - Discards incomplete transactions

3. **Consistency Guarantees**:
   - Atomic transactions (all or nothing)
   - Durable writes (when Sync is enabled)
   - Ordered record sequence

## Error Handling

All WAL methods return errors that should be properly handled:

```go
wal, err := wal.Open(config)
if errors.Is(err, os.ErrNotExist) {
    // Handle directory not found
} else if errors.Is(err, os.ErrPermission) {
    // Handle permission issues
} else if err != nil {
    // Handle other errors
}
```

## Best Practices

1. **Resource Management**:
   - Always call `Close()` when done
   - Use `defer` for cleanup
   - Handle panics in transaction code

2. **Durability**:
   - Use `Sync: true` for critical data
   - Implement proper shutdown procedures
   - Monitor disk space usage

3. **Monitoring**:
   - Track WAL size and growth
   - Monitor flush latencies
   - Alert on error conditions

## Building Blocks

The WAL is built using these key components:

1. **LogWriter**: Handles writing records to disk
   - Manages segment rotation
   - Implements buffered I/O
   - Handles file system operations

2. **LogReader**: Reads records from disk
   - Supports sequential scanning
   - Handles segment transitions
   - Validates record integrity

3. **WAL Manager**: Coordinates operations
   - Manages transactions
   - Handles recovery
   - Provides public API

## Command Line Interface (CLI)

The `wald` CLI tool provides command-line access to WAL operations. Here are the available commands:

### Basic Commands

```bash
# Write a record (non-transactional)
wald -dir ./data/wal write key1 value1

# Read all records
wald -dir ./data/wal read

# Show WAL information
wald -dir ./data/wal info
```

### Transaction Commands

```bash
# Start a transaction and write records
wald -dir ./data/wal -tx 123 write key1 value1

# Commit a transaction
wald -dir ./data/wal commit 123
```

### Maintenance Commands

```bash
# Compact WAL (remove old segments)
wald -dir ./data/wal compact
```

Note: The `-dir` flag specifies the directory where WAL files are stored. If not provided, it defaults to `./data/wal`.

## License

MIT
