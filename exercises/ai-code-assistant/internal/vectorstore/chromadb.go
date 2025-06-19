package vectorstore

import (
	"context"
	"fmt"
	"log/slog"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/collection"
	"github.com/amikos-tech/chroma-go/types"
)

// ChromaClient is a wrapper around the ChromaDB client
type ChromaClient struct {
	client *chromago.Client
	logger *slog.Logger
}

// NewChromaClient creates a new ChromaDB client
func NewChromaClient(host string, port int, logger *slog.Logger) (*ChromaClient, error) {
	url := fmt.Sprintf("http://%s:%d", host, port)
	client, err := chromago.NewClient(chromago.WithBasePath(url))
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	return &ChromaClient{
		client: client,
		logger: logger,
	}, nil
}

// CreateCollection creates a new collection in ChromaDB
func (c *ChromaClient) CreateCollection(ctx context.Context, name string) (*chromago.Collection, error) {
	collection, err := c.client.NewCollection(
		ctx,
		name,
		collection.WithHNSWDistanceFunction(types.L2),
		collection.WithCreateIfNotExist(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get collection: %w", err)
	}

	c.logger.Info("Created or retrieved collection", "name", name)
	return collection, nil
}

// AddDocuments adds documents to a collection
func (c *ChromaClient) AddDocuments(ctx context.Context, collectionName string, documents []string, ids []string, metadatas []map[string]interface{}) error {
	collection, err := c.client.GetCollection(ctx, collectionName, nil)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	chromaMetadatas := make([]map[string]interface{}, len(metadatas))
	copy(chromaMetadatas, metadatas)
	_, err = collection.Add(
		ctx,
		nil, // embeddings (nil means Chroma will compute them)
		chromaMetadatas,
		documents,
		ids,
	)
	if err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	c.logger.Info("Added documents to collection", "collection", collectionName, "count", len(documents))
	return nil
}

// Query performs a similarity search on the collection
func (c *ChromaClient) Query(ctx context.Context, collectionName string, query string, nResults int) ([]map[string]interface{}, error) {
	collection, err := c.client.GetCollection(ctx, collectionName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	nResults32 := int32(nResults)

	results, err := collection.Query(
		ctx,
		[]string{query}, // query texts
		nResults32,      // n results
		nil,             // where filter
		nil,             // where document filter
		[]types.QueryEnum{
			"documents",
			"metadatas",
			"distances",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results to a more usable format
	var docs []map[string]interface{}
	if len(results.Documents) > 0 {
		for i := 0; i < len(results.Documents[0]); i++ {
			doc := map[string]interface{}{
				"document": results.Documents[0][i],
			}

			if len(results.Ids) > 0 && len(results.Ids[0]) > i {
				doc["id"] = results.Ids[0][i]
			}

			if len(results.Distances) > 0 && len(results.Distances[0]) > i {
				doc["distance"] = results.Distances[0][i]
			}

			if len(results.Metadatas) > 0 && len(results.Metadatas[0]) > i && results.Metadatas[0][i] != nil {
				doc["metadata"] = results.Metadatas[0][i]
			}

			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Close closes the ChromaDB client
func (c *ChromaClient) Close() error {
	// The underlying HTTP client doesn't need explicit cleanup
	return nil
}
