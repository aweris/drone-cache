package backend

import (
	"context"
	"io"
	"time"
)

const (
	FileSystem = "filesystem"
	S3         = "s3"
	SFTP       = "sftp"
	Azure      = "azure"
	GCS        = "gcs"
)

// TODO: Find a better place.

// FileEntry defines a single cache item.
type FileEntry struct {
	Path         string
	Size         int64
	LastModified time.Time
}

// Backend implements operations for caching files.
type Backend interface {
	// Get TODO
	Get(ctx context.Context, p string) (io.ReadCloser, error)

	// Put TODO
	Put(ctx context.Context, p string, rs io.Reader) error

	// TODO: Implement!
	// List(ctx context.Context, p string) ([]FileEntry, error)

	// TODO: Implement!
	// Delete(ctx context.Context, p string) error
}
