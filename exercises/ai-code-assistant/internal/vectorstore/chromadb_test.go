package vectorstore_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/vectorstore"
)

func TestChromaClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	client, err := vectorstore.NewChromaClient("localhost", 8000, logger)
	if err != nil {
		t.Fatalf("Failed to create ChromaDB client: %v", err)
	}
	defer client.Close()

	collectionName := "test_collection"
	_, err = client.CreateCollection(context.Background(), collectionName)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}

	documents := []string{
		"This is a test document about programming in Go.",
		"This is another test document about machine learning.",
		"Go is a statically typed, compiled programming language.",
	}
	ids := []string{"doc1", "doc2", "doc3"}
	metadatas := []map[string]interface{}{
		{"source": "test", "type": "code"},
		{"source": "test", "type": "ml"},
		{"source": "test", "type": "code"},
	}

	err = client.AddDocuments(context.Background(), collectionName, documents, ids, metadatas)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	results, err := client.Query(context.Background(), collectionName, "programming language", 2)
	if err != nil {
		t.Fatalf("Failed to query documents: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result, got none")
	} else {
		t.Logf("Got %d results:", len(results))
		for i, result := range results {
			t.Logf("Result %d: %+v", i+1, result)
		}
	}
}
