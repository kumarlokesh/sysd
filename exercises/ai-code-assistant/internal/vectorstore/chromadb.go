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
	c.logger.Info("Starting collection creation/retrieval", "collection_name", name)

	exists := false
	collections, err := c.client.ListCollections(ctx)
	if err != nil {
		c.logger.Warn("Failed to list collections, will try to create anyway",
			"error", err)
	} else {
		for _, col := range collections {
			if col.Name == name {
				exists = true
				c.logger.Debug("Collection already exists",
					"name", name,
					"id", col.ID)
				break
			}
		}
	}

	c.logger.Debug("Creating or getting collection",
		"name", name,
		"exists", exists)

	collection, err := c.client.NewCollection(
		ctx,
		name,
		collection.WithHNSWDistanceFunction(types.L2),
		collection.WithCreateIfNotExist(true),
	)
	if err != nil {
		c.logger.Error("Failed to create or get collection",
			"name", name,
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return nil, fmt.Errorf("failed to create or get collection: %w", err)
	}

	if collection == nil {
		c.logger.Error("Created collection is nil")
		return nil, fmt.Errorf("created collection is nil")
	}

	c.logger.Info("Successfully created or retrieved collection",
		"name", name,
		"collection_id", collection.ID,
		"was_existing", exists)

	c.logger.Debug("Collection details",
		"name", collection.Name,
		"id", collection.ID,
		"metadata", collection.Metadata)

	return collection, nil
}

// AddDocuments adds documents to a collection
func (c *ChromaClient) AddDocuments(ctx context.Context, collectionName string, documents []string, ids []string, metadatas []map[string]interface{}) error {
	c.logger.Info("Starting to add documents to collection",
		"collection", collectionName,
		"document_count", len(documents))

	if len(documents) == 0 {
		c.logger.Warn("No documents to add to collection", "collection", collectionName)
		return nil
	}

	if len(documents) != len(ids) || len(documents) != len(metadatas) {
		err := fmt.Errorf("mismatched slice lengths: documents=%d, ids=%d, metadatas=%d",
			len(documents), len(ids), len(metadatas))
		c.logger.Error("Invalid arguments to AddDocuments", "error", err)
		return fmt.Errorf("invalid arguments: %w", err)
	}

	c.logger.Debug("Getting collection for adding documents",
		"collection", collectionName)

	collections, listErr := c.client.ListCollections(ctx)
	if listErr != nil {
		c.logger.Warn("Failed to list collections", "error", listErr)
	} else {
		c.logger.Debug("Available collections",
			"count", len(collections),
			"collections", collections)
	}

	collection, err := c.client.GetCollection(ctx, collectionName, nil)
	if err != nil {
		c.logger.Error("Failed to get collection for adding documents",
			"collection", collectionName,
			"error", err,
			"error_type", fmt.Sprintf("%T", err))

		c.logger.Info("Attempting to create collection that doesn't exist",
			"collection", collectionName)

		var createErr error
		collection, createErr = c.CreateCollection(ctx, collectionName)
		if createErr != nil {
			c.logger.Error("Failed to create collection",
				"collection", collectionName,
				"error", createErr)
			return fmt.Errorf("failed to create collection: %w", createErr)
		}
	}

	c.logger.Debug("Preparing to add documents to collection",
		"collection", collectionName,
		"count", len(documents))

	chromaMetadatas := make([]map[string]interface{}, len(metadatas))
	copy(chromaMetadatas, metadatas)

	logCount := 3
	if len(ids) < logCount {
		logCount = len(ids)
	}
	for i := 0; i < logCount; i++ {
		c.logger.Debug("Sample document being added",
			"id", ids[i],
			"document_preview", truncateString(documents[i], 100)+"...")
	}
	if len(ids) > logCount {
		c.logger.Debug("Additional documents not logged", "count", len(ids)-logCount)
	}

	_, err = collection.Add(
		ctx,
		nil, // embeddings (nil means Chroma will compute them)
		chromaMetadatas,
		documents,
		ids,
	)
	if err != nil {
		c.logger.Error("Failed to add documents to collection",
			"collection", collectionName,
			"error", err,
			"document_count", len(documents))
		return fmt.Errorf("failed to add documents: %w", err)
	}

	c.logger.Info("Successfully added documents to collection",
		"collection", collectionName,
		"count", len(documents))
	return nil
}

// truncateString shortens a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
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
