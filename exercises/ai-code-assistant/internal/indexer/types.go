package indexer

import (
	"context"
	"time"
)

// Indexer defines the interface for indexing code and retrieving relevant code chunks.
type Indexer interface {
	// IndexPath indexes all files in the given path (file or directory)
	// and stores them in the configured storage.
	IndexPath(ctx context.Context, path string) error

	// IndexFile indexes a single file and stores the generated chunks.
	IndexFile(ctx context.Context, filePath string) error

	// GetSupportedLanguages returns the list of supported programming languages
	GetSupportedLanguages() []string
}

// DocumentInfo holds metadata about a document being indexed
type DocumentInfo struct {
	Path     string
	Content  []byte
	Language string
}

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

// LanguageDetector detects the programming language of a file
type LanguageDetector interface {
	// Detect detects the programming language of a file
	Detect(path string, content []byte) (string, error)
	// GetSupportedLanguages returns the list of supported programming languages
	GetSupportedLanguages() []string
}
