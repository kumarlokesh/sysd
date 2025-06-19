package types

import (
	"time"
)

// Chunk represents a chunk of code with its metadata
type Chunk struct {
	ID          string            `json:"id"`
	DocumentID  string            `json:"document_id"`
	Content     string            `json:"content"`
	FilePath    string            `json:"file_path"`
	Language    string            `json:"language"`
	StartLine   int               `json:"start_line"`
	EndLine     int               `json:"end_line"`
	NodeType    string            `json:"node_type"` // e.g., "function_definition", "class_definition"
	Metadata    map[string]string `json:"metadata"`
	Embedding   []float32         `json:"embedding,omitempty"`
	ChunkIndex  int               `json:"chunk_index"`
	TotalChunks int               `json:"total_chunks"`
	CreatedAt   time.Time         `json:"created_at"`
}
