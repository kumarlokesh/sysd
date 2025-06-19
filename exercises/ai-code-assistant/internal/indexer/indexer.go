package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/storage"
	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/types"
)

// DefaultIndexer is the default implementation of the Indexer interface
type DefaultIndexer struct {
	storage          storage.Storage
	languageDetector LanguageDetector
	parser           *Parser
	chunker          *Chunker

	// File extensions to include (defaults to common code file extensions)
	includeExts map[string]bool

	// Directories to ignore (e.g., .git, node_modules)
	ignoreDirs map[string]bool

	// Maximum file size to process (in bytes)
	maxFileSize int64

	// Number of workers for parallel processing
	workerCount int

	// Logger for the indexer
	logger *slog.Logger
}

// IndexerOption defines a function that configures an Indexer
type IndexerOption func(*DefaultIndexer)

// WithLanguageDetector sets the language detector for the indexer
func WithLanguageDetector(detector LanguageDetector) IndexerOption {
	return func(i *DefaultIndexer) {
		i.languageDetector = detector
	}
}

// WithParser sets the parser for the indexer
func WithParser(parser *Parser) IndexerOption {
	return func(i *DefaultIndexer) {
		i.parser = parser
	}
}

// WithChunker sets the chunker for the indexer
func WithChunker(chunker *Chunker) IndexerOption {
	return func(i *DefaultIndexer) {
		i.chunker = chunker
	}
}

// WithFileExtensions sets the file extensions to include
func WithFileExtensions(exts ...string) IndexerOption {
	return func(i *DefaultIndexer) {
		i.includeExts = make(map[string]bool)
		for _, ext := range exts {
			i.includeExts[ext] = true
		}
	}
}

// WithIgnoredDirs sets the directories to ignore
func WithIgnoredDirs(dirs ...string) IndexerOption {
	return func(i *DefaultIndexer) {
		i.ignoreDirs = make(map[string]bool)
		for _, dir := range dirs {
			i.ignoreDirs[dir] = true
		}
	}
}

// WithMaxFileSize sets the maximum file size to process
func WithMaxFileSize(size int64) IndexerOption {
	return func(i *DefaultIndexer) {
		i.maxFileSize = size
	}
}

// WithWorkerCount sets the number of workers for parallel processing
func WithWorkerCount(count int) IndexerOption {
	return func(i *DefaultIndexer) {
		if count > 0 {
			i.workerCount = count
		}
	}
}

// WithLogger sets the logger for the indexer
func WithLogger(logger *slog.Logger) IndexerOption {
	return func(i *DefaultIndexer) {
		i.logger = logger
	}
}

// NewDefaultIndexer creates a new DefaultIndexer with the given storage and options
func NewDefaultIndexer(storage storage.Storage, opts ...IndexerOption) *DefaultIndexer {
	logger := slog.Default()
	logger.Debug("Initializing new DefaultIndexer")

	// Initialize with default file extensions
	includeExts := make(map[string]bool)
	for _, ft := range DefaultFileTypes {
		for _, ext := range ft.Extensions {
			includeExts[ext] = true
		}
	}

	// Log the included extensions
	var extensions []string
	for ext := range includeExts {
		extensions = append(extensions, ext)
	}
	logger.Debug("Default file extensions", "extensions", extensions)

	indexer := &DefaultIndexer{
		storage:          storage,
		includeExts:      includeExts,
		ignoreDirs:       make(map[string]bool),
		maxFileSize:      10 * 1024 * 1024, // 10MB
		workerCount:      4,
		languageDetector: NewDefaultLanguageDetector(),
		parser:           NewParser(),
		chunker:          NewChunker(),
		logger:           logger,
	}

	// Apply options
	logger.Debug("Applying options", "count", len(opts))
	for i, opt := range opts {
		logger.Debug("Applying option", "index", i)
		opt(indexer)
	}

	logger.Debug("DefaultIndexer initialization complete",
		"workerCount", indexer.workerCount,
		"maxFileSize", indexer.maxFileSize)

	return indexer
}

