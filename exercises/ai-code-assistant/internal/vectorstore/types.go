package vectorstore

import (
	"context"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/types"
)

// Storage defines the interface for vector store operations
type Storage interface {
	// StoreChunks stores document chunks in the vector store
	StoreChunks(ctx context.Context, chunks []types.Chunk) error

	// Search performs a similarity search across stored chunks
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)

	// GetChunk retrieves a specific chunk by ID
	GetChunk(ctx context.Context, id string) (*types.Chunk, error)

	// DeleteChunks removes chunks by their IDs
	DeleteChunks(ctx context.Context, ids []string) error
}

// SearchResult represents a search result from the vector store
type SearchResult struct {
	Chunk    *types.Chunk
	Score    float64
	Metadata map[string]interface{}
}
