package cache

import (
	"fmt"
	"time"

	"github.com/meltwater/drone-cache/storage"
	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Flusher TODO
type Flusher interface {
	// Flush TODO
	Flush(src string) error
}

// Flush TODO
func (c *cache) Flush(src string) error {
	return nil
}

type flusher struct {
	logger log.Logger

	store storage.Storage
	dirty func(backend.FileEntry) bool
}

// NewFlusher creates a new cache flusher.
func NewFlusher(s storage.Storage, ttl time.Duration) Flusher {
	return &flusher{store: s, dirty: IsExpired(ttl)}
}

// Flush cleans the expired files from the cache.
func (f *flusher) Flush(src string) error {
	level.Info(f.logger).Log("msg", "Cleaning files", "src", src)

	files, err := f.store.List(src)
	if err != nil {
		return fmt.Errorf("flusher list %w", err)
	}

	for _, file := range files {
		if f.dirty(file) {
			err := f.store.Delete(file.Path)
			if err != nil {
				return fmt.Errorf("flusher delete %w", err)
			}
		}
	}

	return nil
}

// IsExpired creates a function to check if file expired.
func IsExpired(ttl time.Duration) func(file backend.FileEntry) bool {
	return func(file backend.FileEntry) bool {
		return time.Now().After(file.LastModified.Add(ttl))
	}
}
