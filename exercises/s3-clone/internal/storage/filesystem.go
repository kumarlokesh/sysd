package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/types"
)

// filesystemStorage is a filesystem-based implementation of the Storage interface
type filesystemStorage struct {
	mu       sync.RWMutex
	rootDir  string
	metadata metadata.Service
}

// NewFilesystemStorage creates a new filesystem-based storage instance
func NewFilesystemStorage(rootDir string, metaSvc metadata.Service) (Storage, error) {
	// Create the root directory if it doesn't exist
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &filesystemStorage{
		rootDir:  rootDir,
		metadata: metaSvc,
	}, nil
}

// bucketPath returns the filesystem path for a bucket
func (s *filesystemStorage) bucketPath(bucket string) string {
	// Use a hash of the bucket name to ensure valid directory names
	h := sha256.Sum256([]byte(bucket))
	hash := hex.EncodeToString(h[:])
	return filepath.Join(s.rootDir, hash[:2], hash[2:4], hash)
}

// objectPath returns the filesystem path for an object
func (s *filesystemStorage) objectPath(bucket, key string) string {
	bucketPath := s.bucketPath(bucket)
	// Replace path separators in the key to prevent directory traversal
	safeKey := strings.ReplaceAll(key, string(filepath.Separator), "_")
	return filepath.Join(bucketPath, safeKey)
}

// CreateBucket creates a new bucket
func (s *filesystemStorage) CreateBucket(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.metadata.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if exists {
		return fmt.Errorf("bucket already exists")
	}

	bucketPath := s.bucketPath(name)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return fmt.Errorf("failed to create bucket directory: %w", err)
	}

	return s.metadata.CreateBucketMetadata(ctx, name)
}

// DeleteBucket deletes a bucket
func (s *filesystemStorage) DeleteBucket(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	objects, err := s.metadata.ListObjectMetadata(ctx, name, "")
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	if len(objects) > 0 {
		return fmt.Errorf("bucket is not empty")
	}

	bucketPath := s.bucketPath(name)
	if err := os.RemoveAll(bucketPath); err != nil {
		return fmt.Errorf("failed to remove bucket directory: %w", err)
	}

	return s.metadata.DeleteBucketMetadata(ctx, name)
}

// ListBuckets lists all buckets
func (s *filesystemStorage) ListBuckets(ctx context.Context) ([]string, error) {
	return s.metadata.ListBucketsMetadata(ctx)
}

// PutObject stores an object in the bucket
func (s *filesystemStorage) PutObject(ctx context.Context, bucket, key string, data []byte, opts *types.PutObjectOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exists, err := s.metadata.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket does not exist")
	}

	objectPath := s.objectPath(bucket, key)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}

	if err := os.WriteFile(objectPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write object data: %w", err)
	}
	obj := &types.Object{
		Key:         key,
		Bucket:      bucket,
		Content:     data,
		ContentType: opts.ContentType,
		Metadata:    opts.Metadata,
	}

	return s.metadata.PutObjectMetadata(ctx, obj)
}

// GetObject retrieves an object from the bucket
func (s *filesystemStorage) GetObject(ctx context.Context, bucket, key string, opts *types.GetObjectOptions) (*types.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, err := s.metadata.GetObjectMetadata(ctx, bucket, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}
	objectPath := s.objectPath(bucket, key)
	data, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	obj.Content = data
	return obj, nil
}

// DeleteObject deletes an object from the bucket
func (s *filesystemStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	objectPath := s.objectPath(bucket, key)
	if err := os.Remove(objectPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete object file: %w", err)
	}

	return s.metadata.DeleteObjectMetadata(ctx, bucket, key)
}

// ListObjects lists objects in a bucket with the given prefix
func (s *filesystemStorage) ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	return s.metadata.ListObjectMetadata(ctx, bucket, prefix)
}

// Ping checks if the storage backend is accessible
func (s *filesystemStorage) Ping(ctx context.Context) error {
	if err := os.MkdirAll(s.rootDir, 0755); err != nil {
		return fmt.Errorf("failed to access root directory: %w", err)
	}

	return s.metadata.Ping(ctx)
}
