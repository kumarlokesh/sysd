package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kumarlokesh/s3-clone/internal/api"
	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/storage"
	"github.com/kumarlokesh/s3-clone/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	metaSvc := metadata.NewInMemoryMetadata()
	store := storage.NewMemoryStorage(metaSvc)
	server := api.NewServer(":0", store)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	client := testServer.Client()

	t.Run("Bucket operations", func(t *testing.T) {
		bucketName := "test-bucket"

		t.Run("Create bucket", func(t *testing.T) {
			req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s", testServer.URL, bucketName), nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			buckets, err := store.ListBuckets(context.Background())
			require.NoError(t, err)
			assert.Contains(t, buckets, bucketName)
		})

		t.Run("List objects in bucket", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/%s", testServer.URL, bucketName))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result struct {
				Bucket  string   `json:"bucket"`
				Objects []string `json:"objects"`
			}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Equal(t, bucketName, result.Bucket)
			assert.Empty(t, result.Objects) // Should be empty right after bucket creation
		})

		t.Run("Put object", func(t *testing.T) {
			objectKey := "test-object"
			objectData := []byte("test data")
			req, err := http.NewRequest(
				"PUT",
				fmt.Sprintf("%s/%s/%s", testServer.URL, bucketName, objectKey),
				bytes.NewReader(objectData),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "text/plain")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			obj, err := store.GetObject(context.Background(), bucketName, objectKey, &types.GetObjectOptions{})
			require.NoError(t, err)
			assert.Equal(t, objectData, obj.Content)
		})

		t.Run("Get object", func(t *testing.T) {
			objectKey := "test-object"
			resp, err := http.Get(fmt.Sprintf("%s/%s/%s", testServer.URL, bucketName, objectKey))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "test data", string(data))
		})

		t.Run("List objects", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/%s", testServer.URL, bucketName))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var result struct {
				Bucket  string   `json:"bucket"`
				Prefix  string   `json:"prefix"`
				Objects []string `json:"objects"`
			}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)
			assert.Contains(t, result.Objects, "test-object")
		})

		t.Run("Delete object", func(t *testing.T) {
			objectKey := "test-object"
			req, err := http.NewRequest(
				"DELETE",
				fmt.Sprintf("%s/%s/%s", testServer.URL, bucketName, objectKey),
				nil,
			)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			_, err = store.GetObject(context.Background(), bucketName, objectKey, &types.GetObjectOptions{})
			assert.Error(t, err)
		})

		t.Run("Delete bucket", func(t *testing.T) {
			req, err := http.NewRequest(
				"DELETE",
				fmt.Sprintf("%s/%s", testServer.URL, bucketName),
				nil,
			)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			buckets, err := store.ListBuckets(context.Background())
			require.NoError(t, err)
			assert.NotContains(t, buckets, bucketName)
		})
	})
}
