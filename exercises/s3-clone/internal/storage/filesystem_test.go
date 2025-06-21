package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/storage"
	"github.com/kumarlokesh/s3-clone/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFilesystemStorage(t *testing.T) (storage.Storage, string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "s3-clone-test-")
	require.NoError(t, err, "Failed to create temp directory")

	metaSvc := metadata.NewInMemoryMetadata()

	store, err := storage.NewFilesystemStorage(tempDir, metaSvc)
	require.NoError(t, err, "Failed to create filesystem storage")

	return store, tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

func TestFilesystemStorage(t *testing.T) {
	t.Run("Create and list buckets", func(t *testing.T) {
		store, _, cleanup := setupFilesystemStorage(t)
		defer cleanup()

		ctx := context.Background()

		err := store.CreateBucket(ctx, "test-bucket")
		require.NoError(t, err)

		buckets, err := store.ListBuckets(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"test-bucket"}, buckets)

		err = store.CreateBucket(ctx, "test-bucket")
		assert.Error(t, err)
	})

	t.Run("Object operations", func(t *testing.T) {
		store, tempDir, cleanup := setupFilesystemStorage(t)
		defer cleanup()

		ctx := context.Background()
		bucket := "test-bucket"
		key := "test-object"
		content := []byte("test content")

		err := store.CreateBucket(ctx, bucket)
		require.NoError(t, err)

		err = store.PutObject(ctx, bucket, key, content, &types.PutObjectOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"key1": "value1"},
		})
		require.NoError(t, err)

		obj, err := store.GetObject(ctx, bucket, key, &types.GetObjectOptions{})
		require.NoError(t, err)
		assert.Equal(t, content, obj.Content)
		assert.Equal(t, "text/plain", obj.ContentType)
		assert.Equal(t, "value1", obj.Metadata["key1"])

		files, err := filepath.Glob(filepath.Join(tempDir, "*", "*", "*", key))
		assert.NoError(t, err)
		assert.Len(t, files, 1)
		objects, err := store.ListObjects(ctx, bucket, "")
		require.NoError(t, err)
		require.Len(t, objects, 1)
		assert.Equal(t, key, objects[0].Key)

		err = store.DeleteObject(ctx, bucket, key)
		require.NoError(t, err)

		_, err = store.GetObject(ctx, bucket, key, &types.GetObjectOptions{})
		assert.Error(t, err)

		files, err = filepath.Glob(filepath.Join(tempDir, "*", "*", "*", key))
		assert.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("Delete bucket", func(t *testing.T) {
		store, _, cleanup := setupFilesystemStorage(t)
		defer cleanup()

		ctx := context.Background()
		bucket := "test-bucket"

		err := store.CreateBucket(ctx, bucket)
		require.NoError(t, err)

		err = store.PutObject(ctx, bucket, "test-object", []byte("content"), &types.PutObjectOptions{})
		require.NoError(t, err)

		err = store.DeleteBucket(ctx, bucket)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket is not empty")

		err = store.DeleteObject(ctx, bucket, "test-object")
		require.NoError(t, err)

		err = store.DeleteBucket(ctx, bucket)
		assert.NoError(t, err)

		buckets, err := store.ListBuckets(ctx)
		require.NoError(t, err)
		assert.Empty(t, buckets)
	})

	t.Run("Ping", func(t *testing.T) {
		store, _, cleanup := setupFilesystemStorage(t)
		defer cleanup()

		err := store.Ping(context.Background())
		assert.NoError(t, err)
	})
}
