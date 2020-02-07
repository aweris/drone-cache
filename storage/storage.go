package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/meltwater/drone-cache/storage/backend"
	"github.com/meltwater/drone-cache/storage/backend/azure"
	"github.com/meltwater/drone-cache/storage/backend/filesystem"
	"github.com/meltwater/drone-cache/storage/backend/gcs"
	"github.com/meltwater/drone-cache/storage/backend/s3"
	"github.com/meltwater/drone-cache/storage/backend/sftp"

	"github.com/go-kit/kit/log"
)

// FileEntry defines a single cache item.
type FileEntry struct {
	Path         string
	Size         int64
	LastModified time.Time
}

// Storage is a place that files can be written to and read from.
type Storage interface {
	// Get writes contents of the given object with given key from remote storage to io.Writer.
	Get(p string, dst io.Writer) error
	// Put writes contents of io.Reader to remote storage at given key location.
	Put(p string, src io.Reader) error
	// List lists contents of the given directory by given key from remote storage.
	List(p string) ([]FileEntry, error)
	// Delete deletes the object from remote storage.
	Delete(p string) error
}

type storage struct {
	b       backend.Backend
	timeout time.Duration
}

func newStorage(b backend.Backend) *storage {
	// TODO: Parametric timeout value.
	return &storage{b, 30 * time.Second}
}

func (s *storage) Get(p string, dst io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	rc, err := s.b.Get(ctx, p)
	if err != nil {
		return fmt.Errorf("storage backend, Get: %w", err)
	}
	defer rc.Close()

	_, err = io.Copy(dst, rc)
	if err != nil {
		return fmt.Errorf("storage backend, Get: %w", err)
	}

	return nil
}

func (s *storage) Put(p string, src io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.b.Put(ctx, p, src)
}

//nolint:godox
func (s *storage) List(p string) ([]FileEntry, error) {
	// TODO: Implement!
	return []FileEntry{}, nil
}

//nolint:godox
func (s *storage) Delete(p string) error {
	// TODO: Implement!
	return nil
}

// FromConfig creates new Storage by initializing corresponding backend using given configuration.
func FromConfig(l log.Logger, backedType string, cfgs ...backend.Config) (Storage, error) {
	configs := backend.Configs{}
	for _, c := range cfgs {
		c.Apply(&configs)
	}

	var b backend.Backend
	var err error
	switch backedType {
	case backend.Azure:
		b, err = azure.New(l, configs)
	case backend.S3:
		b, err = s3.New(l, configs)
	case backend.GCS:
		b, err = gcs.New(l, configs)
	case backend.FileSystem:
		b, err = filesystem.New(l, configs)
	case backend.SFTP:
		b, err = sftp.New(l, configs)
	default:
		return nil, errors.New("unknown backend")
	}

	if err != nil {
		return nil, fmt.Errorf("initialize backend: %w", err)
	}

	return newStorage(b), nil
}
