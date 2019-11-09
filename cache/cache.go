// Package cache provides functionality for cache storage
package cache

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/cache/archive"
)

// Backend implements operations for caching files.
type Backend interface {
	Get(string) (io.ReadCloser, error)
	Put(string, io.ReadSeeker) error
}

// Cache contains configuration for Cache functionality.
type Cache struct {
	logger log.Logger

	b    Backend
	opts options
}

// New creates a new cache with given parameters.
func New(logger log.Logger, b Backend, opts ...Option) Cache {
	options := options{
		archiveFmt:       DefaultArchiveFormat,
		compressionLevel: DefaultCompressionLevel,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	return Cache{
		logger: log.With(logger, "component", "cache"),
		b:      b,
		opts:   options,
	}
}

// Push pushes the archived file to the cache.
func (c Cache) Push(src, dst string) error {
	// 1. check if source is reachable.
	src, err := filepath.Abs(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("read source directory %w", err)
	}

	level.Info(c.logger).Log("msg", "archiving directory", "src", src)

	// 2. create a temporary file for the archive.
	if err := os.MkdirAll("/tmp", os.FileMode(0755)); err != nil {
		return fmt.Errorf("create tmp directory %w", err)
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("create tmp folder for archive %w", err)
	}

	archivePath := filepath.Join(dir, "archive.tar")

	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create tarball file <%s> %w", archivePath, err)
	}

	// 3. write files in the src to the archive.
	archiveWriter := archive.NewWriter(src, c.opts.archiveFmt, c.opts.compressionLevel, c.opts.skipSymlinks)

	written, err := archiveWriter.WriteTo(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("archive write to %w", err)
	}

	// 4. get written file stats.
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("archive file stat %w", err)
	}

	level.Debug(c.logger).Log(
		"msg", "archive created",
		"archive format", c.opts.archiveFmt,
		"compression level", c.opts.compressionLevel,
		"raw size", written,
		"compressed size", stat.Size(),
		"compression ratio %", written/stat.Size()*100,
	)

	// 5. close resources before upload.
	if err := archiveWriter.Close(); err != nil {
		return fmt.Errorf("archive writer close %w", err)
	}
	// file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync archived file %w", err)
	}

	// 6. upload archive file to server.
	level.Info(c.logger).Log("msg", "uploading archived directory", "src", src, "dst", dst)

	// TODO: TEST !!!
	// f, err := os.Open(archivePath)
	// if err != nil {
	// 	return fmt.Errorf("open archived file to send %w", err)
	// }
	// defer f.Close()

	if err := c.b.Put(dst, file); err != nil {
		return fmt.Errorf("upload file %w", err)
	}

	return nil
}

// Pull fetches the archived file from the cache and restores to the host machine's file system.
func (c Cache) Pull(src, dst string) error {
	level.Info(c.logger).Log("msg", "downloading archived directory", "src", src)
	// 1. download archive
	rc, err := c.b.Get(src)
	if err != nil {
		return fmt.Errorf("get file from storage backend %w", err)
	}
	defer rc.Close()

	// 2. extract archive
	level.Info(c.logger).Log("msg", "extracting archived directory", "src", src, "dst", dst)

	extractor := archive.NewExtractor(c.opts.archiveFmt)
	defer extractor.Close()

	written, err := extractor.ExtractFrom(rc)
	if err != nil {
		return fmt.Errorf("extract files from downloaded archive %w", err)
	}

	level.Debug(c.logger).Log(
		"msg", "archive extracted",
		"archive format", c.opts.archiveFmt,
		"raw size", written,
	)

	return nil
}
