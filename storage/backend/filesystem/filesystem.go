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

// TODO: Utilize context!

// filesystem is an file system implementation of the Backend.
type filesystem struct {
	cacheRoot string
}

// New creates a filesystem backend.
func New(l log.Logger, c Config) (*filesystem, error) {
	if strings.TrimRight(path.Clean(c.CacheRoot), "/") == "" {
		return nil, fmt.Errorf("empty or root path given, <%s> as cache root, ", c.CacheRoot)
	}

	if _, err := os.Stat(c.CacheRoot); err != nil {
		return nil, fmt.Errorf("make sure volume is mounted, <%s> as cache root %w", c.CacheRoot, err)
	}

	level.Debug(l).Log("msg", "filesystem backend", "config", fmt.Sprintf("%#v", c))

	return &filesystem{cacheRoot: c.CacheRoot}, nil
}

// Get returns an io.Reader for reading the contents of the file.
func (c *filesystem) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(c.cacheRoot, p)))
	if err != nil {
		return nil, fmt.Errorf("get the object %w", err)
	}

	return os.Open(absPath)
}

// Put uploads the contents of the io.Reader.
func (c *filesystem) Put(ctx context.Context, p string, src io.Reader) error {
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(c.cacheRoot, p)))
	if err != nil {
		return fmt.Errorf("build path %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, os.FileMode(defaultFileMode)); err != nil {
		return fmt.Errorf("create directory <%s> %w", dir, err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("create cache file <%s> %w", absPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write contents of reader to a file %w", err)
	}

	return nil
}
