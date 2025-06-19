package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/indexer"
	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/storage"
	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/vectorstore"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger

	if len(os.Args) < 2 {
		log.Fatal().Msg("Usage: indexer_example <path_to_index> [chroma_url]")
	}

	path := os.Args[1]
	chromaURL := "http://localhost:8000"
	if len(os.Args) > 2 {
		chromaURL = os.Args[2]
	}

	log.Info().
		Str("path", path).
		Str("chroma_url", chromaURL).
		Msg("Starting to index path")

	// Initialize ChromaDB storage
	// For simplicity, we'll use localhost:8000 as the default ChromaDB URL
	chromaHost := "localhost"
	chromaPort := 8000
	chromaClient, err := vectorstore.NewChromaClient(chromaHost, chromaPort, slog.Default())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create ChromaDB client")
	}

	// Create storage implementation
	collectionName := "code_chunks"
	storageImpl := vectorstore.NewChromaStore(chromaClient, collectionName, slog.Default())

	// Initialize indexer with storage
	idx := indexer.NewDefaultIndexer(storageImpl,
		indexer.WithLogger(slog.Default()),
		indexer.WithWorkerCount(4),
	)

	supportedLangs := idx.GetSupportedLanguages()
	log.Info().Strs("languages", supportedLangs).Msg("Supported languages")

	log.Info().Msg("Starting indexing...")

	// Start indexing
	startTime := time.Now()
	err = idx.IndexPath(ctx, path)
	if err != nil {
		log.Fatal().Err(err).Str("path", path).Msg("Failed to index path")
	}

	duration := time.Since(startTime)
	log.Info().
		Str("duration", duration.String()).
		Msg("Finished indexing")

	// Example of searching the indexed content
	if len(os.Args) > 3 && os.Args[3] == "--search" {
		searchQuery := "function definition"
		if len(os.Args) > 4 {
			searchQuery = os.Args[4]
		}
		searchIndexedContent(ctx, storageImpl, searchQuery)
	}
}

// searchIndexedContent demonstrates how to search the indexed content
func searchIndexedContent(ctx context.Context, store storage.Storage, query string) {
	log.Info().Str("query", query).Msg("Searching indexed content")

	results, err := store.Search(ctx, query, 5) // Get top 5 results
	if err != nil {
		log.Error().Err(err).Msg("Failed to search indexed content")
		return
	}

	if len(results) == 0 {
		log.Info().Msg("No results found")
		return
	}

	fmt.Printf("\nFound %d results for '%s':\n\n", len(results), query)
	for i, result := range results {
		fmt.Printf("Result %d (Score: %.2f):\n", i+1, result.Score)
		fmt.Printf("  File: %s\n", result.Chunk.FilePath)
		fmt.Printf("  Lines: %d-%d\n", result.Chunk.StartLine, result.Chunk.EndLine)
		fmt.Printf("  Node: %s\n", result.Chunk.NodeType)

		// Show preview of the content
		preview := result.Chunk.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("  Preview: %s\n\n", preview)
	}
}
