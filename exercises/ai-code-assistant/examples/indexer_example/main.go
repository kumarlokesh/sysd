package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/indexer"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger
	if len(os.Args) < 2 {
		log.Fatal().Msg("Please provide a file or directory path to index")
	}
	path := os.Args[1]

	log.Info().Str("path", path).Msg("Starting to index path")

	idx := indexer.NewDefaultIndexer()

	supportedLangs := idx.GetSupportedLanguages()
	log.Info().Strs("languages", supportedLangs).Msg("Supported languages")

	log.Info().Msg("Starting indexing...")
	chunks, err := idx.IndexPath(path)
	if err != nil {
		log.Fatal().Err(err).Str("path", path).Msg("Failed to index path")
	}

	log.Info().Int("chunks", len(chunks)).Msg("Finished indexing")

	fmt.Printf("Found %d chunks in %s\n\n", len(chunks), path)

	for i, chunk := range chunks {
		fmt.Printf("Chunk %d (%s):\n", i+1, chunk.NodeType)
		fmt.Printf("  File: %s:%d-%d\n", chunk.FilePath, chunk.StartLine, chunk.EndLine)
		fmt.Printf("  Language: %s\n", chunk.Language)

		preview := chunk.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("  Preview: %s\n", preview)
		fmt.Println()
	}

	if len(chunks) > 0 {
		jsonData, err := json.MarshalIndent(chunks[0], "", "  ")
		if err != nil {
			log.Printf("Failed to marshal chunk to JSON: %v", err)
		} else {
			fmt.Println("First chunk as JSON:")
			fmt.Println(string(jsonData))
		}
	}
}
