package storage_test

import (
	"context"
	"testing"

	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/storage"
	"github.com/kumarlokesh/s3-clone/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage(t *testing.T) {
	metaSvc := metadata.NewInMemoryMetadata()
	store := storage.NewMemoryStorage(metaSvc)
	ctx := context.Background()
	t.Run("Create and list buckets", func(t *testing.T) {
		err := store.CreateBucket(ctx, "test-bucket")
		require.NoError(t, err)

		buckets, err := store.ListBuckets(ctx)
		require.NoError(t, err)
		assert.Len(t, buckets, 1)
		assert.Equal(t, "test-bucket", buckets[0])

		err = store.CreateBucket(ctx, "test-bucket")
		assert.Error(t, err)
	})

	t.Run("Object operations", func(t *testing.T) {
		testData := []byte("test data")
		opts := &types.PutObjectOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"key": "value"},
		}

		err := store.PutObject(ctx, "test-bucket", "test-object", testData, opts)
		require.NoError(t, err)

		obj, err := store.GetObject(ctx, "test-bucket", "test-object", &types.GetObjectOptions{})
		require.NoError(t, err)
		require.NotNil(t, obj)
		assert.Equal(t, "test-object", obj.Key)
		assert.Equal(t, "test-bucket", obj.Bucket)
		assert.Equal(t, "text/plain", obj.ContentType)
		assert.Equal(t, int64(len(testData)), obj.Size)
		assert.Equal(t, "value", obj.Metadata["key"])
		assert.Equal(t, testData, obj.Content)

		objects, err := store.ListObjects(ctx, "test-bucket", "")
		require.NoError(t, err)
		assert.Len(t, objects, 1)
		assert.Equal(t, "test-object", objects[0].Key)

		err = store.DeleteObject(ctx, "test-bucket", "test-object")
		require.NoError(t, err)

		obj, err = store.GetObject(ctx, "test-bucket", "test-object", &types.GetObjectOptions{})
		assert.ErrorIs(t, err, storage.ErrObjectNotFound)
		assert.Nil(t, obj)
	})

	t.Run("Delete bucket", func(t *testing.T) {
		err := store.CreateBucket(ctx, "test-bucket-2")
		require.NoError(t, err)

		err = store.PutObject(ctx, "test-bucket-2", "test-object", []byte("data"), &types.PutObjectOptions{})
		require.NoError(t, err)

		err = store.DeleteBucket(ctx, "test-bucket-2")
		assert.Error(t, err)

		err = store.DeleteObject(ctx, "test-bucket-2", "test-object")
		require.NoError(t, err)

		err = store.DeleteBucket(ctx, "test-bucket-2")
		require.NoError(t, err)

		buckets, err := store.ListBuckets(ctx)
		require.NoError(t, err)
		assert.Len(t, buckets, 1) // Only the first test bucket should remain
	})

	t.Run("Ping", func(t *testing.T) {
		err := store.Ping(ctx)
		require.NoError(t, err)
	})
}
