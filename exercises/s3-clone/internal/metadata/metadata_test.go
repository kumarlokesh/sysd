package metadata_test

import (
	"context"
	"testing"
	"time"

	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryMetadata(t *testing.T) {
	svc := metadata.NewInMemoryMetadata()
	ctx := context.Background()

	t.Run("Create and list buckets", func(t *testing.T) {
		err := svc.CreateBucketMetadata(ctx, "test-bucket")
		require.NoError(t, err)

		buckets, err := svc.ListBucketsMetadata(ctx)
		require.NoError(t, err)
		assert.Len(t, buckets, 1)
		assert.Equal(t, "test-bucket", buckets[0])

		exists, err := svc.BucketExists(ctx, "test-bucket")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = svc.BucketExists(ctx, "non-existent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Object operations", func(t *testing.T) {
		now := time.Now()
		obj := &types.Object{
			Key:         "test-object",
			Bucket:      "test-bucket",
			ContentType: "text/plain",
			Size:        123,
			CreatedAt:   now,
			ModifiedAt:  now,
			Metadata:    map[string]string{"key": "value"},
		}

		err := svc.PutObjectMetadata(ctx, obj)
		require.NoError(t, err)

		gotObj, err := svc.GetObjectMetadata(ctx, "test-bucket", "test-object")
		require.NoError(t, err)
		require.NotNil(t, gotObj)
		assert.Equal(t, obj.Key, gotObj.Key)
		assert.Equal(t, obj.Bucket, gotObj.Bucket)
		assert.Equal(t, obj.ContentType, gotObj.ContentType)
		assert.Equal(t, obj.Size, gotObj.Size)
		assert.Equal(t, obj.Metadata, gotObj.Metadata)

		objects, err := svc.ListObjectMetadata(ctx, "test-bucket", "")
		require.NoError(t, err)
		require.Len(t, objects, 1)
		assert.Equal(t, "test-object", objects[0].Key)

		objects, err = svc.ListObjectMetadata(ctx, "test-bucket", "test-")
		require.NoError(t, err)
		assert.Len(t, objects, 1)

		objects, err = svc.ListObjectMetadata(ctx, "test-bucket", "non-matching")
		require.NoError(t, err)
		assert.Empty(t, objects)

		err = svc.DeleteObjectMetadata(ctx, "test-bucket", "test-object")
		require.NoError(t, err)

		gotObj, err = svc.GetObjectMetadata(ctx, "test-bucket", "test-object")
		require.NoError(t, err)
		assert.Nil(t, gotObj)
	})

	t.Run("Delete bucket", func(t *testing.T) {
		err := svc.DeleteBucketMetadata(ctx, "test-bucket")
		require.NoError(t, err)

		exists, err := svc.BucketExists(ctx, "test-bucket")
		require.NoError(t, err)
		assert.False(t, exists)

		buckets, err := svc.ListBucketsMetadata(ctx)
		require.NoError(t, err)
		assert.Empty(t, buckets)
	})

	t.Run("Ping", func(t *testing.T) {
		err := svc.Ping(ctx)
		require.NoError(t, err)
	})
}