// WithLanguageDetector sets a custom language detector
func (i *DefaultIndexer) WithLanguageDetector(detector LanguageDetector) *DefaultIndexer {
	i.languageDetector = detector
	return i
}

// WithParser sets a custom parser
func (i *DefaultIndexer) WithParser(parser *Parser) *DefaultIndexer {
	i.parser = parser
	return i
}

// WithChunker sets a custom chunker
func (i *DefaultIndexer) WithChunker(chunker *Chunker) *DefaultIndexer {
	i.chunker = chunker
	return i
}

// WithIncludeExts sets the file extensions to include
func (i *DefaultIndexer) WithIncludeExts(exts ...string) *DefaultIndexer {
	i.includeExts = make(map[string]bool)
	for _, ext := range exts {
		i.includeExts[ext] = true
	}
	return i
}

// WithIgnoreDirs sets the directories to ignore
func (i *DefaultIndexer) WithIgnoreDirs(dirs ...string) *DefaultIndexer {
	i.ignoreDirs = make(map[string]bool)
	for _, dir := range dirs {
		i.ignoreDirs[dir] = true
	}
	return i
}

// WithMaxFileSize sets the maximum file size to process (in bytes)
func (i *DefaultIndexer) WithMaxFileSize(size int64) *DefaultIndexer {
	i.maxFileSize = size
	return i
}

// WithWorkerCount sets the number of workers for parallel processing
func (i *DefaultIndexer) WithWorkerCount(count int) *DefaultIndexer {
	if count > 0 {
		i.workerCount = count
	}
	return i
}

// IndexPath implements the Indexer interface
func (i *DefaultIndexer) IndexPath(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	if info.IsDir() {
		return i.indexDirectory(ctx, path)
	}
	return i.IndexFile(ctx, path)
}

// IndexFile implements the Indexer interface
func (i *DefaultIndexer) IndexFile(ctx context.Context, path string) error {
	i.logger.Debug("Indexing file", "path", path)

	// Check if context is done
	select {
	case <-ctx.Done():
		i.logger.Warn("Context canceled before indexing file", "path", path, "error", ctx.Err())
		return ctx.Err()
	default:
		// Continue with indexing
	}

	chunks, err := i.indexFile(path)
	if err != nil {
		i.logger.Error("Failed to index file", "path", path, "error", err)
		return fmt.Errorf("failed to index file %s: %w", path, err)
	}

	if len(chunks) == 0 {
		i.logger.Info("No chunks generated from file", "path", path)
		return nil
	}

	i.logger.Debug("Storing chunks in vector store", "path", path, "chunk_count", len(chunks))

	// Store chunks in the vector store
	if err := i.storage.StoreChunks(ctx, chunks); err != nil {
		i.logger.Error("Failed to store chunks", "path", path, "error", err)
		return fmt.Errorf("failed to store chunks for file %s: %w", path, err)
	}

	i.logger.Info("Successfully indexed file",
		"path", path,
		"chunks", len(chunks))
	return nil
}

// GetSupportedLanguages returns the list of supported programming languages
func (i *DefaultIndexer) GetSupportedLanguages() []string {
	if i.languageDetector != nil {
		return i.languageDetector.GetSupportedLanguages()
	}
	return []string{}
}

