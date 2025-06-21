package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/types"
)

// memoryStorage is an in-memory implementation of the Storage interface
type memoryStorage struct {
	mu       sync.RWMutex
	objects  map[string][]byte // key: "bucket/key"
	metadata metadata.Service
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage(meta metadata.Service) Storage {
	return &memoryStorage{
		objects:  make(map[string][]byte),
		metadata: meta,
	}
}

func (s *memoryStorage) key(bucket, key string) string {
	return bucket + "/" + key
}

func (s *memoryStorage) PutObject(ctx context.Context, bucket, key string, data []byte, opts *types.PutObjectOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.metadata.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return ErrBucketNotFound
	}

	objKey := s.key(bucket, key)
	s.objects[objKey] = data

	now := time.Now()
	obj := &types.Object{
		Key:         key,
		Bucket:      bucket,
		ContentType: opts.ContentType,
		Metadata:    opts.Metadata,
		Size:        int64(len(data)),
		CreatedAt:   now,
		ModifiedAt:  now,
	}

	return s.metadata.PutObjectMetadata(ctx, obj)
}

func (s *memoryStorage) GetObject(ctx context.Context, bucket, key string, opts *types.GetObjectOptions) (*types.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, err := s.metadata.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, ErrObjectNotFound
	}

	data, exists := s.objects[s.key(bucket, key)]
	if !exists {
		return nil, ErrObjectNotFound
	}

	content := make([]byte, len(data))
	copy(content, data)

	result := *meta
	result.Content = content
	return &result, nil
}

func (s *memoryStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.metadata.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return err
	}

	delete(s.objects, s.key(bucket, key))
	return s.metadata.DeleteObjectMetadata(ctx, bucket, key)
}

func (s *memoryStorage) ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	return s.metadata.ListObjectMetadata(ctx, bucket, prefix)
}

func (s *memoryStorage) CreateBucket(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.metadata.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if exists {
		return fmt.Errorf("bucket already exists")
	}

	return s.metadata.CreateBucketMetadata(ctx, name)
}

func (s *memoryStorage) DeleteBucket(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.metadata.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket does not exist")
	}

	objects, err := s.metadata.ListObjectMetadata(ctx, name, "")
	if err != nil {
		return fmt.Errorf("failed to list objects in bucket: %w", err)
	}
	if len(objects) > 0 {
		return fmt.Errorf("bucket is not empty")
	}

	return s.metadata.DeleteBucketMetadata(ctx, name)
}

func (s *memoryStorage) ListBuckets(ctx context.Context) ([]string, error) {
	return s.metadata.ListBucketsMetadata(ctx)
}

func (s *memoryStorage) Ping(ctx context.Context) error {
	return s.metadata.Ping(ctx)
}

// Common errors
var (
	ErrObjectNotFound = &Error{"object not found"}
	ErrBucketNotFound = &Error{"bucket not found"}
)

// Error represents a storage error
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
