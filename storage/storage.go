package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/go-kit/kit/log"
)

const DefaultOperationTimeout = 30 * time.Second

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

// Default Storage implementation.
type storage struct {
	logger log.Logger

	b       backend.Backend
	timeout time.Duration
}

// New create a new default storage.
func New(l log.Logger, b backend.Backend, timeout time.Duration) Storage {
	return &storage{l, b, timeout}
}

// Get writes contents of the given object with given key from remote storage to io.Writer.
func (s *storage) Get(p string, dst io.Writer) error {
	// TODO: Rethink backend.Get(ctx context.Context, p string, io.Writer) error
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
	// Implement me!
	// Make sure consumer utilizes context.
	return []backend.FileEntry{}, nil
}

// Delete deletes the object from remote storage.
func (s *storage) Delete(p string) error {
	// Implement me!
	// Make sure consumer utilizes context.
	return nil
}
