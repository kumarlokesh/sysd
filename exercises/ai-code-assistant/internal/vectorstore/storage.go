package vectorstore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/storage"
	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/types"
)

// ChromaStore implements the storage.Storage interface using ChromaDB
type ChromaStore struct {
	client         *ChromaClient
	collectionName string
	logger         *slog.Logger
}

// NewChromaStore creates a new ChromaStore that implements storage.Storage
func NewChromaStore(client *ChromaClient, collectionName string, logger *slog.Logger) storage.Storage {
	return &ChromaStore{
		client:         client,
		collectionName: collectionName,
		logger:         logger,
	}
}

// StoreChunks implements Storage.StoreChunks
func (s *ChromaStore) StoreChunks(ctx context.Context, chunks []types.Chunk) error {
	s.logger.Info("Starting to store chunks", "count", len(chunks))

	if len(chunks) == 0 {
		s.logger.Warn("No chunks to store")
		return nil
	}

	s.logger.Info("Creating or getting collection",
		"collection", s.collectionName,
		"client", fmt.Sprintf("%+v", s.client))

	collection, err := s.client.CreateCollection(ctx, s.collectionName)
	if err != nil {
		s.logger.Error("Failed to create or get collection",
			"collection", s.collectionName,
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return fmt.Errorf("failed to create collection: %w", err)
	}

	if collection == nil {
		err := fmt.Errorf("created collection is nil")
		s.logger.Error("Created collection is nil")
		return err
	}

	s.logger.Info("Successfully created or retrieved collection",
		"collection", s.collectionName,
		"collection_id", collection.ID,
		"collection_name", collection.Name)

	// Prepare data for ChromaDB
	s.logger.Info("Preparing chunks for storage", "count", len(chunks))

	ids := make([]string, 0, len(chunks))
	documents := make([]string, 0, len(chunks))
	metadatas := make([]map[string]interface{}, 0, len(chunks))

	for i, chunk := range chunks {
		// Convert chunk metadata to interface{} for ChromaDB
		metadata := make(map[string]interface{})
		for k, v := range chunk.Metadata {
			metadata[k] = v
		}

		// Add chunk-specific metadata
		metadata["document_id"] = chunk.DocumentID
		metadata["file_path"] = chunk.FilePath
		metadata["language"] = chunk.Language
		metadata["node_type"] = chunk.NodeType
		metadata["start_line"] = chunk.StartLine
		metadata["end_line"] = chunk.EndLine
		metadata["chunk_index"] = chunk.ChunkIndex
		metadata["total_chunks"] = chunk.TotalChunks
		metadata["created_at"] = chunk.CreatedAt.Format(time.RFC3339)

		ids = append(ids, chunk.ID)
		documents = append(documents, chunk.Content)
		metadatas = append(metadatas, metadata)

		s.logger.Debug("Prepared chunk",
			"index", i,
			"id", chunk.ID,
			"file", chunk.FilePath,
			"start_line", chunk.StartLine,
			"end_line", chunk.EndLine,
			"content_preview", truncateString(chunk.Content, 100)+"...")
	}

	s.logger.Info("Adding documents to collection",
		"count", len(ids),
		"collection", s.collectionName,
		"sample_id", safeGet(ids, 0, ""),
		"sample_doc", safeGetString(documents, 0, ""),
		"sample_meta", safeGetMap(metadatas, 0, nil))

	startTime := time.Now()

	// Convert metadatas to the format expected by ChromaDB
	chromaMetadatas := make([]map[string]interface{}, len(metadatas))
	copy(chromaMetadatas, metadatas)

	s.logger.Debug("Sample document data being sent to ChromaDB",
		"first_id", safeGet(ids, 0, ""),
		"first_doc_preview", safeGetString(documents, 0, ""),
		"first_meta", safeGetMap(chromaMetadatas, 0, nil),
		"total_docs", len(documents))

	s.logger.Debug("Calling collection.Add() with documents", "count", len(documents))
	_, err = collection.Add(
		ctx,
		nil, // embeddings (nil means Chroma will compute them)
		chromaMetadatas,
		documents,
		ids,
	)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Failed to add documents to collection",
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"duration", duration,
			"collection", s.collectionName,
			"document_count", len(documents))
		return fmt.Errorf("failed to add documents to collection: %w", err)
	}

	s.logger.Debug("Successfully called collection.Add()",
		"duration", duration,
		"document_count", len(documents))

	s.logger.Info("Successfully added documents to collection",
		"count", len(ids),
		"collection", s.collectionName,
		"duration", duration)
	return nil
}

