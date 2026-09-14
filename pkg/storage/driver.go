package storage

import (
	"context"
	"io"
	"time"
)

// Driver defines the contract for pluggable file storage providers (Local Disk, S3, MinIO, etc.).
type Driver interface {
	// Put writes a file stream to the specified path/key.
	Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error

	// Get opens a read stream for the file at the specified path/key.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file at the specified path/key.
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists at the specified path/key.
	Exists(ctx context.Context, path string) (bool, error)

	// GetURL returns a public or temporary signed URL for the file if supported.
	GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}
