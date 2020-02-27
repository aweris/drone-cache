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
	"github.com/go-kit/kit/log/level"
)

// Storage is a place that files can be written to and read from.
type Storage interface {
	// Get writes contents of the given object with given key from remote storage to io.Writer.
	Get(p string, dst io.Writer) error

	// Put writes contents of io.Reader to remote storage at given key location.
	Put(p string, src io.Reader) error

	// List lists contents of the given directory by given key from remote storage.
	List(p string) ([]backend.FileEntry, error)

	// Delete deletes the object from remote storage.
	Delete(p string) error
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
		level.Warn(l).Log("msg", "using azure blob as backend")
		b, err = azure.New(log.With(l, "backend", backend.Azure), configs.Azure)
	case backend.S3:
		level.Warn(l).Log("msg", "using aws s3 as backend")
		b, err = s3.New(log.With(l, "backend", backend.S3), configs.S3, configs.Debug)
	case backend.GCS:
		level.Warn(l).Log("msg", "using gc storage as backend")
		b, err = gcs.New(log.With(l, "backend", backend.GCS), configs.GCS)
	case backend.FileSystem:
		level.Warn(l).Log("msg", "using filesystem as backend")
		b, err = filesystem.New(log.With(l, "backend", backend.FileSystem), configs.FileSystem)
	case backend.SFTP:
		level.Warn(l).Log("msg", "using sftp as backend")
		b, err = sftp.New(log.With(l, "backend", backend.SFTP), configs.SFTP)
	default:
		return nil, errors.New("unknown backend")
	}

	if err != nil {
		return nil, fmt.Errorf("initialize backend %w", err)
	}

	// TODO: Parametric timeout value from CLI.
	// With defaults!
	return newStorage(b, 30*time.Second), nil
}

// Default Storage implementation.
type storage struct {
	b       backend.Backend
	timeout time.Duration
}

// newStorage create a new deafult storage.
func newStorage(b backend.Backend, timeout time.Duration) *storage {
	return &storage{b, timeout}
}

// Get writes contents of the given object with given key from remote storage to io.Writer.
func (s *storage) Get(p string, dst io.Writer) error {
	// TODO: Make sure consumer utilizes context.
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	rc, err := s.b.Get(ctx, p)
	if err != nil {
		return fmt.Errorf("storage backend, Get %w", err)
	}
	defer rc.Close()

	_, err = io.Copy(dst, rc)
	if err != nil {
		return fmt.Errorf("storage backend, Copy %w", err)
	}

	return nil
}

// Put writes contents of io.Reader to remote storage at given key location.
func (s *storage) Put(p string, src io.Reader) error {
	// TODO: Make sure consumer utilizes context.
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	return s.b.Put(ctx, p, src)
}

// List lists contents of the given directory by given key from remote storage.
func (s *storage) List(p string) ([]backend.FileEntry, error) {
	// TODO: Implement me!
	// TODO: Make sure consumer utilizes context.
	return []backend.FileEntry{}, nil
}

// Delete deletes the object from remote storage.
func (s *storage) Delete(p string) error {
	// TODO: Implement me!
	// TODO: Make sure consumer utilizes context.
	return nil
}
