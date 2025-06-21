# System Design Exercises

A collection of system design exercises implemented in Go. Each exercise is a self-contained project demonstrating the design and implementation of various distributed systems concepts.

## Structure

- `exercises/` - Contains individual system design exercises
  - `exercise-name/` - Each exercise is in its own directory
    - `cmd/` - Main application packages
    - `internal/` - Private application code
    - `pkg/` - Public packages that can be imported by other exercises
    - `test/` - Integration and end-to-end tests
    - `go.mod` - Go module definition
    - `README.md` - Exercise documentation
- `pkg/` - Shared packages across exercises
- `docs/` - Additional documentation
- `.github/` - GitHub workflows and templates
- `Makefile` - Common build and test commands

## Exercises

1. **[AI Code Assistant](exercises/ai-code-assistant/)** - A system design exercise for building an AI-powered coding assistant.
2. **[Kafka Transactional Messaging](exercises/kafka-transactional-messaging/)** - Implementation of reliable message processing using Kafka transactions.
3. **[Write-Ahead Log (WAL)](exercises/wal/)** - A low-level implementation of a write-ahead log for data durability.
4. **[S3 Clone](exercises/s3-clone/)** - A minimal implementation of an Amazon S3-compatible object storage service with support for buckets and objects.
5. **[Kubernetes Custom Controller](exercises/k8s-controller/)** - A custom Kubernetes controller that manages Task resources to execute commands within the cluster, with support for both one-time and scheduled tasks.

## Getting Started

1. Clone the repository
2. Navigate to an exercise directory
3. Run `go test ./...` to run tests
4. Check the exercise's README for specific instructions