// indexDirectory recursively indexes all files in a directory
func (i *DefaultIndexer) indexDirectory(ctx context.Context, dirPath string) error {
	i.logger.Info("Indexing directory", "path", dirPath)

	// Create a channel to collect errors from goroutines
	errCh := make(chan error, 1)
	// Buffer channel for files to process
	fileCh := make(chan string, 100)
	// Channel to track worker completion
	doneCh := make(chan struct{})
	var wg sync.WaitGroup

	// Start worker goroutines
	for w := 0; w < i.workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			i.logger.Debug("Worker started", "worker_id", workerID)

			for filePath := range fileCh {
				select {
				case <-ctx.Done():
					i.logger.Debug("Worker context canceled", "worker_id", workerID)
					errCh <- ctx.Err()
					return
				default:
					i.logger.Debug("Worker processing file",
						"worker_id", workerID,
						"file", filePath)

					if err := i.IndexFile(ctx, filePath); err != nil {
						i.logger.Error("Failed to index file",
							"worker_id", workerID,
							"file", filePath,
							"error", err)
						// Continue with next file on error
					}
				}
			}

			i.logger.Debug("Worker finished", "worker_id", workerID)
		}(w)
	}

	// Close the done channel when all workers are done
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	// Walk the directory and send files to workers
	go func() {
		defer close(fileCh)
		i.logger.Debug("Starting directory walk", "path", dirPath)

		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				i.logger.Debug("Directory walk canceled", "path", path)
				return ctx.Err()
			default:
				// Continue with walking
			}

			if err != nil {
				i.logger.Warn("Error accessing path",
					"path", path,
					"error", err)
				return nil // Continue walking on error
			}

			if d.IsDir() {
				dirName := filepath.Base(path)
				if i.ignoreDirs[dirName] {
					i.logger.Debug("Skipping ignored directory", "path", path)
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if !i.includeExts[ext] {
				i.logger.Debug("Skipping file with unhandled extension",
					"path", path,
					"extension", ext)
				return nil
			}

			info, err := d.Info()
			if err != nil {
				i.logger.Warn("Failed to get file info",
					"path", path,
					"error", err)
				return nil
			}

			if info.Size() > i.maxFileSize {
				i.logger.Info("Skipping large file",
					"file", path,
					"size", info.Size())
				return nil
			}

			select {
			case fileCh <- path:
				i.logger.Debug("Queued file for processing", "path", path)
			case <-ctx.Done():
				i.logger.Debug("Context canceled while queuing file", "path", path)
				return ctx.Err()
			}

			return nil
		})

		if err != nil {
			err = fmt.Errorf("error walking directory %s: %w", dirPath, err)
			i.logger.Error("Directory walk failed", "error", err)
			errCh <- err
		}
	}()

	// Wait for either all workers to finish or an error to occur
	select {
	case err := <-errCh:
		// If there was an error, cancel the context to signal workers to stop
		i.logger.Error("Error during directory indexing", "error", err)
		return err
	case <-doneCh:
		i.logger.Info("Finished indexing directory", "path", dirPath)
		return nil
	case <-ctx.Done():
		i.logger.Warn("Directory indexing canceled", "path", dirPath, "error", ctx.Err())
		return ctx.Err()
	}
}

// generateDocumentID generates a unique ID for a document based on its path
func generateDocumentID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:])
}

