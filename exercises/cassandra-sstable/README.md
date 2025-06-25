# Cassandra SSTable Implementation

This exercise implements a simplified version of Cassandra's trie-indexed SSTable storage format, based on [CEP-25](https://cwiki.apache.org/confluence/display/CASSANDRA/CEP-25%3A+Trie-indexed+SSTable+format).

## Project Structure

```
.
├── examples/
│   └── basic/           # Example usage of the SSTable package
├── internal/
│   ├── trie/            # Trie data structure implementation
│   └── sstable/          # SSTable implementation
├── test/                 # Integration and end-to-end tests
└── README.md             # This file
```

## Implementation Status

### ✅ Completed Features

- **Trie Implementation**
  - In-memory trie with prefix compression
  - Serialization/deserialization
  - Basic operations (insert, search, range scan)
  - Prefix-based search and traversal

- **SSTable Implementation**
  - On-disk format with block-based storage
  - Write path with trie-indexed keys
  - Read path with point lookups and range scans
  - Memory-mapped I/O for efficient reads

## Getting Started

### Prerequisites

- Go 1.16 or later

### Building

```bash
go build -o bin/sstable ./cmd/sstable
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...


# Run benchmarks
go test -bench=. ./...
```

### Example Usage

See the [basic example](examples/basic/main.go) for a complete example of how to use the SSTable package. The example demonstrates:

- Creating a new SSTable
- Writing key-value pairs
- Reading data using point lookups
- Iterating over a range of keys

To run the example:

```bash
# Build and run the example
cd examples/basic
go run main.go
```

This will create a new SSTable file (`data.sst`) in the current directory, write some test data to it, and then read it back.

## Next Steps

### Planned Features

- [ ] Bloom filters for faster non-existent key lookups
- [ ] Compression for smaller on-disk size
- [ ] Memory-mapped block cache
- [ ] Background compaction
- [ ] Concurrent read/write support

## References

- [CEP-25: Trie-indexed SSTable format](https://cwiki.apache.org/confluence/display/CASSANDRA/CEP-25%3A+Trie-indexed+SSTable+format)
- [Cassandra Documentation](https://cassandra.apache.org/doc/latest/)
- [LSM-Tree and SSTable Overview](https://www.igvita.com/2012/02/06/sstable-and-log-structured-storage-leveldb/)
