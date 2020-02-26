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

// FileEntry defines a single cache item.
type FileEntry struct {
	Path         string
	Size         int64
	LastModified time.Time
}

// Backend implements operations for caching files.
type Backend interface {
	Get(ctx context.Context, p string) (io.ReadCloser, error)
	Put(ctx context.Context, p string, rs io.Reader) error
	// TODO: Implement!
	// List(ctx context.Context, p string) ([]FileEntry, error)
	// Delete(ctx context.Context, p string) error
}