// indexFile indexes a single file
func (i *DefaultIndexer) indexFile(filePath string) ([]types.Chunk, error) {
	i.logger.Debug("Starting to index file", "file", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		i.logger.Error("Failed to read file", "file", filePath, "error", err)
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	i.logger.Debug("Successfully read file", "file", filePath, "size_bytes", len(content))

	// Get file info for metadata
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		i.logger.Error("Failed to get file info", "file", filePath, "error", err)
		return nil, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	// Detect language
	i.logger.Debug("Detecting language", "file", filePath)
	language, err := i.languageDetector.Detect(filePath, content)
	if err != nil {
		i.logger.Warn("Failed to detect language, defaulting to empty",
			"file", filePath,
			"error", err)
		language = ""
	}
	i.logger.Info("Detected language",
		"file", filePath,
		"language", language)

	// Create document ID (hash of file path for now)
	docID := generateDocumentID(filePath)
	i.logger.Debug("Generated document ID",
		"file", filePath,
		"doc_id", docID)

	// Parse and chunk the file
	i.logger.Debug("Starting to parse and chunk file",
		"file", filePath)

	var chunks []types.Chunk

	// If no language detected or parser not available, use simple chunking
	if language == "" || i.parser == nil {
		i.logger.Debug("No language detected or parser not available, using simple chunking", "file", filePath)
		var chunkErr error
		chunks, chunkErr = i.chunker.ChunkFile(filePath, content, "", nil)
		if chunkErr != nil {
			i.logger.Error("Failed to chunk file with simple chunker",
				"file", filePath,
				"error", chunkErr)
			return nil, fmt.Errorf("failed to chunk file with simple chunker: %w", chunkErr)
		}
	} else {
		i.logger.Debug("Parsing file with language",
			"file", filePath,
			"language", language)

		// Parse the file to get the syntax tree
		tree, err := i.parser.Parse(content, language)
		if err != nil {
			i.logger.Warn("Failed to parse file, falling back to whole file chunking",
				"file", filePath,
				"language", language,
				"error", err)
			chunks, err = i.chunker.ChunkFile(filePath, content, language, nil)
			if err != nil {
				var chunkErr error
				chunks, chunkErr = i.chunker.ChunkFile(filePath, content, language, nil)
				if chunkErr != nil {
					chunks, chunkErr = i.chunker.ChunkFile(filePath, content, language, tree)
					if chunkErr != nil {
						i.logger.Error("Failed to chunk file",
							"file", filePath,
							"error", err)
						return nil, fmt.Errorf("all chunking attempts failed: %w, %w", err, chunkErr)
					}
				}
			}
		} else if tree == nil {
			i.logger.Warn("Parser returned nil tree, falling back to whole file chunking",
				"file", filePath,
				"language", language)
			chunks, err = i.chunker.ChunkFile(filePath, content, language, nil)
			if err != nil {
				var chunkErr error
				chunks, chunkErr = i.chunker.ChunkFile(filePath, content, language, nil)
				if chunkErr != nil {
					chunks, chunkErr = i.chunker.ChunkFile(filePath, content, language, tree)
					if chunkErr != nil {
						i.logger.Error("Failed to chunk file",
							"file", filePath,
							"error", err)
						return nil, fmt.Errorf("all chunking attempts failed: %w, %w", err, chunkErr)
					}
				}
			}
		} else {
			i.logger.Debug("Successfully parsed file, using AST-based chunking",
				"file", filePath)
			var chunkErr error
			chunks, chunkErr = i.chunker.ChunkFile(filePath, content, language, tree)
			if chunkErr != nil {
				i.logger.Error("Failed to chunk file with AST",
					"file", filePath,
					"error", chunkErr)
				return nil, fmt.Errorf("failed to chunk file with AST: %w", chunkErr)
			}
		}
	}

	i.logger.Info("Successfully chunked file",
		"file", filePath,
		"chunk_count", len(chunks))

	// Enrich chunks with metadata
	enrichedChunks := make([]types.Chunk, 0, len(chunks))
	for idx, chunk := range chunks {
		// Copy the chunk to avoid modifying the original
		enrichedChunk := chunk

		// Set document ID and other metadata
		enrichedChunk.DocumentID = docID
		if enrichedChunk.Metadata == nil {
			enrichedChunk.Metadata = make(map[string]string)
		}
		enrichedChunk.Metadata["file_path"] = filePath
		enrichedChunk.Metadata["file_name"] = fileInfo.Name()
		enrichedChunk.Metadata["file_size"] = fmt.Sprintf("%d", fileInfo.Size())
		enrichedChunk.Metadata["file_mode"] = fileInfo.Mode().String()
		enrichedChunk.Metadata["file_mod_time"] = fileInfo.ModTime().Format(time.RFC3339)
		enrichedChunk.Metadata["language"] = language
		enrichedChunk.ChunkIndex = idx
		enrichedChunk.TotalChunks = len(chunks)
		enrichedChunk.CreatedAt = time.Now()

		enrichedChunks = append(enrichedChunks, enrichedChunk)

		if idx < 3 { // Log first few chunks for debugging
			i.logger.Debug("Sample chunk created",
				"file", filePath,
				"chunk_index", idx,
				"content_preview", truncateString(chunk.Content, 100)+"...",
				"node_type", chunk.NodeType)
		} else if idx == 3 {
			i.logger.Debug("... (additional chunks not shown)", "file", filePath)
		}
	}

	i.logger.Info("Finished enriching chunks with metadata",
		"file", filePath,
		"enriched_chunk_count", len(enrichedChunks))

	return enrichedChunks, nil
}

// truncateString shortens a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
