package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const defaultFileMode = 0755

// Filesystem is an file system implementation of the Backend.
type Filesystem struct {
	cacheRoot string
}

// New creates a Filesystem backend.
func New(l log.Logger, c Config) (*Filesystem, error) {
	if strings.TrimRight(path.Clean(c.CacheRoot), "/") == "" {
		return nil, fmt.Errorf("empty or root path given, <%s> as cache root, ", c.CacheRoot)
	}

	if _, err := os.Stat(c.CacheRoot); err != nil {
		return nil, fmt.Errorf("make sure volume is mounted, <%s> as cache root %w", c.CacheRoot, err)
	}

	level.Debug(l).Log("msg", "Filesystem backend", "config", fmt.Sprintf("%#v", c))

	return &Filesystem{cacheRoot: c.CacheRoot}, nil
}

// Get writes downloaded content to the given writer.
func (b *Filesystem) Get(ctx context.Context, p string, w io.Writer) error {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(b.cacheRoot, p)))
	if err != nil {
		return fmt.Errorf("absolute path %w", err)
	}

	errCh := make(chan error)

	go func() {
		defer close(errCh)

		rc, err := os.Open(absPath)
		if err != nil {
			errCh <- fmt.Errorf("get the object %w", err)
		}
		defer rc.Close()

		_, err = io.Copy(w, rc)
		if err != nil {
			errCh <- fmt.Errorf("copy the object %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Put uploads contents of the given reader.
func (b *Filesystem) Put(ctx context.Context, p string, r io.Reader) error {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(b.cacheRoot, p)))
	if err != nil {
		return fmt.Errorf("build path %w", err)
	}

	errCh := make(chan error)

	go func() {
		defer close(errCh)

		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, os.FileMode(defaultFileMode)); err != nil {
			errCh <- fmt.Errorf("create directory <%s> %w", dir, err)
		}

		w, err := os.Create(absPath)
		if err != nil {
			errCh <- fmt.Errorf("create cache file <%s> %w", absPath, err)
		}
		defer w.Close()

		if _, err := io.Copy(w, r); err != nil {
			errCh <- fmt.Errorf("write contents of reader to a file %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
