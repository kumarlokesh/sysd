# HNSW (Hierarchical Navigable Small World) in Go

This project implements the HNSW (Hierarchical Navigable Small World) algorithm for approximate nearest neighbor search in Go. HNSW is a state-of-the-art algorithm for efficient similarity search in high-dimensional spaces, widely used in recommendation systems, image search, and natural language processing applications.

## Theoretical Foundations

### 1. Core Concepts

#### 1.1 Approximate Nearest Neighbor (ANN) Search

- Traditional nearest neighbor search becomes computationally expensive in high-dimensional spaces ("curse of dimensionality").
- ANN trades off some accuracy for significant speed improvements.

#### 1.2 Small World Property

- A graph where most nodes can be reached from every other node in a small number of steps.
- Combines characteristics of regular and random graphs.
- Provides efficient routing between any two nodes.

#### 1.3 Skip List Inspiration

- HNSW borrows the hierarchical structure from skip lists.
- Multiple layers of graphs with decreasing density.
- Higher layers (lower density) enable fast navigation across the graph.

### 2. HNSW Algorithm

#### 2.1 Graph Structure

- Multiple layers (L0 to Lmax)
- Each layer is a graph where nodes represent data points
- Lower layers (higher numbers) have fewer nodes and edges
- Bottom layer (L0) contains all data points

#### 2.2 Construction (Insertion)

1. Randomly select maximum layer for new element
2. For each layer from top to bottom:
   - Find the nearest neighbors in current layer
   - Connect to M nearest neighbors
   - Move to next layer and repeat

#### 2.3 Search (K-NN Query)

1. Start from the top layer
2. Find the nearest neighbor in current layer
3. Move to the same node in the layer below
4. Repeat until reaching the bottom layer
5. Return K nearest neighbors

### 3. Key Parameters

- **M**: Maximum number of connections per layer (controls graph density)
- **ef_construction**: Size of the dynamic candidate list during construction
- **ef_search**: Size of the dynamic candidate list during search
- **M_max0**: Maximum number of connections for the zero layer

### 4. Advantages

- **Efficient**: O(log n) search complexity
- **Scalable**: Handles high-dimensional data well
- **Flexible**: Parameters can be tuned for different use cases
- **Memory efficient**: Only stores connections, not the full distance matrix

## Features

- **High Performance**: Optimized Go implementation of HNSW algorithm
- **Concurrent Safe**: Thread-safe for concurrent searches
- **Configurable**: Tuneable parameters for different use cases
- **Extensible**: Easy to integrate with different distance metrics
- **Comprehensive Tests**: High test coverage with various test cases

## Installation

```bash
go get github.com/kumarlokesh/hnsw-poc
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "math/rand"

    "github.com/kumarlokesh/hnsw-poc/internal/hnsw"
)

func main() {
    // Initialize HNSW with default parameters
    h := hnsw.New(10, hnsw.Config{
        M:              16,     // Number of connections per layer
        EfConstruction: 200,   // Size of dynamic candidate list during construction
        EfSearch:       400,   // Size of dynamic candidate list during search
    })

    // Insert some random vectors
    dim := 128
    for i := 0; i < 1000; i++ {
        vector := make([]float32, dim)
        for j := range vector {
            vector[j] = rand.Float32()
        }
        h.Insert(i, vector)
    }

    // Search for nearest neighbors
    query := make([]float32, dim)
    for i := range query {
        query[i] = rand.Float32()
    }

    k := 5
    results := h.Search(query, k)
    fmt.Printf("Nearest neighbors: %v\n", results)
}
```

## Project Structure

```
hnsw-poc/
├── cmd/
│   └── hnsw-cli/       # Command-line interface
├── internal/
│   └── hnsw/           # Core HNSW implementation
│       ├── connect.go  # Graph connection logic
│       ├── distance.go # Distance calculations
│       ├── insert.go   # Insertion logic
│       ├── layer.go    # Layer management
│       ├── search.go   # Search functionality
│       └── types.go    # Core data structures
├── go.mod
├── go.sum
└── README.md
```

## Configuration

HNSW can be configured with the following parameters:

- `M` (default: 16): Maximum number of connections per layer
- `M0` (default: 2*M): Maximum number of connections for the zero layer
- `EfConstruction` (default: 200): Size of dynamic candidate list during construction
- `EfSearch` (default: 400): Size of dynamic candidate list during search
- `RandomSeed` (default: 42): Seed for random number generation

## Benchmarks

```bash
cd internal/hnsw
go test -bench=.
```

Example benchmark results:

```text
BenchmarkHNSW_Insert-8          10000            123456 ns/op
BenchmarkHNSW_Search-8         100000            12345 ns/op
```

## License

MIT

## References

1. [Efficient and robust approximate nearest neighbor search using Hierarchical Navigable Small World graphs](https://arxiv.org/abs/1603.09320) - Original HNSW paper
2. [HNSW in Wikipedia](https://en.wikipedia.org/wiki/Hierarchical_navigable_small_world)
3. [Awesome Vector Search](https://github.com/currentsapi/awesome-vector-search) - Collection of vector search resources
4. [HNSW: The Ultimate Guide](https://www.pinecone.io/learn/hnsw/) - Practical guide to HNSW
