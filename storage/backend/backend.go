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

// MOTICE: FileEntry needs a better place.

// FileEntry defines a single cache item.
type FileEntry struct {
	Path         string
	Size         int64
	LastModified time.Time
}

// Backend implements operations for caching files.
type Backend interface {
	// TODO: Can we have a io.Writer or io.WriterAt
	// Get(ctx context.Context, p string, io.Writer) error
	// Get TODO
	Get(ctx context.Context, p string) (io.ReadCloser, error)

	// Put TODO
	Put(ctx context.Context, p string, r io.Reader) error

	// Implement me!
	// List(ctx context.Context, p string) ([]FileEntry, error)

	// Implement me!
	// Delete(ctx context.Context, p string) error
}
