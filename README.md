# System Design Exercises

A collection of system design exercises implemented in Go and Rust. Each exercise is a self-contained project demonstrating the design and implementation of various distributed systems concepts.

## Exercises

### Go Exercises

1. **[AI Code Assistant](exercises/ai-code-assistant/)** - A system design exercise for building an AI-powered coding assistant.
2. **[Kafka Transactional Messaging](exercises/kafka-transactional-messaging/)** - Implementation of reliable message processing using Kafka transactions.
3. **[Write-Ahead Log (WAL)](exercises/wal/)** - A low-level implementation of a write-ahead log for data durability.
4. **[S3 Clone](exercises/s3-clone/)** - A minimal implementation of an Amazon S3-compatible object storage service with support for buckets and objects.
5. **[Kubernetes Custom Controller](exercises/k8s-controller/)** - A custom Kubernetes controller that manages Task resources to execute commands within the cluster.
6. **[SQL Parser](exercises/sql-parser/)** - A simplified SQL parser implementation in Go, focusing on PostgreSQL's query processing internals.
7. **[Cassandra SSTable](exercises/cassandra-sstable/)** - An implementation of a simplified version of Cassandra's SSTable storage format.
8. **[HNSW Vector Search](exercises/hnsw-poc/)** - A high-performance implementation of the HNSW algorithm for approximate nearest neighbor search.

### Rust Exercises

1. **[RocksDB Clone](exercises/rocksdb-clone/)** - A step-by-step implementation of a key-value store inspired by RocksDB, covering:
   - LSM Tree Storage Engine
   - Write-Ahead Log with Crash Recovery
   - SSTable Implementation
   - Compaction Strategies
   - MVCC (Multi-Version Concurrency Control)
   - Column Families
   - Merge Operators

2. **[SIMD-POC](exercises/simd-poc/)** - A proof-of-concept demonstrating SIMD (Single Instruction, Multiple Data) optimizations in Rust, including:
   - Vectorized operations for performance-critical code paths
   - Cross-platform SIMD using Rust's portable_simd
   - Performance benchmarking and comparison with scalar implementations

## Getting Started

### For Go Exercises

1. Clone the repository
2. Navigate to an exercise directory
3. Run `go test ./...` to run tests
4. Check the exercise's README for specific instructions

### For Rust Exercises

1. Clone the repository
2. Navigate to an exercise directory
3. Run `cargo test` to run tests
4. Check the exercise's README for specific instructions
