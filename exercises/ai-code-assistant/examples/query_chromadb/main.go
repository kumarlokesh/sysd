package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/vectorstore"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	client, err := vectorstore.NewChromaClient("localhost", 8000, logger)
	if err != nil {
		log.Fatalf("Failed to create ChromaDB client: %v", err)
	}
	defer client.Close()

	collectionName := "code_chunks"
	queries := []string{
		"hello world program",
		"function main",
		"package main",
		"fmt.Println",
	}

	for _, query := range queries {
		fmt.Printf("\n=== Query: %q ===\n", query)

		results, err := client.Query(context.Background(), collectionName, query, 3)
		if err != nil {
			log.Printf("Error querying for %q: %v", query, err)
			continue
		}

		if len(results) == 0 {
			fmt.Println("No results found")
			continue
		}

		for i, result := range results {
			fmt.Printf("\nResult %d:\n", i+1)

			// Extract and display the document content
			doc, _ := result["document"].(string)
			fmt.Printf("Code:\n%s\n", formatCodeSnippet(doc))

			// Display metadata
			if metadata, ok := result["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
				fmt.Println("\nMetadata:")
				// Display important metadata fields
				if filePath, ok := metadata["file_path"].(string); ok {
					fmt.Printf("  File: %s\n", filePath)
				}
				if language, ok := metadata["language"].(string); ok {
					fmt.Printf("  Language: %s\n", language)
				}
				if nodeType, ok := metadata["node_type"].(string); ok {
					fmt.Printf("  Node Type: %s\n", nodeType)
				}
				if startLine, ok := metadata["start_line"].(float64); ok {
					fmt.Printf("  Lines: %.0f", startLine)
					if endLine, ok := metadata["end_line"].(float64); ok && endLine > startLine {
						fmt.Printf("-%.0f", endLine)
					}
					fmt.Println()
				}
				if chunkIdx, ok := metadata["chunk_index"].(float64); ok {
					totalChunks, _ := metadata["total_chunks"].(float64)
					fmt.Printf("  Chunk: %.0f/%.0f\n", chunkIdx+1, totalChunks)
				}
				if createdAt, ok := metadata["created_at"].(string); ok {
					fmt.Printf("  Indexed at: %s\n", createdAt)
				}
			}

			// Display similarity score (distance)
			if distance, ok := result["distance"].(float64); ok {
				fmt.Printf("Similarity: %.2f\n", 1.0-distance) // Convert distance to similarity score
			}

			fmt.Println("\n" + strings.Repeat("-", 80))
		}
	}
}

// formatCodeSnippet formats code with line numbers for better readability
func formatCodeSnippet(code string) string {
	lines := strings.Split(code, "\n")
	var builder strings.Builder
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			builder.WriteString("\n")
			continue
		}
		builder.WriteString(fmt.Sprintf("%3d | %s\n", i+1, line))
	}
	return builder.String()
}
