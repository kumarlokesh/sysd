package types

import (
	"time"
)

// Object represents a stored object in our S3 clone
type Object struct {
	Key         string            `json:"key"`
	Bucket      string            `json:"bucket"`
	Content     []byte            `json:"content,omitempty"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
	Size        int64             `json:"size"`
	CreatedAt   time.Time         `json:"created_at"`
	ModifiedAt  time.Time         `json:"modified_at"`
}

// Bucket represents a container for objects
type Bucket struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ObjectListing represents a list of objects in a bucket
type ObjectListing struct {
	Bucket  string   `json:"bucket"`
	Prefix  string   `json:"prefix"`
	Objects []Object `json:"objects"`
}

// PutObjectOptions contains optional parameters for PutObject
type PutObjectOptions struct {
	ContentType string
	Metadata    map[string]string
}

// GetObjectOptions contains optional parameters for GetObject
type GetObjectOptions struct {
	// Future: Add range, versioning, etc.
}

// ListObjectsOptions contains optional parameters for listing objects
type ListObjectsOptions struct {
	Prefix string
	// Future: Add delimiter, max keys, etc.
}
