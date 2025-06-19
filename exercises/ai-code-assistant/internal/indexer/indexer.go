package indexer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// DefaultIndexer is the default implementation of the Indexer interface
type DefaultIndexer struct {
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
}

// NewDefaultIndexer creates a new DefaultIndexer with default settings
func NewDefaultIndexer() *DefaultIndexer {
	return &DefaultIndexer{
		languageDetector: NewDefaultLanguageDetector(),
		parser:           NewParser(),
		chunker:          NewChunker(),
		includeExts: map[string]bool{
			".go":   true,
			".py":   true,
			".js":   true,
			".ts":   true,
			".jsx":  true,
			".tsx":  true,
			".rs":   true,
			".rb":   true,
			".java": true,
			".c":    true,
			".h":    true,
			".cpp":  true,
			".hpp":  true,
		},
		ignoreDirs: map[string]bool{
			".git":         true,
			"node_modules": true,
			"vendor":       true,
			"__pycache__":  true,
			"target":       true,
		},
		maxFileSize: 10 * 1024 * 1024, // 10MB
		workerCount: 4,                // Default to 4 workers
	}
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
func (i *DefaultIndexer) IndexPath(path string) ([]Chunk, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	var chunks []Chunk

	if info.IsDir() {
		chunks, err = i.indexDirectory(path)
	} else {
		chunks, err = i.indexFile(path)
	}

	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// IndexFile implements the Indexer interface
func (i *DefaultIndexer) IndexFile(path string) ([]Chunk, error) {
	return i.indexFile(path)
}

// GetSupportedLanguages returns the list of supported programming languages
func (i *DefaultIndexer) GetSupportedLanguages() []string {
	if i.languageDetector != nil {
		return i.languageDetector.GetSupportedLanguages()
	}
	return []string{}
}

// indexDirectory recursively indexes all files in a directory
func (i *DefaultIndexer) indexDirectory(dirPath string) ([]Chunk, error) {
	var chunks []Chunk
	var mu sync.Mutex
	var wg sync.WaitGroup

	fileCh := make(chan string, 100)

	for w := 0; w < i.workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileCh {
				fileChunks, err := i.indexFile(filePath)
				if err != nil {
					log.Error().Err(err).Str("file", filePath).Msg("Failed to index file")
					continue
				}

				mu.Lock()
				chunks = append(chunks, fileChunks...)
				mu.Unlock()
			}
		}()
	}

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if i.ignoreDirs[filepath.Base(path)] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !i.includeExts[ext] {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > i.maxFileSize {
			log.Warn().
				Str("file", path).
				Int64("size", info.Size()).
				Msg("Skipping large file")
			return nil
		}

		fileCh <- path
		return nil
	})
	close(fileCh)

	wg.Wait()

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return chunks, nil
}

// indexFile indexes a single file
func (i *DefaultIndexer) indexFile(filePath string) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	language, err := i.languageDetector.Detect(filePath, content)
	if err != nil {
		return nil, fmt.Errorf("failed to detect language for %s: %w", filePath, err)
	}

	if language == "" {
		return i.chunker.ChunkFile(filePath, content, "", nil)
	}
	tree, err := i.parser.Parse(content, language)
	if err != nil {
		log.Warn().
			Err(err).
			Str("file", filePath).
			Str("language", language).
			Msg("Failed to parse file, falling back to whole file chunking")
		return i.chunker.ChunkFile(filePath, content, language, nil)
	}

	chunks, err := i.chunker.ChunkFile(filePath, content, language, tree)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk file %s: %w", filePath, err)
	}

	return chunks, nil
}
