# AI Code Assistant

A proof-of-concept AI-powered code assistant with context awareness, built with CodeLlama and ChromaDB.

## Features

### Implemented

- **Core Infrastructure**
  - [x] ChromaDB integration for vector storage
  - [x] Configuration management
  - [x] Basic CLI structure

- **Document Processing**
  - [x] File system traversal
  - [x] Language detection
  - [x] Basic code parsing with tree-sitter
  - [x] Document chunking
  - [x] Basic metadata handling
  - [x] Advanced code parsing for multiple languages
  - [x] Error handling and logging

- **Vector Store**
  - [x] ChromaDB client implementation
  - [x] Collection management
  - [x] Document storage and retrieval
  - [x] Basic vector search

### In Progress

- **Document Indexing**
  - [ ] Embedding generation
  - [ ] Incremental updates
  - [ ] Performance optimizations for large codebases

### In Development

- **LLM Integration**
  - [ ] Local LLM service (CodeLlama)
  - [ ] Basic context management

### Planned

- **LLM Integration**
  - [ ] Advanced context management
  - [ ] Code completion
  - [ ] Code explanation and documentation generation

- **Search & Navigation**
  - [ ] Semantic search across codebase
  - [ ] Context-aware code navigation

- **Developer Experience**
  - [ ] VS Code extension
  - [ ] Interactive REPL

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌───────────────────────┐
│  CLI Tool      │     │  API Service    │     │  ChromaDB            │
│  (Go)          │◄───►│  (Go)           │◄────►  (Vector Store)    │
└─────────────────┘     └────────┬─────────┘     └───────────────────────┘
                                 │
                                 ▼
                         ┌─────────────────┐     ┌───────────────────────┐
                         │  LLM Service    │     │  Local Model         │
                         │  (Go)           │◄────►  (CodeLlama)       │
                         └─────────────────┘     └───────────────────────┘
```

### Key Design Decisions

1. **Vector Storage**: Using ChromaDB instead of PostgreSQL/pgvector for:
   - Simpler setup and maintenance
   - Native support for vector operations
   - Better performance for semantic search use cases
   - Built-in support for document storage and retrieval

2. **Technology Stack**:
   - **Backend**: Go for high performance and type safety
   - **Vector Database**: ChromaDB (v0.4.24) for vector storage and search
   - **LLM**: CodeLlama for code understanding and generation
   - **Containerization**: Docker for easy setup and deployment

3. **Development Approach**:
   - Iterative development with a focus on core functionality first
   - Comprehensive testing at all levels (unit, integration, e2e)
   - Clear separation of concerns in the codebase

## Getting Started

### Prerequisites

- Go 1.18+
- Docker and Docker Compose
- ChromaDB (via Docker)

### Installation

1. Clone the repository

2. Install Go dependencies:

   ```bash
   go mod tidy
   ```

3. Start ChromaDB using Docker Compose:

   ```bash
   docker-compose -f docker-compose.test.yml up -d
   ```

4. Configure the application:

   ```bash
   cp configs/config.example.yaml config.yaml
   ```

   Update the configuration as needed in `config.yaml`

### Running the Application

1. Start the API service:
   ```bash
   go run cmd/api/main.go
   ```

2. Use the CLI tool:
   ```bash
   # Show help
   go run cmd/cli/main.go --help
   
   # Index a directory
   go run cmd/cli/main.go index /path/to/your/code
   
   # View configuration
   go run cmd/cli/main.go config
   ```

### Development

To run tests:

```bash
# Run unit tests
go test ./...

# Run integration tests (requires ChromaDB running)
go test -tags=integration ./...
```

To build the project:

```bash
# Build CLI tool
go build -o bin/ai-code-assistant ./cmd/cli

# Build API server
go build -o bin/api-server ./cmd/api
```

## Project Structure

```markdown
.
├── cmd/                 # Main applications
│   ├── api/            # API server
│   └── cli/            # Command line interface
├── internal/            # Private application code
│   ├── config/         # Configuration management
│   ├── indexer/        # Code indexing and chunking
│   ├── llm/            # LLM integration
│   ├── parser/         # Code parsing
│   ├── search/         # Vector search
│   └── vectorstore/    # ChromaDB client and models
├── configs/             # Configuration files
├── docker-compose.test.yml  # Docker Compose for ChromaDB
└── scripts/             # Build and utility scripts
```

## Development Notes

### Implementation Details

1. **ChromaDB Version**: Using ChromaDB v0.4.24 as it's the latest version compatible with the current `chroma-go` client.
2. **Code Organization**: Separated concerns into clear packages for better maintainability.
3. **Testing**: Includes both unit tests and integration tests with a real ChromaDB instance.

### Future Improvements

1. **Upgrade Path**: Consider contributing v2 API support to `chroma-go` for future ChromaDB versions.
2. **Performance**: Optimize chunking and embedding generation for large codebases.
3. **Extensibility**: Make the vector store backend pluggable to support other databases in the future.

## License

MIT
