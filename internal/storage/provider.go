package storage

import (
	"context"
	"io"
	"time"
)

// StorageProvider defines the interface for file storage operations.
// This abstraction allows us to swap R2 with other providers (S3, GCS) easily.
type StorageProvider interface {
	UploadFile(ctx context.Context, key string, body io.Reader, contentType string) (string, error)
	DeleteFile(ctx context.Context, key string) error
	GetFileURL(key string) string

	// Presigned URL support
	GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (string, error)
	GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	FileExists(ctx context.Context, key string) (bool, error)
}
