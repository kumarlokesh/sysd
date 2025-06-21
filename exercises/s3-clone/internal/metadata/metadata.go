package metadata

import (
	"context"
	"github.com/kumarlokesh/s3-clone/internal/types"
)

// Service defines the interface for metadata operations
type Service interface {
	// Object metadata operations
	PutObjectMetadata(ctx context.Context, obj *types.Object) error
	GetObjectMetadata(ctx context.Context, bucket, key string) (*types.Object, error)
	DeleteObjectMetadata(ctx context.Context, bucket, key string) error
	ListObjectMetadata(ctx context.Context, bucket, prefix string) ([]types.Object, error)

	// Bucket metadata operations
	CreateBucketMetadata(ctx context.Context, bucket string) error
	DeleteBucketMetadata(ctx context.Context, bucket string) error
	ListBucketsMetadata(ctx context.Context) ([]string, error)
	BucketExists(ctx context.Context, bucket string) (bool, error)

	// Health check
	Ping(ctx context.Context) error
}

// In-memory implementation of the metadata service
// This is a simple implementation for development and testing
// In a production system, this would use a distributed key-value store
// like etcd, Redis, or a database

type inMemoryMetadata struct {
	buckets map[string]struct{}
	objects map[string]types.Object // key: "bucket/key"
}

// NewInMemoryMetadata creates a new in-memory metadata service
func NewInMemoryMetadata() Service {
	return &inMemoryMetadata{
		buckets: make(map[string]struct{}),
		objects: make(map[string]types.Object),
	}
}

func (m *inMemoryMetadata) PutObjectMetadata(ctx context.Context, obj *types.Object) error {
	key := obj.Bucket + "/" + obj.Key
	m.objects[key] = *obj
	return nil
}

func (m *inMemoryMetadata) GetObjectMetadata(ctx context.Context, bucket, key string) (*types.Object, error) {
	obj, exists := m.objects[bucket+"/"+key]
	if !exists {
		return nil, nil
	}
	return &obj, nil
}

func (m *inMemoryMetadata) DeleteObjectMetadata(ctx context.Context, bucket, key string) error {
	delete(m.objects, bucket+"/"+key)
	return nil
}

func (m *inMemoryMetadata) ListObjectMetadata(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	var result []types.Object
	for _, obj := range m.objects {
		if obj.Bucket == bucket {
			if prefix == "" || (len(obj.Key) >= len(prefix) && obj.Key[:len(prefix)] == prefix) {
				result = append(result, obj)
			}
		}
	}
	return result, nil
}

func (m *inMemoryMetadata) CreateBucketMetadata(ctx context.Context, bucket string) error {
	m.buckets[bucket] = struct{}{}
	return nil
}

func (m *inMemoryMetadata) DeleteBucketMetadata(ctx context.Context, bucket string) error {
	delete(m.buckets, bucket)
	// In a real implementation, we would also need to clean up objects
	return nil
}

func (m *inMemoryMetadata) ListBucketsMetadata(ctx context.Context) ([]string, error) {
	buckets := make([]string, 0, len(m.buckets))
	for bucket := range m.buckets {
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func (m *inMemoryMetadata) BucketExists(ctx context.Context, bucket string) (bool, error) {
	_, exists := m.buckets[bucket]
	return exists, nil
}

func (m *inMemoryMetadata) Ping(ctx context.Context) error {
	return nil // Always healthy in-memory
}
