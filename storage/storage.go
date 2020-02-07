package storage

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/meltwater/drone-cache/internal/plugin"
)

// FileEntry defines a single cache item.
type FileEntry struct {
	Path         string
	Size         int64
	LastModified time.Time
}

// Storage is a place that files can be written to and read from.
type Storage interface {
	Get(p string, dst io.Writer) error
	Put(p string, src io.Reader) error
	List(p string) ([]FileEntry, error)
	Delete(p string) error
}

// Backend implements operations for caching files.
type Backend interface {
	Get(ctx context.Context, p string) (io.ReadCloser, error)
	Put(ctx context.Context, p string, io.ReadSeeker) error
	List(ctx context.Context, p string) ([]FileEntry, error)
	Delete(ctx context.Context, p string) error
}

// FromConfig initializes corresponding backend using given configuration.
func FromConfig(l log.Logger, cfg plugin.Config) (storage.Backend, error) {
	switch cfg.Backend {
	case Azure:
		return InitializeAzureBackend(l, cfg.Azure, cfg.Debug)
	case S3:
		return InitializeS3Backend(l, cfg.S3, cfg.Debug)
	case GCS:
		return InitializeGCSBackend(l, cfg.GCS, cfg.Debug)
	case FileSystem:
		return InitializeFileSystemBackend(l, cfg.FileSystem, cfg.Debug)
	case SFTP:
		return InitializeSFTPBackend(l, cfg.SFTP, cfg.Debug)
	default:
		return nil, errors.New("unknown backend")
	}
}

// TODO: Generic configuration for storage

// Config
type Config interface {
	Get(p string, dst io.Writer) error
	Put(p string, src io.Reader) error
	List(p string) ([]FileEntry, error)
	Delete(p string) error
}

type storage struct {
	b storage.Backend
}

func (s *storage) Get(p string, dst io.Writer) error {
	return nil
}

func (s *storage) Put(p string, src io.Reader) error {
	return nil
}

func (s *storage) List(p string) ([]FileEntry, error) {
	return nil
}

func (s *storage) Delete(p string) error {
	return nil
}
