// Package cache provides functionality for cache storage
package cache

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/cache/archive"
	"github.com/meltwater/drone-cache/storage"
)

// Cache defines Cache functionality and stores configuration.
type Cache interface {
	Push(src, dst string) error
	Pull(src, dst string) error
}

type cache struct {
	logger log.Logger

	a archive.Archive
	b storage.Backend
}

// New creates a new cache with given parameters.
func New(logger log.Logger, b storage.Backend, a archive.Archive) *cache {
	return &cache{
		logger: log.With(logger, "component", "cache"),
		a:      a,
		b:      b,
	}
}

// Push pushes the archived file to the cache.
func (c *cache) Push(src, dst string) error {
	// 1. check if source is reachable.
	src, err := filepath.Abs(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("read source directory %w", err)
	}

	level.Info(c.logger).Log("msg", "archiving directory", "src", src)

	// 2. create temp file
	file, err := ioutil.TempFile("", "archive-*.tar")
	if err != nil {
		return fmt.Errorf("create tarball file <%s> %w", file.Name(), err)
	}
	defer file.Close()

	// 3. write files in the src to the archive.
	written, err := c.a.Create(src, file)
	if err != nil {
		return fmt.Errorf("archive write to %w", err)
	}

	// 4. get written file stats.
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("archive file stat %w", err)
	}

	level.Debug(c.logger).Log(
		"msg", "archive created",
		"raw size", written,
		"compressed size", stat.Size(),
		"compression ratio %", written/stat.Size()*100,
	)

	level.Info(c.logger).Log("msg", "uploading archived directory", "src", src, "dst", dst)

	if err := c.b.Put(dst, file); err != nil {
		return fmt.Errorf("upload file %w", err)
	}

	return nil
}

// Pull fetches the archived file from the cache and restores to the host machine's file system.
func (c *cache) Pull(src, dst string) error {
	level.Info(c.logger).Log("msg", "downloading archived directory", "src", src)

	// 1. download archive
	rc, err := c.b.Get(src)
	if err != nil {
		return fmt.Errorf("get file from storage backend %w", err)
	}
	defer rc.Close()

	level.Info(c.logger).Log("msg", "extracting archived directory", "src", src, "dst", dst)
	// 2. extract archive
	written, err := c.a.Extract(dst, rc)
	if err != nil {
		return fmt.Errorf("extract files from downloaded archive %w", err)
	}

	level.Debug(c.logger).Log(
		"msg", "archive extracted",
		"raw size", written,
	)

	return nil
}
