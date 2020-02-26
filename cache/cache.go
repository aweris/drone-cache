// Package cache provides functionality for cache storage
package cache

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/internal/metadata"
	"github.com/meltwater/drone-cache/key"
	"github.com/meltwater/drone-cache/storage"
)

// Cache defines Cache functionality and stores configuration.
type Cache interface {
	// Push(src, dst string) error
	// Pull(src, dst string) error

	// Rebuild TODO
	Rebuild(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error

	// Restore TODO
	Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error
}

type cache struct {
	logger log.Logger

	a archive.Archive
	s storage.Storage
	g key.Generator
}

// New creates a new cache with given parameters.
func New(logger log.Logger, s storage.Storage, a archive.Archive, g key.Generator) *cache {
	return &cache{
		logger: log.With(logger, "component", "cache"),
		a:      a,
		s:      s,
		g:      g,
	}
}

func (c *cache) Rebuild(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error {
	level.Info(c.logger).Log("msg", "rebuilding cache")

	// TODO: Do this for each mounted path!!
	// TODO: Abstract!
	pr, pw := io.Pipe()
	defer pr.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer pw.Close()
		defer close(done)

		if err := c.a.Create(src, pw); err != nil {
			pr.CloseWithError(err) // TODO: Wrap
		}
	}()

	// WriteCloser? Make sure not exit before writer finishes!!
	if err := s.Put(dst, pr); err != nil {
		pw.CloseWithError(err) // TODO: Wrap
		return err
	}

	<-done

	return nil
}

func (c *cache) Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error {
	level.Info(c.logger).Log("msg", "restoring  cache")

	// TODO: Do this for each mounted path!!
	// TODO: Abstract!
	pr, pw := io.Pipe()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer pw.Close()
		defer close(done)

		if err := c.s.Get(src, pw); err != nil {
			pr.CloseWithError(err) // TODO: Wrap
		}
	}()

	// Make sure not exit before writer finishes!!
	if err := c.a.Extract("", pr); err != nil {
		pw.CloseWithError(err) // TODO: Wrap
		return err
	}

	<-done

	return nil
}

// processRebuild the remote cache from the local environment
func processRebuild(l log.Logger, c cache.Cache, g key.Generator, cacheKeyTmpl string, m metadata.Metadata, mountedDirs []string) error {
	now := time.Now()
	branch := m.Commit.Branch

	for _, mount := range mountedDirs {
		if _, err := os.Stat(mount); err != nil {
			return fmt.Errorf("mount <%s>, make sure file or directory exists and readable %w", mount, err)
		}

		key, err := g.Generate(cacheKeyTmpl, mount, branch)
		if err != nil {
			return fmt.Errorf("generate cache key %w", err)
		}

		path := filepath.Join(m.Repo.Name, key)

		level.Info(l).Log("msg", "rebuilding cache for directory", "local", mount, "remote", path)

		if err := c.Push(mount, path); err != nil {
			return fmt.Errorf("upload %w", err)
		}
	}

	level.Info(l).Log("msg", "cache built", "took", time.Since(now))

	return nil
}

// push pushes the archived file to the cache.
func (c *cache) push(src, dst string) error {
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

	if err := c.s.Put(dst, file); err != nil {
		return fmt.Errorf("upload file %w", err)
	}

	return nil
}

// processRestore the local environment from the remote cache.
func processRestore(l log.Logger, c cache.Cache, g key.Generator, cacheKeyTmpl string, m metadata.Metadata, mountedDirs []string) error {
	now := time.Now()
	branch := m.Commit.Branch

	for _, mount := range mountedDirs {
		key, err := g.Generate(cacheKeyTmpl, mount, branch)
		if err != nil {
			return fmt.Errorf("generate cache key %w", err)
		}

		path := filepath.Join(m.Repo.Name, key)
		level.Info(l).Log("msg", "restoring directory", "local", mount, "remote", path)

		if err := c.Pull(path, mount); err != nil {
			return fmt.Errorf("download %w", err)
		}
	}

	level.Info(l).Log("msg", "cache restored", "took", time.Since(now))

	return nil
}

// pull fetches the archived file from the cache and restores to the host machine's file system.
func (c *cache) pull(src, dst string) error {
	level.Info(c.logger).Log("msg", "downloading archived directory", "src", src)

	// 1. download archive
	rc, err := c.s.Get(src, io.Writer) // TODO: !!
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
