package indexer

// FileInfo represents a file to be indexed
type FileInfo struct {
	Path     string
	Content  []byte
	Language string
}

// Chunk represents a chunk of code with its metadata
type Chunk struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	FilePath  string            `json:"file_path"`
	Language  string            `json:"language"`
	StartLine int               `json:"start_line"`
	EndLine   int               `json:"end_line"`
	NodeType  string            `json:"node_type"` // e.g., "function_definition", "class_definition"
	Metadata  map[string]string `json:"metadata"`
}

// Indexer defines the interface for code indexing
type Indexer interface {
	// IndexPath recursively indexes all code files in the given path
	IndexPath(path string) ([]Chunk, error)

	// IndexFile indexes a single file
	IndexFile(path string) ([]Chunk, error)

	// GetSupportedLanguages returns the list of supported programming languages
	GetSupportedLanguages() []string
}

// LanguageDetector detects the programming language of a file
type LanguageDetector interface {
	Detect(path string, content []byte) (string, error)
	GetSupportedLanguages() []string
}