// Helper function to get minimum of two integers

// safeGet safely gets a value from a slice by index, returning a default value if out of bounds
func safeGet(slice []string, index int, defaultValue string) string {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return defaultValue
}

// safeGetString safely gets a string from a string slice by index, returning a default value if out of bounds
func safeGetString(slice []string, index int, defaultValue string) string {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return defaultValue
}

// safeGetMap safely gets a map from a slice by index, returning a default value if out of bounds
func safeGetMap(slice []map[string]interface{}, index int, defaultValue map[string]interface{}) map[string]interface{} {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return defaultValue
}

// Search implements storage.Storage.Search
func (s *ChromaStore) Search(ctx context.Context, query string, limit int) ([]storage.SearchResult, error) {
	results, err := s.client.Query(ctx, s.collectionName, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}

	var searchResults []storage.SearchResult

	for _, r := range results {
		// Parse metadata
		metadata, ok := r["metadata"].(map[string]interface{})
		if !ok {
			s.logger.Warn("Failed to parse metadata")
			continue
		}

		// Convert metadata back to chunk
		chunk := &types.Chunk{
			ID:       r["id"].(string),
			Content:  r["document"].(string),
			Metadata: make(map[string]string),
		}

		// Parse metadata fields
		if documentID, ok := metadata["document_id"].(string); ok {
			chunk.DocumentID = documentID
		}
		if filePath, ok := metadata["file_path"].(string); ok {
			chunk.FilePath = filePath
		}
		if language, ok := metadata["language"].(string); ok {
			chunk.Language = language
		}
		if nodeType, ok := metadata["node_type"].(string); ok {
			chunk.NodeType = nodeType
		}
		if startLine, ok := metadata["start_line"].(float64); ok {
			chunk.StartLine = int(startLine)
		}
		if endLine, ok := metadata["end_line"].(float64); ok {
			chunk.EndLine = int(endLine)
		}
		if chunkIndex, ok := metadata["chunk_index"].(float64); ok {
			chunk.ChunkIndex = int(chunkIndex)
		}
		if totalChunks, ok := metadata["total_chunks"].(float64); ok {
			chunk.TotalChunks = int(totalChunks)
		}

		// Get score (inverse of distance)
		score := 0.0
		if dist, ok := r["distance"].(float64); ok && dist > 0 {
			score = 1.0 / (1.0 + dist)
		}

		// Convert metadata to map[string]interface{} for SearchResult
		searchMetadata := make(map[string]interface{})
		for k, v := range metadata {
			switch val := v.(type) {
			case string, int, int64, float64, bool:
				searchMetadata[k] = val
			default:
				// Convert other types to string
				searchMetadata[k] = fmt.Sprint(val)
			}
		}

		searchResults = append(searchResults, storage.SearchResult{
			Chunk:    chunk,
			Score:    score,
			Metadata: searchMetadata,
		})
	}

	return searchResults, nil
}

// GetChunk implements Storage.GetChunk
func (s *ChromaStore) GetChunk(ctx context.Context, id string) (*types.Chunk, error) {
	// ChromaDB's Get operation would go here
	// For now, we'll use a search with the ID as the query
	results, err := s.Search(ctx, id, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0].Chunk, nil
}

// DeleteChunks implements Storage.DeleteChunks
func (s *ChromaStore) DeleteChunks(ctx context.Context, ids []string) error {
	// ChromaDB's Delete operation would go here
	// This is a simplified version
	_, err := s.client.client.GetCollection(ctx, s.collectionName, nil)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	// Note: The current ChromaDB Go client might not support Delete yet
	// This is a placeholder for the actual implementation
	s.logger.Warn("Delete operation not fully implemented",
		"collection", s.collectionName,
		"ids", ids)
	return nil
}
