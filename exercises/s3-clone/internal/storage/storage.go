package storage

import (
	"context"
	"github.com/kumarlokesh/s3-clone/internal/types"
)

// Storage defines the interface for object storage operations
type Storage interface {
	// Object operations
	PutObject(ctx context.Context, bucket, key string, data []byte, opts *types.PutObjectOptions) error
	GetObject(ctx context.Context, bucket, key string, opts *types.GetObjectOptions) (*types.Object, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error)

	// Bucket operations
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	ListBuckets(ctx context.Context) ([]string, error)

	// Health check
	Ping(ctx context.Context) error
}
